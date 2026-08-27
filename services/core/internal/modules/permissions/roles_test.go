package permissions

import "testing"

func TestRolesAreAdditiveCapabilityBundles(t *testing.T) {
	capabilities := Capabilities(RoleGuest, RoleEditor)
	for _, required := range []Capability{CanRead, CanCreate, CanEdit, CanDownload} {
		if !capabilities[required] {
			t.Fatalf("missing %q", required)
		}
	}
	if capabilities[CanManagePermissions] {
		t.Fatal("editor unexpectedly manages permissions")
	}
}
func TestOwnerIsOnlyTransferRole(t *testing.T) {
	for _, role := range []Role{RoleAdmin, RoleManager, RoleEditor, RoleContributor, RoleViewer, RoleGuest} {
		if Capabilities(role)[CanTransferOwnership] {
			t.Fatalf("%q transfers ownership", role)
		}
	}
}
