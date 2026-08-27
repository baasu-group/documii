package permissions

type Role string
type Capability string

const (
	RoleOwner            Role       = "owner"
	RoleAdmin            Role       = "admin"
	RoleManager          Role       = "manager"
	RoleEditor           Role       = "editor"
	RoleContributor      Role       = "contributor"
	RoleViewer           Role       = "viewer"
	RoleGuest            Role       = "guest"
	CanRead              Capability = "read"
	CanCreate            Capability = "create"
	CanEdit              Capability = "edit"
	CanDownload          Capability = "download"
	CanDelete            Capability = "delete"
	CanManagePermissions Capability = "manage_permissions"
	CanManageMembers     Capability = "manage_members"
	CanTransferOwnership Capability = "transfer_ownership"
)

var roleCapabilities = map[Role][]Capability{
	RoleOwner:   {CanRead, CanCreate, CanEdit, CanDownload, CanDelete, CanManagePermissions, CanManageMembers, CanTransferOwnership},
	RoleAdmin:   {CanRead, CanCreate, CanEdit, CanDownload, CanDelete, CanManagePermissions, CanManageMembers},
	RoleManager: {CanRead, CanCreate, CanEdit, CanDownload, CanManagePermissions}, RoleEditor: {CanRead, CanCreate, CanEdit, CanDownload},
	RoleContributor: {CanRead, CanCreate, CanEdit}, RoleViewer: {CanRead, CanDownload}, RoleGuest: {CanRead},
}

func Capabilities(roles ...Role) map[Capability]bool {
	result := map[Capability]bool{}
	for _, role := range roles {
		for _, capability := range roleCapabilities[role] {
			result[capability] = true
		}
	}
	return result
}
func ValidRole(role Role) bool { _, ok := roleCapabilities[role]; return ok }
