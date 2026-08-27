package editors

import (
	"context"
	"fmt"
	"sync"
)

type Registry struct {
	mu        sync.RWMutex
	providers map[ProviderID]Provider
	order     []ProviderID
}

func NewRegistry(providers ...Provider) (*Registry, error) {
	registry := &Registry{providers: make(map[ProviderID]Provider)}
	for _, provider := range providers {
		if err := registry.Register(provider); err != nil {
			return nil, err
		}
	}
	return registry, nil
}

func (registry *Registry) Register(provider Provider) error {
	if provider == nil || provider.ID() == "" {
		return fmt.Errorf("register editor provider: provider and ID are required")
	}
	registry.mu.Lock()
	defer registry.mu.Unlock()
	if _, exists := registry.providers[provider.ID()]; exists {
		return fmt.Errorf("register editor provider %q: duplicate ID", provider.ID())
	}
	registry.providers[provider.ID()] = provider
	registry.order = append(registry.order, provider.ID())
	return nil
}

func (registry *Registry) Get(id ProviderID) (Provider, bool) {
	registry.mu.RLock()
	defer registry.mu.RUnlock()
	provider, exists := registry.providers[id]
	return provider, exists
}

func (registry *Registry) Statuses(ctx context.Context) []Status {
	registry.mu.RLock()
	defer registry.mu.RUnlock()
	statuses := make([]Status, 0, len(registry.order))
	for _, id := range registry.order {
		provider := registry.providers[id]
		availability := provider.Available(ctx)
		capabilities := provider.Capabilities(ctx)
		statuses = append(statuses, Status{
			ID:           id,
			Available:    availability.Available,
			Reason:       availability.Reason,
			Capabilities: append([]Capability(nil), capabilities.Operations...),
			Families:     append([]DocumentFamily(nil), capabilities.Families...),
		})
	}
	return statuses
}
