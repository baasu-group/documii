import type {
  AuditEvent,
  AuthorizationGrant,
  AuthorizationCapability,
  AuthorizationRole,
  DocumentFamily,
  EditorPolicy,
  FolderRecord,
  GroupRecord,
  GroupMember,
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
  const [auditEventFilter, setAuditEventFilter] = useState("");
  const [auditWorkspaceOnly, setAuditWorkspaceOnly] = useState(false);
  const [error, setError] = useState<string>();
  const [notice, setNotice] = useState<string>();
  const [folderName, setFolderName] = useState("");
  const [folderParentId, setFolderParentId] = useState("");
  const [groupName, setGroupName] = useState("");
  const [memberGroupId, setMemberGroupId] = useState("");
  const [memberUserId, setMemberUserId] = useState("");
  const [members, setMembers] = useState<GroupMember[]>([]);
  const [principalType, setPrincipalType] = useState<"user" | "group">("user");
  const [principalId, setPrincipalId] = useState("");
  const [scopeType, setScopeType] = useState<"organization" | "workspace" | "folder" | "document">(
    "workspace",
  );
  const [scopeId, setScopeId] = useState("");
  const [role, setRole] = useState<AuthorizationRole>("viewer");
  const [diagnosticSubjectId, setDiagnosticSubjectId] = useState("");
  const [effectiveCapabilities, setEffectiveCapabilities] = useState<AuthorizationCapability[]>([]);
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

  useEffect(() => {
    if (!gateway || !memberGroupId) {
      setMembers([]);
      return;
    }
    gateway
      .groupMembers(memberGroupId)
      .then(setMembers)
      .catch((reason: Error) => setError(reason.message));
  }, [gateway, memberGroupId]);

  if (!gateway)
    return (
      <p className="rounded-xl border border-black/[0.08] bg-white p-5 text-sm text-black/55">
        Server administration is not available in a local desktop workspace.
      </p>
    );
  if (error && !workspace)
    return (
      <p role="alert" className="rounded-xl bg-red-50 p-4 text-sm text-red-800">
        {error === "administration_resource_not_found"
          ? "You do not have permission to manage this workspace."
          : error}
      </p>
    );
  if (!workspace?.organizationId) return <p className="text-sm text-black/50">Loading…</p>;

  const perform = async (action: () => Promise<void>) => {
    setError(undefined);
    try {
      await action();
    } catch (reason) {
      setError(reason instanceof Error ? reason.message : "administration_operation_failed");
    }
  };

  if (view === "policies")
    return (
      <section className="mt-8 rounded-xl border border-black/[0.08] bg-white p-5">
        <h2 className="m-0 text-lg font-semibold">Default editor policy</h2>
        {error && (
          <p role="alert" className="rounded-lg bg-red-50 p-3 text-sm text-red-800">
            {error}
          </p>
        )}
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
        {error && (
          <p role="alert" className="rounded-lg bg-red-50 p-3 text-sm text-red-800">
            {error}
          </p>
        )}
        <form
          className="mt-4 grid gap-2 md:grid-cols-[1fr_auto_auto]"
          onSubmit={async (event) => {
            event.preventDefault();
            await perform(async () => {
              const page = await gateway.audit(workspace.organizationId!, {
                eventType: auditEventFilter || undefined,
                workspaceId: auditWorkspaceOnly ? workspace.id : undefined,
              });
              setEvents(page.events);
              setNextCursor(page.nextCursor);
            });
          }}
        >
          <input
            className="rounded-lg border border-black/10 px-3"
            value={auditEventFilter}
            onChange={(event) => setAuditEventFilter(event.target.value)}
            placeholder="Exact event type (optional)"
          />
          <label className="flex items-center gap-2 text-sm">
            <input
              type="checkbox"
              checked={auditWorkspaceOnly}
              onChange={(event) => setAuditWorkspaceOnly(event.target.checked)}
            />
            This workspace only
          </label>
          <Button type="submit" variant="secondary">
            Apply filters
          </Button>
        </form>
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
              await perform(async () => {
                const page = await gateway.audit(workspace.organizationId!, {
                  cursor: nextCursor,
                  eventType: auditEventFilter || undefined,
                  workspaceId: auditWorkspaceOnly ? workspace.id : undefined,
                });
                setEvents((values) => [...values, ...page.events]);
                setNextCursor(page.nextCursor);
              });
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
      {error && (
        <p role="alert" className="m-0 rounded-lg bg-red-50 p-3 text-sm text-red-800">
          {error}
        </p>
      )}
      <section className="rounded-xl border border-black/[0.08] bg-white p-5">
        <h2 className="m-0 text-lg font-semibold">Folders</h2>
        <form
          className="mt-4 grid gap-2 md:grid-cols-[1fr_1fr_auto]"
          onSubmit={async (event) => {
            event.preventDefault();
            await perform(async () => {
              await gateway.createFolder(workspace.id, folderName, folderParentId || undefined);
              setFolderName("");
              setFolderParentId("");
              await reload();
            });
          }}
        >
          <input
            className="min-w-0 flex-1 rounded-lg border border-black/10 px-3"
            value={folderName}
            onChange={(event) => setFolderName(event.target.value)}
            placeholder="Folder name"
            required
          />
          <select
            className="rounded-lg border border-black/10 px-3 py-2"
            value={folderParentId}
            onChange={(event) => setFolderParentId(event.target.value)}
          >
            <option value="">Workspace root</option>
            {folders.map((folder) => (
              <option key={folder.id} value={folder.id}>
                {folder.name}
              </option>
            ))}
          </select>
          <Button type="submit">Add folder</Button>
        </form>
        <div className="mt-3 grid gap-2">
          {folders.map((folder) => (
            <div
              key={folder.id}
              className="flex items-center justify-between rounded-lg bg-paper px-3 py-2 text-sm"
            >
              <span>
                {folder.name}
                {folder.parentId ? " · nested" : ""}
              </span>
              <button
                type="button"
                className="text-red-700"
                onClick={async () => {
                  await perform(async () => {
                    await gateway.deleteFolder(folder.id);
                    await reload();
                  });
                }}
              >
                Delete
              </button>
            </div>
          ))}
          {!folders.length && <p className="mb-0 text-sm text-black/45">No folders</p>}
        </div>
      </section>

      <section className="rounded-xl border border-black/[0.08] bg-white p-5">
        <h2 className="m-0 text-lg font-semibold">Groups and members</h2>
        <form
          className="mt-4 flex gap-2"
          onSubmit={async (event) => {
            event.preventDefault();
            await perform(async () => {
              await gateway.createGroup(workspace.organizationId!, groupName);
              setGroupName("");
              await reload();
            });
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
                  await perform(async () => {
                    await gateway.deleteGroup(group.id);
                    await reload();
                  });
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
            await perform(async () => {
              await gateway.addGroupMember(memberGroupId, memberUserId);
              setMemberUserId("");
              setMembers(await gateway.groupMembers(memberGroupId));
              setNotice("Member added.");
            });
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
        <div className="mt-3 grid gap-2">
          {members.map((member) => (
            <div
              key={member.id}
              className="flex items-center justify-between rounded-lg bg-paper px-3 py-2 text-sm"
            >
              <span>
                {member.displayName}
                {member.email ? ` · ${member.email}` : ""}
              </span>
              <button
                type="button"
                className="text-red-700"
                onClick={async () => {
                  await perform(async () => {
                    await gateway.removeGroupMember(memberGroupId, member.id);
                    setMembers(await gateway.groupMembers(memberGroupId));
                  });
                }}
              >
                Remove
              </button>
            </div>
          ))}
        </div>
      </section>

      <section className="rounded-xl border border-black/[0.08] bg-white p-5">
        <h2 className="m-0 text-lg font-semibold">Authorization grants</h2>
        <form
          className="mt-4 grid gap-2 md:grid-cols-2"
          onSubmit={async (event) => {
            event.preventDefault();
            await perform(async () => {
              await gateway.createGrant(workspace.organizationId!, {
                principalType,
                principalId,
                scopeType,
                scopeId,
                role,
              });
              setPrincipalId("");
              await reload();
            });
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
          {principalType === "group" ? (
            <select
              className="rounded-lg border border-black/10 px-3 py-2"
              value={principalId}
              onChange={(event) => setPrincipalId(event.target.value)}
              required
            >
              <option value="">Choose group</option>
              {groups.map((group) => (
                <option key={group.id} value={group.id}>
                  {group.name}
                </option>
              ))}
            </select>
          ) : (
            <input
              className="rounded-lg border border-black/10 px-3"
              value={principalId}
              onChange={(event) => setPrincipalId(event.target.value)}
              placeholder="User UUID"
              required
            />
          )}
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
                  await perform(async () => {
                    await gateway.deleteGrant(grant.id);
                    await reload();
                  });
                }}
              >
                Remove
              </button>
            </div>
          ))}
        </div>
        {notice && <p className="mb-0 text-sm text-forest">{notice}</p>}
      </section>

      <section className="rounded-xl border border-black/[0.08] bg-white p-5">
        <h2 className="m-0 text-lg font-semibold">Effective access diagnostics</h2>
        <p className="text-sm text-black/50">
          Verify the union of direct, group, and inherited grants at the scope selected above.
        </p>
        <form
          className="grid gap-2 md:grid-cols-[1fr_auto]"
          onSubmit={async (event) => {
            event.preventDefault();
            await perform(async () => {
              setEffectiveCapabilities(
                await gateway.effectiveAccess(diagnosticSubjectId, scopeType, scopeId),
              );
            });
          }}
        >
          <input
            className="rounded-lg border border-black/10 px-3"
            value={diagnosticSubjectId}
            onChange={(event) => setDiagnosticSubjectId(event.target.value)}
            placeholder="User UUID"
            required
          />
          <Button type="submit">Check access</Button>
        </form>
        <p className="mb-0 mt-3 text-sm text-black/55">
          {effectiveCapabilities.length
            ? effectiveCapabilities.join(" · ")
            : "No diagnostic has been run."}
        </p>
      </section>
    </div>
  );
}
