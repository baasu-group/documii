package editors

import (
	"context"
	"testing"
)

func TestGoogleProvidersAreServerSideDisabledInAirGap(t *testing.T) {
	registry, err := NewRegistry(DefaultProviders(ProviderConfiguration{AirGapped: true})...)
	if err != nil {
		t.Fatal(err)
	}
	for _, id := range []ProviderID{ProviderGoogleDocs, ProviderGoogleSheets, ProviderGoogleSlides} {
		provider, _ := registry.Get(id)
		availability := provider.Available(context.Background())
		if availability.Available || availability.Reason != "air_gapped" {
			t.Fatalf("provider %q availability = %#v", id, availability)
		}
		if _, err := provider.Create(context.Background(), CreateDocumentRequest{}); err == nil {
			t.Fatalf("provider %q allowed Create() in air-gapped mode", id)
		}
	}
}
