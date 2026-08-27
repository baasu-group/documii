package editors

import (
	"context"
	"fmt"
)

type Policy map[DocumentFamily][]ProviderID

type PolicyStore interface {
	Preferred(context.Context, string, DocumentFamily) (ProviderID, bool, error)
}

type Resolver struct {
	registry *Registry
	policy   Policy
	store    PolicyStore
}

func (resolver *Resolver) WithPolicyStore(store PolicyStore) *Resolver {
	resolver.store = store
	return resolver
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
	return resolver.resolveCandidates(ctx, family, resolver.policy[family], required...)
}

func (resolver *Resolver) ResolveForOrganization(ctx context.Context, organizationID string, family DocumentFamily, required ...Capability) (Provider, error) {
	candidates := append([]ProviderID(nil), resolver.policy[family]...)
	if resolver.store != nil {
		preferred, found, err := resolver.store.Preferred(ctx, organizationID, family)
		if err != nil {
			return nil, err
		}
		if found {
			candidates = append([]ProviderID{preferred}, candidates...)
		}
	}
	return resolver.resolveCandidates(ctx, family, candidates, required...)
}

func (resolver *Resolver) resolveCandidates(ctx context.Context, family DocumentFamily, candidates []ProviderID, required ...Capability) (Provider, error) {
	seen := map[ProviderID]bool{}
	for _, id := range candidates {
		if seen[id] {
			continue
		}
		seen[id] = true
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
