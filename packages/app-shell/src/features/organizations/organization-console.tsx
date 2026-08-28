import { useCallback, useEffect, useState } from "react";
import { Button } from "@documi/ui";
import type {
  OrganizationDashboard,
  OrganizationRecord,
  SessionContext,
  ReportDefinition,
  ReportRun,
  OrganizationMember,
  OrganizationInvitation,
  AuthorizationRole,
  IntegrationStatus,
  IntegrationConnection,
  PlatformDocumentMetadata,
  DocumentRecord,
} from "@documi/shared-types";
import type { OrganizationGateway } from "../documents/editor-platform";

function Metric({
  label,
  value,
  detail,
}: {
  label: string;
  value: string | number;
  detail?: string;
}) {
  return (
    <article className="rounded-2xl border border-black/[0.07] bg-white p-5 shadow-soft">
      <p className="m-0 text-xs font-bold uppercase tracking-[0.16em] text-black/40">{label}</p>
      <p className="mb-0 mt-3 text-3xl font-semibold tracking-[-0.04em]">{value}</p>
      {detail && <p className="mb-0 mt-2 text-xs text-black/45">{detail}</p>}
    </article>
  );
}
function State({
  loading,
  error,
  children,
}: {
  loading: boolean;
  error?: string;
  children: React.ReactNode;
}) {
  if (loading)
    return (
      <div className="rounded-2xl border border-black/[0.07] bg-white p-10 text-sm text-black/50">
        Loading authorized dashboard…
      </div>
    );
  if (error)
    return (
      <div
        role="alert"
        className="rounded-2xl border border-red-200 bg-red-50 p-5 text-sm text-red-800"
      >
        {error}
      </div>
    );
  return <>{children}</>;
}
export function PlatformConsole({
  gateway,
  organizationsOnly = false,
}: {
  gateway: OrganizationGateway;
  organizationsOnly?: boolean;
}) {
  const [organizations, setOrganizations] = useState<OrganizationRecord[]>([]);
  const [dashboard, setDashboard] = useState<OrganizationDashboard>();
  const [error, setError] = useState<string>();
  const [loading, setLoading] = useState(true);
  const [creating, setCreating] = useState(false);
  const [inviteURL, setInviteURL] = useState<string>();
  const [integrationStatuses, setIntegrationStatuses] = useState<IntegrationStatus[]>([]);
  const [inspectedDocuments, setInspectedDocuments] = useState<PlatformDocumentMetadata[]>();
  const [inspectedOrganization, setInspectedOrganization] = useState<string>();
  const load = useCallback(async () => {
    setLoading(true);
    setError(undefined);
    try {
      const [items, stats, integrations] = await Promise.all([
        gateway.platformOrganizations(),
        gateway.platformDashboard(),
        gateway.integrations(),
      ]);
      setOrganizations(items);
      setDashboard(stats);
      setIntegrationStatuses(integrations);
    } catch (e) {
      setError(e instanceof Error ? e.message : "Unable to load platform administration");
    } finally {
      setLoading(false);
    }
  }, [gateway]);
  useEffect(() => {
    void load();
  }, [load]);
  async function create(event: React.FormEvent<HTMLFormElement>) {
    event.preventDefault();
    const data = new FormData(event.currentTarget);
    setCreating(true);
    setError(undefined);
    try {
      const result = await gateway.createOrganization(
        String(data.get("name")),
        String(data.get("ownerEmail")),
      );
      setInviteURL(result.ownerInvitation?.acceptanceUrl);
      event.currentTarget.reset();
      await load();
    } catch (e) {
      setError(e instanceof Error ? e.message : "Unable to create organization");
    } finally {
      setCreating(false);
    }
  }
  async function toggle(item: OrganizationRecord) {
    try {
      await gateway.updateOrganization(item.id, {
        status: item.status === "suspended" ? "active" : "suspended",
      });
      await load();
    } catch (e) {
      setError(e instanceof Error ? e.message : "Unable to update organization");
    }
  }
  return (
    <State loading={loading} error={error}>
      <header className="mb-7">
        <p className="mb-2 text-xs font-bold uppercase tracking-[0.2em] text-forest">
          Platform administration
        </p>
        <h1 className="m-0 text-4xl font-semibold tracking-[-0.04em]">
          {organizationsOnly ? "Organizations" : "Deployment overview"}
        </h1>
        <p className="mt-2 text-sm text-black/50">
          Platform authority can inspect bounded metadata but never document bytes.
        </p>
      </header>
      {!organizationsOnly && dashboard && (
        <section className="mb-6 grid gap-4 sm:grid-cols-2 xl:grid-cols-4">
          <Metric label="Organizations" value={dashboard.organizations ?? 0} />
          <Metric label="Active members" value={dashboard.members} />
          <Metric label="Documents" value={dashboard.documents} />
          <Metric
            label="Managed storage"
            value={`${(dashboard.storageBytes / 1048576).toFixed(1)} MiB`}
          />
          <Metric label="Versions" value={dashboard.versions} />
          <Metric label="Suspended" value={dashboard.suspendedOrganizations ?? 0} />
          <Metric label="Report failures" value={dashboard.reportFailures ?? 0} />
          <Metric label="Provider errors" value={dashboard.providerErrors ?? 0} />
        </section>
      )}
      {!organizationsOnly && (
        <section className="mb-6 rounded-2xl border border-black/[0.07] bg-white p-5 shadow-soft">
          <div className="mb-4">
            <h2 className="m-0 text-lg font-semibold">Trusted integration installations</h2>
            <p className="mb-0 mt-1 text-xs text-black/45">
              Global installation policy. Organization connections remain separately owned.
            </p>
          </div>
          <div className="grid gap-3 md:grid-cols-2">
            {integrationStatuses.map((item) => (
              <div
                key={item.id}
                className="flex items-center justify-between rounded-xl border border-black/[0.07] p-4"
              >
                <span>
                  <strong className="block text-sm">{item.name}</strong>
                  <span className="text-xs text-black/45">{item.state.replaceAll("-", " ")}</span>
                </span>
                <button
                  className="mr-3 font-semibold text-forest"
                  onClick={async () => {
                    setInspectedOrganization(item.name);
                    setInspectedDocuments(await gateway.platformDocuments(item.id));
                  }}
                >
                  Inspect metadata
                </button>
                <button
                  disabled={item.id === "native" || item.state === "blocked-by-air-gap"}
                  onClick={async () => {
                    await gateway.setIntegrationEnabled(item.id, item.state === "disabled");
                    await load();
                  }}
                  className="rounded-lg border border-black/10 px-3 py-1.5 text-xs font-semibold disabled:opacity-40"
                >
                  {item.state === "disabled" ? "Enable" : "Disable"}
                </button>
              </div>
            ))}
          </div>
        </section>
      )}
      <section className="grid gap-5 xl:grid-cols-[1fr_320px]">
        <div className="overflow-hidden rounded-2xl border border-black/[0.07] bg-white shadow-soft">
          <div className="border-b border-black/[0.06] px-5 py-4">
            <h2 className="m-0 text-lg font-semibold">Organizations</h2>
          </div>
          <div className="overflow-x-auto">
            <table className="w-full border-collapse text-left text-sm">
              <thead className="bg-paper text-xs uppercase tracking-wider text-black/45">
                <tr>
                  <th className="px-5 py-3">Organization</th>
                  <th className="px-5 py-3">Status</th>
                  <th className="px-5 py-3">Title visibility</th>
                  <th className="px-5 py-3">Action</th>
                </tr>
              </thead>
              <tbody>
                {organizations.map((item) => (
                  <tr key={item.id} className="border-t border-black/[0.05]">
                    <td className="px-5 py-4 font-semibold">
                      {item.name}
                      <span className="block font-mono text-[10px] font-normal text-black/35">
                        {item.id}
                      </span>
                    </td>
                    <td className="px-5 py-4">
                      <span
                        className={`rounded-full px-2.5 py-1 text-xs font-semibold ${item.status === "suspended" ? "bg-red-50 text-red-700" : item.status === "pending" ? "bg-amber-50 text-amber-700" : "bg-moss text-forest"}`}
                      >
                        {item.status}
                      </span>
                    </td>
                    <td className="px-5 py-4">
                      {item.shareDocumentTitlesWithPlatformAdmins
                        ? "Titles shared"
                        : "Metadata only"}
                    </td>
                    <td className="px-5 py-4">
                      <button
                        className="font-semibold text-forest"
                        onClick={() => void toggle(item)}
                      >
                        {item.status === "suspended" ? "Reactivate" : "Suspend"}
                      </button>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
            {organizations.length === 0 && (
              <p className="p-8 text-center text-sm text-black/45">
                No organizations have been created.
              </p>
            )}
          </div>
        </div>
        <form
          onSubmit={create}
          className="h-fit rounded-2xl border border-black/[0.07] bg-white p-5 shadow-soft"
        >
          <h2 className="m-0 text-lg font-semibold">Create organization</h2>
          <p className="mt-1 text-xs leading-5 text-black/45">
            Creates a pending organization, General workspace, and expiring owner invitation
            atomically.
          </p>
          <label className="mt-5 block text-xs font-semibold">
            Organization name
            <input
              required
              name="name"
              maxLength={255}
              className="mt-2 w-full rounded-xl border border-black/10 px-3 py-2.5 text-sm"
            />
          </label>
          <label className="mt-4 block text-xs font-semibold">
            Owner email
            <input
              required
              type="email"
              name="ownerEmail"
              className="mt-2 w-full rounded-xl border border-black/10 px-3 py-2.5 text-sm"
            />
          </label>
          <Button className="mt-5 w-full" disabled={creating}>
            {creating ? "Creating…" : "Create and invite owner"}
          </Button>
          {inviteURL && (
            <div className="mt-4 rounded-xl bg-amber-50 p-3 text-xs text-amber-900">
              <strong>Development invitation URL</strong>
              <input
                readOnly
                value={inviteURL}
                className="mt-2 w-full bg-transparent font-mono text-[10px]"
              />
            </div>
          )}
        </form>
      </section>
      {inspectedDocuments && (
        <section className="mt-5 overflow-hidden rounded-2xl border border-black/[0.07] bg-white shadow-soft">
          <div className="flex items-center justify-between border-b border-black/[0.06] px-5 py-4">
            <div>
              <h2 className="m-0 text-lg font-semibold">
                Document metadata · {inspectedOrganization}
              </h2>
              <p className="mb-0 mt-1 text-xs text-black/45">
                This drill-down was audited. Document bytes are unavailable to platform authority.
              </p>
            </div>
            <button
              onClick={() => setInspectedDocuments(undefined)}
              className="text-xs font-semibold"
            >
              Close
            </button>
          </div>
          <div className="overflow-x-auto">
            <table className="w-full text-left text-sm">
              <thead className="bg-paper text-xs uppercase tracking-wider text-black/45">
                <tr>
                  <th className="px-5 py-3">Document</th>
                  <th className="px-5 py-3">Family</th>
                  <th className="px-5 py-3">Provider</th>
                  <th className="px-5 py-3">Storage</th>
                  <th className="px-5 py-3">Size</th>
                </tr>
              </thead>
              <tbody>
                {inspectedDocuments.map((document) => (
                  <tr key={document.id} className="border-t border-black/[0.06]">
                    <td className="px-5 py-4">
                      <strong className="block">
                        {document.title ?? "Title hidden by organization"}
                      </strong>
                      <span className="font-mono text-[10px] text-black/35">{document.id}</span>
                    </td>
                    <td className="px-5 py-4">{document.family}</td>
                    <td className="px-5 py-4">{document.editorProvider}</td>
                    <td className="px-5 py-4">{document.storageMode}</td>
                    <td className="px-5 py-4">{document.size.toLocaleString()} bytes</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </section>
      )}
    </State>
  );
}

function ReportCenter({
  gateway,
  organizationId,
}: {
  gateway: OrganizationGateway;
  organizationId: string;
}) {
  const [definitions, setDefinitions] = useState<ReportDefinition[]>([]);
  const [runs, setRuns] = useState<ReportRun[]>([]);
  const [formats, setFormats] = useState<Record<string, ReportRun["format"]>>({});
  const [busy, setBusy] = useState<string>();
  const [error, setError] = useState<string>();
  const [filters, setFilters] = useState<Record<string, string>>({});
  const load = useCallback(async () => {
    try {
      const [nextDefinitions, nextRuns] = await Promise.all([
        gateway.reportDefinitions(),
        gateway.reportRuns(organizationId),
      ]);
      setDefinitions(nextDefinitions);
      setRuns(nextRuns);
    } catch (e) {
      setError(e instanceof Error ? e.message : "Unable to load reports");
    }
  }, [gateway, organizationId]);
  useEffect(() => {
    void load();
  }, [load]);
  async function generate(definition: ReportDefinition) {
    setBusy(definition.id);
    setError(undefined);
    try {
      const normalizedFilters = Object.fromEntries(
        Object.entries(filters).filter(([, value]) => value),
      );
      const run = await gateway.createReportRun(
        organizationId,
        definition.id,
        formats[definition.id] ?? "pdf",
        normalizedFilters,
      );
      await gateway.exportReport(run.id);
      await load();
    } catch (e) {
      setError(e instanceof Error ? e.message : "Report generation failed");
    } finally {
      setBusy(undefined);
    }
  }
  return (
    <>
      <h1 className="m-0 text-4xl font-semibold tracking-[-0.04em]">Reports</h1>
      <p className="mt-2 text-sm text-black/50">
        Curated, auditable reports. Exports become managed Documi documents with normal permissions.
      </p>
      {error && (
        <div
          role="alert"
          className="mt-4 rounded-xl border border-red-200 bg-red-50 p-3 text-sm text-red-800"
        >
          {error}
        </div>
      )}
      <section className="mt-5 grid gap-3 rounded-2xl border border-black/[0.07] bg-white p-4 sm:grid-cols-2 lg:grid-cols-4">
        <label className="text-xs font-semibold">
          From
          <input
            type="datetime-local"
            onChange={(event) =>
              setFilters((value) => ({
                ...value,
                dateFrom: event.target.value ? new Date(event.target.value).toISOString() : "",
              }))
            }
            className="mt-2 w-full rounded-lg border border-black/10 px-2 py-2"
          />
        </label>
        <label className="text-xs font-semibold">
          To
          <input
            type="datetime-local"
            onChange={(event) =>
              setFilters((value) => ({
                ...value,
                dateTo: event.target.value ? new Date(event.target.value).toISOString() : "",
              }))
            }
            className="mt-2 w-full rounded-lg border border-black/10 px-2 py-2"
          />
        </label>
        <label className="text-xs font-semibold">
          Document family
          <select
            onChange={(event) => setFilters((value) => ({ ...value, family: event.target.value }))}
            className="mt-2 w-full rounded-lg border border-black/10 px-2 py-2"
          >
            <option value="">All families</option>
            <option value="native">Native</option>
            <option value="text">Text</option>
            <option value="spreadsheet">Spreadsheet</option>
            <option value="presentation">Presentation</option>
            <option value="pdf">PDF</option>
          </select>
        </label>
        <label className="text-xs font-semibold">
          Storage mode
          <select
            onChange={(event) =>
              setFilters((value) => ({ ...value, storageMode: event.target.value }))
            }
            className="mt-2 w-full rounded-lg border border-black/10 px-2 py-2"
          >
            <option value="">All storage</option>
            <option value="managed">Managed</option>
            <option value="external">External</option>
          </select>
        </label>
      </section>
      <div className="mt-6 grid gap-4 md:grid-cols-2">
        {definitions.map((definition) => (
          <article
            key={definition.id}
            className="rounded-2xl border border-black/[0.07] bg-white p-5 shadow-soft"
          >
            <h2 className="m-0 text-base font-semibold">{definition.name}</h2>
            <p className="mt-2 min-h-10 text-sm leading-5 text-black/50">
              {definition.description}
            </p>
            <div className="mt-4 flex gap-2">
              <button
                disabled={busy === definition.id}
                onClick={() => void generate(definition)}
                className="rounded-lg bg-forest px-3 py-2 text-xs font-semibold text-white disabled:opacity-50"
              >
                {busy === definition.id ? "Generating…" : "Generate report"}
              </button>
              <select
                aria-label={`Format for ${definition.name}`}
                value={formats[definition.id] ?? "pdf"}
                onChange={(event) =>
                  setFormats((value) => ({
                    ...value,
                    [definition.id]: event.target.value as ReportRun["format"],
                  }))
                }
                className="rounded-lg border border-black/10 bg-white px-2 text-xs"
              >
                <option>PDF</option>
                <option>XLSX</option>
                <option>CSV</option>
              </select>
            </div>
          </article>
        ))}
      </div>
      <section className="mt-6 overflow-hidden rounded-2xl border border-black/[0.07] bg-white">
        <h2 className="m-0 border-b border-black/[0.06] px-5 py-4 text-lg font-semibold">
          Generated reports
        </h2>
        {runs.length === 0 ? (
          <p className="p-6 text-sm text-black/45">No reports generated yet.</p>
        ) : (
          <ul className="m-0 divide-y divide-black/[0.06] p-0">
            {runs.map((run) => (
              <li
                key={run.id}
                className="flex items-center justify-between gap-4 px-5 py-4 text-sm"
              >
                <span>
                  <strong className="block">
                    {definitions.find((item) => item.id === run.definitionId)?.name ??
                      run.definitionId}
                  </strong>
                  <span className="text-xs text-black/45">
                    {new Date(run.createdAt).toLocaleString()} · {run.format.toUpperCase()}
                  </span>
                </span>
                {run.documentId ? (
                  <span className="rounded-full bg-moss px-3 py-1 text-xs font-semibold text-forest">
                    Managed document
                  </span>
                ) : (
                  <span className="text-xs text-black/45">{run.status}</span>
                )}
              </li>
            ))}
          </ul>
        )}
      </section>
    </>
  );
}
function IntegrationCenter({
  gateway,
  organizationId,
}: {
  gateway: OrganizationGateway;
  organizationId: string;
}) {
  const [integrations, setIntegrations] = useState<IntegrationStatus[]>([]);
  const [connections, setConnections] = useState<IntegrationConnection[]>([]);
  const [error, setError] = useState<string>();
  const [workspaces, setWorkspaces] = useState<SessionContext["workspaces"]>([]);
  const [googleWorkspaceId, setGoogleWorkspaceId] = useState("");
  const [googleConnectionId, setGoogleConnectionId] = useState("");
  const [googleFamily, setGoogleFamily] = useState<"text" | "spreadsheet" | "presentation">("text");
  const [googleTitle, setGoogleTitle] = useState("");
  const [existingGoogleFileId, setExistingGoogleFileId] = useState("");
  const [createdGoogleDocument, setCreatedGoogleDocument] = useState<DocumentRecord>();
  const load = useCallback(async () => {
    try {
      const [statuses, values, session] = await Promise.all([
        gateway.integrations(),
        gateway.connections(organizationId),
        gateway.session(),
      ]);
      setIntegrations(statuses);
      setConnections(values);
      const organizationWorkspaces = session.workspaces.filter(
        (workspace) => workspace.organizationId === organizationId,
      );
      setWorkspaces(organizationWorkspaces);
      setGoogleWorkspaceId((current) => current || organizationWorkspaces[0]?.id || "");
      setGoogleConnectionId(
        (current) =>
          current || values.find((connection) => connection.integrationId === "google-workspace")?.id || "",
      );
    } catch (e) {
      setError(e instanceof Error ? e.message : "Unable to load integrations");
    }
  }, [gateway, organizationId]);
  useEffect(() => {
    void load();
  }, [load]);
  return (
    <>
      <h1 className="m-0 text-4xl font-semibold tracking-[-0.04em]">Trusted integrations</h1>
      <p className="mt-2 text-sm text-black/50">
        First-party adapters are compile-time registered. Documi never loads arbitrary provider
        code.
      </p>
      {error && (
        <div
          role="alert"
          className="mt-4 rounded-xl border border-red-200 bg-red-50 p-3 text-sm text-red-800"
        >
          {error}
        </div>
      )}
      <div className="mt-6 space-y-3">
        {integrations.map((item) => (
          <article
            key={item.name}
            className="flex flex-col justify-between gap-4 rounded-2xl border border-black/[0.07] bg-white p-5 shadow-soft sm:flex-row sm:items-center"
          >
            <div>
              <h2 className="m-0 text-base font-semibold">{item.name}</h2>
              <p className="mb-0 mt-1 text-sm text-black/50">{item.description}</p>
            </div>
            <span className="w-fit rounded-full bg-paper px-3 py-1 text-xs font-semibold text-black/55">
              {item.state.replaceAll("-", " ")}
            </span>
            {item.id === "google-workspace" && item.state === "installed" && (
              <button
                onClick={async () => {
                  const start = await gateway.startGoogleOrganizationConnection(organizationId);
                  window.location.assign(start.authorizationUrl);
                }}
                className="rounded-lg bg-forest px-3 py-2 text-xs font-semibold text-white"
              >
                Connect Google Workspace
              </button>
            )}
          </article>
        ))}
      </div>
      {connections.length > 0 && (
        <section className="mt-6 overflow-hidden rounded-2xl border border-black/[0.07] bg-white">
          <h2 className="m-0 border-b border-black/[0.06] px-5 py-4 text-lg font-semibold">
            Organization connections
          </h2>
          <ul className="m-0 divide-y divide-black/[0.06] p-0">
            {connections.map((connection) => (
              <li
                key={connection.id}
                className="flex items-center justify-between px-5 py-4 text-sm"
              >
                <span>
                  <strong className="block">
                    {integrations.find((item) => item.id === connection.integrationId)?.name ??
                      connection.integrationId}
                  </strong>
                  <span className="text-xs text-black/45">
                    {connection.connectionType} · {connection.status}
                  </span>
                </span>
                <button
                  className="text-xs font-semibold text-red-700"
                  onClick={async () => {
                    if (window.confirm("Disconnect this integration?")) {
                      await gateway.deleteConnection(connection.id);
                      await load();
                    }
                  }}
                >
                  Disconnect
                </button>
              </li>
            ))}
          </ul>
        </section>
      )}
      {connections.some(
        (connection) =>
          connection.integrationId === "google-workspace" && connection.status === "connected",
      ) && (
        <section className="mt-6 rounded-2xl border border-black/[0.07] bg-white p-5 shadow-soft">
          <h2 className="m-0 text-lg font-semibold">Google documents</h2>
          <p className="mt-1 text-sm text-black/50">
            Create in Google or reference a Picker-selected file. Documi stores metadata and
            permissions; content remains in Google until an explicit import.
          </p>
          <div className="mt-4 grid gap-3 md:grid-cols-2">
            <label className="text-sm font-semibold">
              Connection
              <select className="mt-1 w-full rounded-lg border border-black/10 p-2" value={googleConnectionId} onChange={(event) => setGoogleConnectionId(event.target.value)}>
                {connections.filter((connection) => connection.integrationId === "google-workspace" && connection.status === "connected").map((connection) => <option key={connection.id} value={connection.id}>{connection.connectionType}</option>)}
              </select>
            </label>
            <label className="text-sm font-semibold">
              Documi workspace
              <select className="mt-1 w-full rounded-lg border border-black/10 p-2" value={googleWorkspaceId} onChange={(event) => setGoogleWorkspaceId(event.target.value)}>
                {workspaces.map((workspace) => <option key={workspace.id} value={workspace.id}>{workspace.name}</option>)}
              </select>
            </label>
            <label className="text-sm font-semibold">
              Document type
              <select className="mt-1 w-full rounded-lg border border-black/10 p-2" value={googleFamily} onChange={(event) => setGoogleFamily(event.target.value as typeof googleFamily)}>
                <option value="text">Google Docs</option>
                <option value="spreadsheet">Google Sheets</option>
                <option value="presentation">Google Slides</option>
              </select>
            </label>
            <label className="text-sm font-semibold">
              Title
              <input className="mt-1 w-full rounded-lg border border-black/10 p-2" value={googleTitle} onChange={(event) => setGoogleTitle(event.target.value)} placeholder="Quarterly plan" />
            </label>
            <label className="text-sm font-semibold md:col-span-2">
              Existing Google file ID (optional)
              <input className="mt-1 w-full rounded-lg border border-black/10 p-2" value={existingGoogleFileId} onChange={(event) => setExistingGoogleFileId(event.target.value)} placeholder="Paste the ID returned by Google Picker" />
            </label>
          </div>
          <button
            type="button"
            disabled={!googleWorkspaceId || !googleConnectionId || (!existingGoogleFileId && !googleTitle.trim())}
            className="mt-4 rounded-lg bg-forest px-4 py-2 text-sm font-semibold text-white disabled:opacity-50"
            onClick={async () => {
              try {
                const result = await gateway.createGoogleDocument(googleWorkspaceId, {
                  connectionId: googleConnectionId,
                  family: googleFamily,
                  title: googleTitle,
                  externalFileId: existingGoogleFileId || undefined,
                });
                setCreatedGoogleDocument(result);
                setError(undefined);
              } catch (reason) {
                setError(reason instanceof Error ? reason.message : "google_document_failed");
              }
            }}
          >
            {existingGoogleFileId ? "Add external reference" : "Create in Google"}
          </button>
          {createdGoogleDocument?.externalUrl && (
            <p className="mb-0 mt-3 text-sm text-forest">
              External reference created. <a className="font-semibold underline" href={createdGoogleDocument.externalUrl} target="_blank" rel="noreferrer">Open official Google editor</a>
            </p>
          )}
        </section>
      )}
      <div className="mt-5 rounded-xl border border-amber-200 bg-amber-50 p-4 text-sm text-amber-900">
        <strong>Credentials are private.</strong> OAuth and service credentials are write-only and
        are never shared with organization members.
      </div>
    </>
  );
}

const organizationRoles: AuthorizationRole[] = [
  "owner",
  "admin",
  "manager",
  "editor",
  "contributor",
  "viewer",
  "guest",
];
function MembersCenter({
  gateway,
  organizationId,
}: {
  gateway: OrganizationGateway;
  organizationId: string;
}) {
  const [members, setMembers] = useState<OrganizationMember[]>([]);
  const [invitations, setInvitations] = useState<OrganizationInvitation[]>([]);
  const [error, setError] = useState<string>();
  const [busy, setBusy] = useState(false);
  const [inviteURL, setInviteURL] = useState<string>();
  const load = useCallback(async () => {
    try {
      const [nextMembers, nextInvitations] = await Promise.all([
        gateway.members(organizationId),
        gateway.invitations(organizationId),
      ]);
      setMembers(nextMembers);
      setInvitations(nextInvitations);
    } catch (e) {
      setError(e instanceof Error ? e.message : "Unable to load members");
    }
  }, [gateway, organizationId]);
  useEffect(() => {
    void load();
  }, [load]);
  async function invite(event: React.FormEvent<HTMLFormElement>) {
    event.preventDefault();
    const form = event.currentTarget;
    const data = new FormData(form);
    setBusy(true);
    setError(undefined);
    try {
      const value = await gateway.invite(
        organizationId,
        String(data.get("email")),
        String(data.get("role")) as AuthorizationRole,
      );
      setInviteURL(value.acceptanceUrl);
      form.reset();
      await load();
    } catch (e) {
      setError(e instanceof Error ? e.message : "Invitation failed");
    } finally {
      setBusy(false);
    }
  }
  async function changeMember(
    member: OrganizationMember,
    value: { role?: AuthorizationRole; status?: "active" | "suspended" },
  ) {
    setError(undefined);
    try {
      await gateway.updateMember(organizationId, member.userId, value);
      await load();
    } catch (e) {
      setError(e instanceof Error ? e.message : "Member update failed");
    }
  }
  async function remove(member: OrganizationMember) {
    if (!window.confirm(`Remove ${member.displayName} from this organization?`)) return;
    try {
      await gateway.removeMember(organizationId, member.userId);
      await load();
    } catch (e) {
      setError(e instanceof Error ? e.message : "Member removal failed");
    }
  }
  return (
    <>
      <h1 className="m-0 text-4xl font-semibold tracking-[-0.04em]">Members and invitations</h1>
      <p className="mt-2 text-sm text-black/50">
        Each person signs in with an individual identity. Passwords and integration credentials are
        never shared.
      </p>
      {error && (
        <div
          role="alert"
          className="mt-4 rounded-xl border border-red-200 bg-red-50 p-3 text-sm text-red-800"
        >
          {error}
        </div>
      )}
      <section className="mt-6 grid gap-5 xl:grid-cols-[1fr_320px]">
        <div className="overflow-hidden rounded-2xl border border-black/[0.07] bg-white">
          <table className="w-full text-left text-sm">
            <thead className="bg-paper text-xs uppercase tracking-wider text-black/45">
              <tr>
                <th className="px-5 py-3">Member</th>
                <th className="px-5 py-3">Role</th>
                <th className="px-5 py-3">Status</th>
                <th className="px-5 py-3">Actions</th>
              </tr>
            </thead>
            <tbody>
              {members.map((member) => (
                <tr key={member.userId} className="border-t border-black/[0.06]">
                  <td className="px-5 py-4">
                    <strong className="block">{member.displayName}</strong>
                    <span className="text-xs text-black/45">{member.email}</span>
                  </td>
                  <td className="px-5 py-4">
                    <select
                      value={member.role}
                      onChange={(event) =>
                        void changeMember(member, { role: event.target.value as AuthorizationRole })
                      }
                      className="rounded-lg border border-black/10 px-2 py-1.5"
                    >
                      {organizationRoles.map((role) => (
                        <option key={role}>{role}</option>
                      ))}
                    </select>
                  </td>
                  <td className="px-5 py-4">
                    <button
                      onClick={() =>
                        void changeMember(member, {
                          status: member.status === "active" ? "suspended" : "active",
                        })
                      }
                      className={`rounded-full px-2.5 py-1 text-xs font-semibold ${member.status === "active" ? "bg-moss text-forest" : "bg-red-50 text-red-700"}`}
                    >
                      {member.status}
                    </button>
                  </td>
                  <td className="px-5 py-4">
                    <button
                      onClick={() => void remove(member)}
                      className="text-xs font-semibold text-red-700"
                    >
                      Remove
                    </button>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
        <form
          onSubmit={invite}
          className="h-fit rounded-2xl border border-black/[0.07] bg-white p-5"
        >
          <h2 className="m-0 text-lg font-semibold">Invite member</h2>
          <label className="mt-4 block text-xs font-semibold">
            Email
            <input
              required
              type="email"
              name="email"
              className="mt-2 w-full rounded-xl border border-black/10 px-3 py-2.5"
            />
          </label>
          <label className="mt-4 block text-xs font-semibold">
            Initial role
            <select
              name="role"
              defaultValue="viewer"
              className="mt-2 w-full rounded-xl border border-black/10 px-3 py-2.5"
            >
              {organizationRoles.map((role) => (
                <option key={role}>{role}</option>
              ))}
            </select>
          </label>
          <Button className="mt-5 w-full" disabled={busy}>
            {busy ? "Inviting…" : "Send invitation"}
          </Button>
          {inviteURL && (
            <div className="mt-4 rounded-xl bg-amber-50 p-3 text-xs text-amber-900">
              <strong>Development invitation URL</strong>
              <input
                readOnly
                value={inviteURL}
                className="mt-2 w-full bg-transparent font-mono text-[10px]"
              />
            </div>
          )}
        </form>
      </section>
      <section className="mt-6 overflow-hidden rounded-2xl border border-black/[0.07] bg-white">
        <h2 className="m-0 border-b border-black/[0.06] px-5 py-4 text-lg font-semibold">
          Invitations
        </h2>
        {invitations.length === 0 ? (
          <p className="p-5 text-sm text-black/45">No invitations.</p>
        ) : (
          <ul className="m-0 divide-y divide-black/[0.06] p-0">
            {invitations.map((invitation) => (
              <li
                key={invitation.id}
                className="flex flex-wrap items-center justify-between gap-3 px-5 py-4 text-sm"
              >
                <span>
                  <strong className="block">{invitation.email}</strong>
                  <span className="text-xs text-black/45">
                    {invitation.role} · {invitation.status} · expires{" "}
                    {new Date(invitation.expiresAt).toLocaleDateString()}
                  </span>
                  {invitation.notificationError && (
                    <span className="block text-xs text-red-700">
                      Delivery failed—resend after SMTP is fixed.
                    </span>
                  )}
                </span>
                {invitation.status === "pending" && (
                  <span className="flex gap-3">
                    <button
                      onClick={async () => {
                        const value = await gateway.resendInvitation(invitation.id);
                        setInviteURL(value.acceptanceUrl);
                        await load();
                      }}
                      className="text-xs font-semibold text-forest"
                    >
                      Resend
                    </button>
                    <button
                      onClick={async () => {
                        await gateway.revokeInvitation(invitation.id);
                        await load();
                      }}
                      className="text-xs font-semibold text-red-700"
                    >
                      Revoke
                    </button>
                  </span>
                )}
              </li>
            ))}
          </ul>
        )}
      </section>
    </>
  );
}
function SettingsCenter({
  gateway,
  organizationId,
}: {
  gateway: OrganizationGateway;
  organizationId: string;
}) {
  const [settings, setSettings] = useState<OrganizationRecord>();
  const [error, setError] = useState<string>();
  const [saving, setSaving] = useState(false);
  useEffect(() => {
    gateway
      .settings(organizationId)
      .then(setSettings)
      .catch((e) => setError(e instanceof Error ? e.message : "Unable to load settings"));
  }, [gateway, organizationId]);
  async function update(value: boolean) {
    setSaving(true);
    setError(undefined);
    try {
      setSettings(await gateway.updateSettings(organizationId, value));
    } catch (e) {
      setError(e instanceof Error ? e.message : "Unable to update settings");
    } finally {
      setSaving(false);
    }
  }
  return (
    <>
      <h1 className="m-0 text-4xl font-semibold tracking-[-0.04em]">Organization settings</h1>
      <p className="mt-2 text-sm text-black/50">
        Privacy and operational policy controlled by this organization.
      </p>
      {error && (
        <div
          role="alert"
          className="mt-4 rounded-xl border border-red-200 bg-red-50 p-3 text-sm text-red-800"
        >
          {error}
        </div>
      )}
      <section className="mt-6 rounded-2xl border border-black/[0.07] bg-white p-6 shadow-soft">
        <div className="flex flex-col justify-between gap-4 sm:flex-row sm:items-center">
          <div>
            <h2 className="m-0 text-base font-semibold">
              Share document titles with platform administrators
            </h2>
            <p className="mb-0 mt-2 max-w-2xl text-sm leading-6 text-black/50">
              When disabled, platform administrators see only bounded metadata such as document ID,
              family, size, provider, storage mode, timestamps, and status. Platform authority never
              grants access to document bytes.
            </p>
          </div>
          <button
            type="button"
            disabled={!settings || saving}
            role="switch"
            aria-checked={settings?.shareDocumentTitlesWithPlatformAdmins ?? false}
            onClick={() => void update(!(settings?.shareDocumentTitlesWithPlatformAdmins ?? false))}
            className={`relative h-7 w-12 shrink-0 rounded-full transition ${settings?.shareDocumentTitlesWithPlatformAdmins ? "bg-forest" : "bg-black/20"}`}
          >
            <span
              className={`absolute top-1 h-5 w-5 rounded-full bg-white shadow transition ${settings?.shareDocumentTitlesWithPlatformAdmins ? "left-6" : "left-1"}`}
            />
          </button>
        </div>
      </section>
    </>
  );
}

export function OrganizationHome({
  gateway,
  organizationId,
  section,
}: {
  gateway: OrganizationGateway;
  organizationId: string;
  section: string;
}) {
  const [data, setData] = useState<OrganizationDashboard>();
  const [error, setError] = useState<string>();
  useEffect(() => {
    gateway
      .organizationDashboard(organizationId)
      .then(setData)
      .catch((e) => setError(e instanceof Error ? e.message : "Unable to load organization"));
  }, [gateway, organizationId]);
  if (section === "reports")
    return <ReportCenter gateway={gateway} organizationId={organizationId} />;
  if (section === "integrations")
    return <IntegrationCenter gateway={gateway} organizationId={organizationId} />;
  if (section === "members")
    return <MembersCenter gateway={gateway} organizationId={organizationId} />;
  if (section === "settings")
    return <SettingsCenter gateway={gateway} organizationId={organizationId} />;
  if (section !== "dashboard")
    return (
      <section className="rounded-2xl border border-black/[0.07] bg-white p-8">
        <h1 className="m-0 text-3xl font-semibold capitalize">{section}</h1>
        <p className="mt-3 text-sm text-black/50">
          This area uses the organization authority boundary. Administration actions appear only
          when the session grants the required capability.
        </p>
      </section>
    );
  return (
    <State loading={!data && !error} error={error}>
      <h1 className="mb-6 text-4xl font-semibold tracking-[-0.04em]">Organization dashboard</h1>
      {data && (
        <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
          <Metric label="Members" value={data.members} />
          <Metric label="Documents" value={data.documents} />
          <Metric label="Versions" value={data.versions} />
          <Metric label="Storage" value={`${(data.storageBytes / 1048576).toFixed(1)} MiB`} />
          <Metric label="External documents" value={data.externalDocuments} />
          <Metric label="Pending invitations" value={data.pendingInvitations} />
        </div>
      )}
      <nav className="mt-6 flex flex-wrap gap-2">
        <a
          href={`/organizations/${organizationId}/members`}
          className="rounded-xl border border-black/10 bg-white px-4 py-2 text-sm font-semibold"
        >
          Members
        </a>
        <a
          href={`/organizations/${organizationId}/reports`}
          className="rounded-xl border border-black/10 bg-white px-4 py-2 text-sm font-semibold"
        >
          Reports
        </a>
        <a
          href={`/organizations/${organizationId}/integrations`}
          className="rounded-xl border border-black/10 bg-white px-4 py-2 text-sm font-semibold"
        >
          Integrations
        </a>
        <a
          href={`/organizations/${organizationId}/settings`}
          className="rounded-xl border border-black/10 bg-white px-4 py-2 text-sm font-semibold"
        >
          Settings
        </a>
      </nav>
    </State>
  );
}

export function MemberHome({ session }: { session: SessionContext }) {
  return (
    <>
      <p className="mb-2 text-xs font-bold uppercase tracking-[0.2em] text-forest">
        Member dashboard
      </p>
      <h1 className="m-0 text-4xl font-semibold tracking-[-0.04em]">
        Welcome, {session.user.displayName}.
      </h1>
      <p className="mt-3 text-sm text-black/50">
        Your recent and shared documents are constrained by your Documi grants.
      </p>
      <div className="mt-6 grid gap-4 md:grid-cols-3">
        <Metric label="Organizations" value={session.organizations.length} />
        <Metric label="Workspaces" value={session.workspaces.length} />
        <Metric
          label="Connected accounts"
          value={0}
          detail="Connections are personal; credentials are never shared."
        />
      </div>
    </>
  );
}
