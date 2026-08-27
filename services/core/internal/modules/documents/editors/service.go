package editors

import (
	"context"
	"errors"
	"fmt"
)

var ErrSessionForbidden = errors.New("editor session forbidden")

type Service struct {
	registry *Registry
	resolver *Resolver
}

func NewService(registry *Registry, resolver *Resolver) *Service {
	return &Service{registry: registry, resolver: resolver}
}

func (service *Service) Providers(ctx context.Context) []Status {
	return service.registry.Statuses(ctx)
}

func (service *Service) Provider(ctx context.Context, id ProviderID) (Status, error) {
	provider, exists := service.registry.Get(id)
	if !exists {
		return Status{}, ErrProviderNotFound
	}
	availability := provider.Available(ctx)
	capabilities := provider.Capabilities(ctx)
	return Status{
		ID:           id,
		Available:    availability.Available,
		Reason:       availability.Reason,
		Capabilities: append([]Capability(nil), capabilities.Operations...),
		Families:     append([]DocumentFamily(nil), capabilities.Families...),
	}, nil
}

func (service *Service) Resolve(ctx context.Context, family DocumentFamily, required ...Capability) (Status, error) {
	provider, err := service.resolver.Resolve(ctx, family, required...)
	if err != nil {
		return Status{}, err
	}
	return service.Provider(ctx, provider.ID())
}

func (service *Service) ResolveForOrganization(ctx context.Context, organizationID string, family DocumentFamily, required ...Capability) (Status, error) {
	provider, err := service.resolver.ResolveForOrganization(ctx, organizationID, family, required...)
	if err != nil {
		return Status{}, err
	}
	return service.Provider(ctx, provider.ID())
}

type SessionRequest struct {
	ActorID       string
	DocumentID    string
	ProviderID    ProviderID
	RequestedEdit bool
}

type Session struct {
	ProviderID     ProviderID `json:"providerId"`
	Mode           string     `json:"mode"`
	URL            string     `json:"url,omitempty"`
	AccessToken    string     `json:"accessToken,omitempty"`
	ExpiresAt      string     `json:"expiresAt,omitempty"`
	ReadOnly       bool       `json:"readOnly"`
	AccessTokenTtl int64      `json:"accessTokenTtl,omitempty"`
}

type SessionService interface {
	CreateSession(context.Context, SessionRequest) (Session, error)
}

type UnavailableSessionService struct{}

func (UnavailableSessionService) CreateSession(context.Context, SessionRequest) (Session, error) {
	return Session{}, fmt.Errorf("document identity, authorization, and version repository are not implemented: %w", ErrOperationDeferred)
}
