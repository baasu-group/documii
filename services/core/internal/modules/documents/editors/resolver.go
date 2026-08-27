package editors

import (
	"context"
	"fmt"
)

type Policy map[DocumentFamily][]ProviderID

type Resolver struct {
	registry *Registry
	policy   Policy
}

func NewResolver(registry *Registry, policy Policy) *Resolver {
	return &Resolver{registry: registry, policy: policy}
}

func DefaultPolicy() Policy {
	return Policy{
		FamilyNative:       {ProviderNative},
		FamilyText:         {ProviderCollabora, ProviderOnlyOffice, ProviderLibreOfficeDesktop},
		FamilySpreadsheet:  {ProviderCollabora, ProviderOnlyOffice, ProviderLibreOfficeDesktop},
		FamilyPresentation: {ProviderCollabora, ProviderOnlyOffice, ProviderLibreOfficeDesktop},
		FamilyPDF:          {ProviderCollabora, ProviderOnlyOffice},
	}
}

func (resolver *Resolver) Resolve(ctx context.Context, family DocumentFamily, required ...Capability) (Provider, error) {
	candidates := resolver.policy[family]
	for _, id := range candidates {
		provider, exists := resolver.registry.Get(id)
		if !exists {
			continue
		}
		capabilities := provider.Capabilities(ctx)
		if !capabilities.SupportsFamily(family) || !capabilities.Supports(required...) {
			continue
		}
		if provider.Available(ctx).Available {
			return provider, nil
		}
	}
	return nil, fmt.Errorf("resolve editor for family %q: %w", family, ErrProviderUnavailable)
}

func (resolver *Resolver) ResolveMIME(ctx context.Context, mimeType string, required ...Capability) (Provider, error) {
	family, ok := FamilyForMIME(mimeType)
	if !ok {
		return nil, ErrProviderUnavailable
	}
	return resolver.Resolve(ctx, family, required...)
}
