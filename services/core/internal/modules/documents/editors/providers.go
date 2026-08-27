package editors

import "context"

type StaticProvider struct {
	id           ProviderID
	capabilities Capabilities
	availability Availability
	create       func(context.Context, CreateDocumentRequest) (*CreateDocumentResult, error)
	open         func(context.Context, OpenDocumentRequest) (*OpenDocumentResult, error)
}

func NewStaticProvider(
	id ProviderID,
	capabilities Capabilities,
	availability Availability,
) *StaticProvider {
	return &StaticProvider{id: id, capabilities: capabilities, availability: availability}
}

func NewNativeProvider() *StaticProvider {
	provider := NewStaticProvider(
		ProviderNative,
		Capabilities{
			Operations: []Capability{CapabilityCreate, CapabilityEdit, CapabilityView, CapabilityBrowser, CapabilityDesktop, CapabilityOffline},
			Families:   []DocumentFamily{FamilyNative},
		},
		Availability{Available: true},
	)
	provider.create = func(_ context.Context, _ CreateDocumentRequest) (*CreateDocumentResult, error) {
		return &CreateDocumentResult{StorageMode: StorageManaged, NativeFormat: "documi-json-v1"}, nil
	}
	provider.open = func(_ context.Context, request OpenDocumentRequest) (*OpenDocumentResult, error) {
		if !request.CanRead {
			return nil, ErrProviderUnavailable
		}
		return &OpenDocumentResult{
			Mode:     "native",
			URL:      "/documents/" + request.DocumentID + "/edit",
			ReadOnly: !request.CanEdit,
		}, nil
	}
	return provider
}

func (provider *StaticProvider) ID() ProviderID { return provider.id }

func (provider *StaticProvider) Capabilities(context.Context) Capabilities {
	return provider.capabilities
}

func (provider *StaticProvider) Available(context.Context) Availability {
	return provider.availability
}

func (provider *StaticProvider) Create(ctx context.Context, request CreateDocumentRequest) (*CreateDocumentResult, error) {
	if !provider.availability.Available {
		return nil, ErrProviderUnavailable
	}
	if provider.create == nil {
		return nil, ErrOperationDeferred
	}
	return provider.create(ctx, request)
}

func (provider *StaticProvider) Open(ctx context.Context, request OpenDocumentRequest) (*OpenDocumentResult, error) {
	if !provider.availability.Available {
		return nil, ErrProviderUnavailable
	}
	if provider.open == nil {
		return nil, ErrOperationDeferred
	}
	return provider.open(ctx, request)
}
