import { useCallback, useEffect, useState } from "react";
import { Button } from "@documi/ui";
import type {
  OrganizationDashboard,
  OrganizationRecord,
  SessionContext,
  ReportDefinition,
  ReportRun,
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
  const load = useCallback(async () => {
    setLoading(true);
    setError(undefined);
    try {
      const [items, stats] = await Promise.all([
        gateway.platformOrganizations(),
        gateway.platformDashboard(),
      ]);
      setOrganizations(items);
      setDashboard(stats);
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
      const run = await gateway.createReportRun(
        organizationId,
        definition.id,
        formats[definition.id] ?? "pdf",
        {},
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
const integrations = [
  {
    name: "Native Documi Editor",
    state: "Installed",
    detail: "Browser, desktop, offline, and air-gapped.",
  },
  {
    name: "Collabora Online",
    state: "Setup required",
    detail: "Trusted browser Office editor through authenticated WOPI.",
  },
  {
    name: "LibreOffice Desktop",
    state: "Desktop only",
    detail: "Detected and installed by explicit local user action.",
  },
  {
    name: "Google Workspace",
    state: "Setup required",
    detail: "External Docs, Sheets, and Slides. Blocked when air-gapped.",
  },
];
function IntegrationCenter() {
  return (
    <>
      <h1 className="m-0 text-4xl font-semibold tracking-[-0.04em]">Trusted integrations</h1>
      <p className="mt-2 text-sm text-black/50">
        First-party adapters are compile-time registered. Documi never loads arbitrary provider
        code.
      </p>
      <div className="mt-6 space-y-3">
        {integrations.map((item) => (
          <article
            key={item.name}
            className="flex flex-col justify-between gap-4 rounded-2xl border border-black/[0.07] bg-white p-5 shadow-soft sm:flex-row sm:items-center"
          >
            <div>
              <h2 className="m-0 text-base font-semibold">{item.name}</h2>
              <p className="mb-0 mt-1 text-sm text-black/50">{item.detail}</p>
            </div>
            <span className="w-fit rounded-full bg-paper px-3 py-1 text-xs font-semibold text-black/55">
              {item.state}
            </span>
          </article>
        ))}
      </div>
      <div className="mt-5 rounded-xl border border-amber-200 bg-amber-50 p-4 text-sm text-amber-900">
        <strong>Credentials are private.</strong> OAuth and service credentials are write-only and
        are never shared with organization members.
      </div>
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
  if (section === "integrations") return <IntegrationCenter />;
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
