package editors

import (
	"context"
	"errors"
	"testing"
)

func TestRegistryRejectsDuplicatePersistedID(t *testing.T) {
	provider := NewNativeProvider()
	_, err := NewRegistry(provider, provider)
	if err == nil {
		t.Fatal("NewRegistry() accepted duplicate provider ID")
	}
}

func TestResolverUsesCapabilitiesAvailabilityAndPolicyOrder(t *testing.T) {
	unavailable := NewStaticProvider(
		ProviderCollabora,
		Capabilities{Operations: []Capability{CapabilityEdit, CapabilityBrowser}, Families: []DocumentFamily{FamilyText}},
		Availability{Available: false, Reason: "disabled"},
	)
	available := NewStaticProvider(
		ProviderOnlyOffice,
		Capabilities{Operations: []Capability{CapabilityEdit, CapabilityBrowser}, Families: []DocumentFamily{FamilyText}},
		Availability{Available: true},
	)
	registry, err := NewRegistry(unavailable, available)
	if err != nil {
		t.Fatal(err)
	}
	resolver := NewResolver(registry, Policy{FamilyText: {ProviderCollabora, ProviderOnlyOffice}})

	provider, err := resolver.Resolve(context.Background(), FamilyText, CapabilityEdit, CapabilityBrowser)
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}
	if provider.ID() != ProviderOnlyOffice {
		t.Fatalf("Resolve() = %q, want %q", provider.ID(), ProviderOnlyOffice)
	}
}

func TestResolverDoesNotSelectMissingCapability(t *testing.T) {
	provider := NewStaticProvider(
		ProviderCollabora,
		Capabilities{Operations: []Capability{CapabilityView}, Families: []DocumentFamily{FamilyText}},
		Availability{Available: true},
	)
	registry, err := NewRegistry(provider)
	if err != nil {
		t.Fatal(err)
	}
	resolver := NewResolver(registry, Policy{FamilyText: {ProviderCollabora}})

	_, err = resolver.Resolve(context.Background(), FamilyText, CapabilityEdit)
	if !errors.Is(err, ErrProviderUnavailable) {
		t.Fatalf("Resolve() error = %v, want ErrProviderUnavailable", err)
	}
}

func TestNativeProviderCreatesManagedNativeFormat(t *testing.T) {
	result, err := NewNativeProvider().Create(context.Background(), CreateDocumentRequest{Family: FamilyNative})
	if err != nil {
		t.Fatal(err)
	}
	if result.StorageMode != StorageManaged || result.NativeFormat != "documi-json-v1" {
		t.Fatalf("Create() = %#v", result)
	}
}

func TestResolverSelectsEditorByMIMEType(t *testing.T) {
	provider := NewStaticProvider(
		ProviderCollabora,
		Capabilities{Operations: []Capability{CapabilityEdit, CapabilityBrowser}, Families: []DocumentFamily{FamilySpreadsheet}},
		Availability{Available: true},
	)
	registry, err := NewRegistry(provider)
	if err != nil {
		t.Fatal(err)
	}
	resolver := NewResolver(registry, Policy{FamilySpreadsheet: {ProviderCollabora}})

	selected, err := resolver.ResolveMIME(context.Background(), "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet", CapabilityEdit, CapabilityBrowser)
	if err != nil {
		t.Fatal(err)
	}
	if selected.ID() != ProviderCollabora {
		t.Fatalf("ResolveMIME() = %q, want %q", selected.ID(), ProviderCollabora)
	}
}
