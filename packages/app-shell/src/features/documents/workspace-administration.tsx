import type {
  AuditEvent,
  AuthorizationGrant,
  AuthorizationRole,
  DocumentFamily,
  EditorPolicy,
  FolderRecord,
  GroupRecord,
  Workspace,
} from "@documi/shared-types";
import { Button } from "@documi/ui";
import { useCallback, useEffect, useState } from "react";

import type { EditorPlatform } from "./editor-platform";
import { useEditorProviders } from "./use-editor-providers";

const roles: AuthorizationRole[] = [
  "owner",
  "admin",
  "manager",
  "editor",
  "contributor",
  "viewer",
  "guest",
];
const policyFamilies: DocumentFamily[] = ["native", "text", "spreadsheet", "presentation", "pdf"];

export function WorkspaceAdministration({
  platform,
  view,
}: {
  platform: EditorPlatform;
  view: "access" | "audit" | "policies";
}) {
  const gateway = platform.administration;
  const [workspace, setWorkspace] = useState<Workspace>();
  const [folders, setFolders] = useState<FolderRecord[]>([]);
  const [groups, setGroups] = useState<GroupRecord[]>([]);
  const [grants, setGrants] = useState<AuthorizationGrant[]>([]);
  const [policies, setPolicies] = useState<EditorPolicy[]>([]);
  const [events, setEvents] = useState<AuditEvent[]>([]);
  const [nextCursor, setNextCursor] = useState<string>();
  const [error, setError] = useState<string>();
  const [notice, setNotice] = useState<string>();
  const [folderName, setFolderName] = useState("");
  const [groupName, setGroupName] = useState("");
  const [memberGroupId, setMemberGroupId] = useState("");
  const [memberUserId, setMemberUserId] = useState("");
  const [principalType, setPrincipalType] = useState<"user" | "group">("user");
  const [principalId, setPrincipalId] = useState("");
  const [scopeType, setScopeType] = useState<"organization" | "workspace" | "folder" | "document">(
    "workspace",
  );
  const [scopeId, setScopeId] = useState("");
  const [role, setRole] = useState<AuthorizationRole>("viewer");
  const { providers } = useEditorProviders(false);

  const refreshAccess = useCallback(
    async (next: Workspace) => {
      if (!gateway || !next.organizationId) return;
      const [nextFolders, nextGroups, nextGrants] = await Promise.all([
        gateway.folders(next.id),
        gateway.groups(next.organizationId),
        gateway.grants(next.organizationId),
      ]);
      setFolders(nextFolders);
      setGroups(nextGroups);
      setGrants(nextGrants);
    },
    [gateway],
  );

  useEffect(() => {
    if (!gateway || !platform.documents) return;
    platform.documents
      .initializeWorkspace()
      .then(async (next) => {
        setWorkspace(next);
        setScopeId(next.id);
        if (!next.organizationId) throw new Error("organization_context_unavailable");
        if (view === "access") await refreshAccess(next);
        if (view === "policies") setPolicies(await gateway.editorPolicies(next.organizationId));
        if (view === "audit") {
          const page = await gateway.audit(next.organizationId);
          setEvents(page.events);
          setNextCursor(page.nextCursor);
        }
      })
      .catch((reason: Error) => setError(reason.message));
  }, [gateway, platform.documents, refreshAccess, view]);

  if (!gateway)
    return (
      <p className="rounded-xl border border-black/[0.08] bg-white p-5 text-sm text-black/55">
        Server administration is not available in a local desktop workspace.
      </p>
    );
  if (error)
    return (
      <p role="alert" className="rounded-xl bg-red-50 p-4 text-sm text-red-800">
        {error === "administration_resource_not_found"
          ? "You do not have permission to manage this workspace."
          : error}
      </p>
    );
  if (!workspace?.organizationId) return <p className="text-sm text-black/50">Loading…</p>;

  if (view === "policies")
    return (
      <section className="mt-8 rounded-xl border border-black/[0.08] bg-white p-5">
        <h2 className="m-0 text-lg font-semibold">Default editor policy</h2>
        <p className="text-sm text-black/50">
          The server resolves these preferences by document family and falls back to another
          available capable provider.
        </p>
        <div className="grid gap-3 md:grid-cols-2">
          {policyFamilies.map((family) => {
            const current = policies.find((policy) => policy.family === family)?.providerId;
            const options = providers.filter((provider) => provider.families.includes(family));
            return (
              <label key={family} className="text-sm font-medium">
                <span className="mb-1 block capitalize">{family}</span>
                <select
                  value={current ?? ""}
                  onChange={async (event) => {
                    try {
                      const saved = await gateway.setEditorPolicy(
                        workspace.organizationId!,
                        family,
                        event.target.value as EditorPolicy["providerId"],
                      );
                      setPolicies((values) => [
                        ...values.filter((value) => value.family !== family),
                        saved,
                      ]);
                      setNotice(`${family} policy saved.`);
                    } catch (reason) {
                      setError(reason instanceof Error ? reason.message : "policy_save_failed");
                    }
                  }}
                  className="w-full rounded-lg border border-black/10 bg-white px-3 py-2"
                >
                  <option value="" disabled>
                    Use server default
                  </option>
                  {options.map((provider) => (
                    <option key={provider.id} value={provider.id}>
                      {provider.id}{" "}
                      {provider.available ? "" : `(currently ${provider.reason ?? "unavailable"})`}
                    </option>
                  ))}
                </select>
              </label>
            );
          })}
        </div>
        {notice && <p className="mb-0 text-sm text-forest">{notice}</p>}
      </section>
    );

  if (view === "audit")
    return (
      <section className="rounded-xl border border-black/[0.08] bg-white p-5">
        <h2 className="m-0 text-lg font-semibold">Append-only audit trail</h2>
        <div className="mt-4 grid gap-2">
          {events.map((event) => (
            <article key={event.id} className="rounded-lg bg-paper px-4 py-3 text-sm">
              <strong>{event.type}</strong>
              <span className="ml-2 text-black/45">
                {new Date(event.timestamp).toLocaleString()}
              </span>
              <p className="mb-0 mt-1 break-all text-xs text-black/45">
                Resource {event.resourceId ?? "workspace"} · Actor {event.actorId ?? "system"}
              </p>
            </article>
          ))}
          {!events.length && <p className="text-sm text-black/45">No audit events found.</p>}
        </div>
        {nextCursor && (
          <Button
            variant="secondary"
            onClick={async () => {
              const page = await gateway.audit(workspace.organizationId!, nextCursor);
              setEvents((values) => [...values, ...page.events]);
              setNextCursor(page.nextCursor);
            }}
          >
            Load more
          </Button>
        )}
      </section>
    );

  const reload = () => refreshAccess(workspace).catch((reason: Error) => setError(reason.message));
  return (
    <div className="grid gap-5">
      <section className="rounded-xl border border-black/[0.08] bg-white p-5">
        <h2 className="m-0 text-lg font-semibold">Folders</h2>
        <form
          className="mt-4 flex gap-2"
          onSubmit={async (event) => {
            event.preventDefault();
            await gateway.createFolder(workspace.id, folderName);
            setFolderName("");
            await reload();
          }}
        >
          <input
            className="min-w-0 flex-1 rounded-lg border border-black/10 px-3"
            value={folderName}
            onChange={(event) => setFolderName(event.target.value)}
            placeholder="Folder name"
            required
          />
          <Button type="submit">Add folder</Button>
        </form>
        <p className="mb-0 text-sm text-black/45">
          {folders.length ? folders.map((folder) => folder.name).join(" · ") : "No folders"}
        </p>
      </section>

      <section className="rounded-xl border border-black/[0.08] bg-white p-5">
        <h2 className="m-0 text-lg font-semibold">Groups and members</h2>
        <form
          className="mt-4 flex gap-2"
          onSubmit={async (event) => {
            event.preventDefault();
            await gateway.createGroup(workspace.organizationId!, groupName);
            setGroupName("");
            await reload();
          }}
        >
          <input
            className="min-w-0 flex-1 rounded-lg border border-black/10 px-3"
            value={groupName}
            onChange={(event) => setGroupName(event.target.value)}
            placeholder="Group name"
            required
          />
          <Button type="submit">Add group</Button>
        </form>
        <div className="mt-3 grid gap-2">
          {groups.map((group) => (
            <div
              key={group.id}
              className="flex items-center justify-between rounded-lg bg-paper px-3 py-2 text-sm"
            >
              <span>{group.name}</span>
              <button
                type="button"
                className="text-red-700"
                onClick={async () => {
                  await gateway.deleteGroup(group.id);
                  await reload();
                }}
              >
                Delete
              </button>
            </div>
          ))}
        </div>
        <form
          className="mt-4 grid gap-2 md:grid-cols-[1fr_1fr_auto]"
          onSubmit={async (event) => {
            event.preventDefault();
            await gateway.addGroupMember(memberGroupId, memberUserId);
            setMemberUserId("");
            setNotice("Member added.");
          }}
        >
          <select
            className="rounded-lg border border-black/10 px-3 py-2"
            value={memberGroupId}
            onChange={(event) => setMemberGroupId(event.target.value)}
            required
          >
            <option value="">Choose group</option>
            {groups.map((group) => (
              <option key={group.id} value={group.id}>
                {group.name}
              </option>
            ))}
          </select>
          <input
            className="rounded-lg border border-black/10 px-3"
            value={memberUserId}
            onChange={(event) => setMemberUserId(event.target.value)}
            placeholder="User UUID"
            required
          />
          <Button type="submit">Add member</Button>
        </form>
      </section>

      <section className="rounded-xl border border-black/[0.08] bg-white p-5">
        <h2 className="m-0 text-lg font-semibold">Authorization grants</h2>
        <form
          className="mt-4 grid gap-2 md:grid-cols-2"
          onSubmit={async (event) => {
            event.preventDefault();
            await gateway.createGrant(workspace.organizationId!, {
              principalType,
              principalId,
              scopeType,
              scopeId,
              role,
            });
            setPrincipalId("");
            await reload();
          }}
        >
          <select
            className="rounded-lg border border-black/10 px-3 py-2"
            value={principalType}
            onChange={(event) => setPrincipalType(event.target.value as typeof principalType)}
          >
            <option value="user">User</option>
            <option value="group">Group</option>
          </select>
          <input
            className="rounded-lg border border-black/10 px-3"
            value={principalId}
            onChange={(event) => setPrincipalId(event.target.value)}
            placeholder="Principal UUID"
            required
          />
          <select
            className="rounded-lg border border-black/10 px-3 py-2"
            value={scopeType}
            onChange={(event) => {
              const next = event.target.value as typeof scopeType;
              setScopeType(next);
              setScopeId(
                next === "organization"
                  ? workspace.organizationId!
                  : next === "workspace"
                    ? workspace.id
                    : "",
              );
            }}
          >
            <option value="organization">Organization</option>
            <option value="workspace">Workspace</option>
            <option value="folder">Folder</option>
            <option value="document">Document</option>
          </select>
          <input
            className="rounded-lg border border-black/10 px-3"
            value={scopeId}
            onChange={(event) => setScopeId(event.target.value)}
            placeholder="Scope UUID"
            required
          />
          <select
            className="rounded-lg border border-black/10 px-3 py-2"
            value={role}
            onChange={(event) => setRole(event.target.value as AuthorizationRole)}
          >
            {roles.map((value) => (
              <option key={value} value={value}>
                {value}
              </option>
            ))}
          </select>
          <Button type="submit">Create grant</Button>
        </form>
        <div className="mt-4 grid gap-2">
          {grants.map((grant) => (
            <div
              key={grant.id}
              className="flex items-center justify-between rounded-lg bg-paper px-3 py-2 text-sm"
            >
              <span className="break-all">
                {grant.role} · {grant.principalType} {grant.principalId} · {grant.scopeType}
              </span>
              <button
                type="button"
                className="ml-3 text-red-700"
                onClick={async () => {
                  await gateway.deleteGrant(grant.id);
                  await reload();
                }}
              >
                Remove
              </button>
            </div>
          ))}
        </div>
        {notice && <p className="mb-0 text-sm text-forest">{notice}</p>}
      </section>
    </div>
  );
}
