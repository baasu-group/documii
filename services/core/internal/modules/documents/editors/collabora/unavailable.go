package collabora

import (
	"context"
	"io"
)

// UnavailableGateway keeps WOPI routes fail-closed until the Documents domain
// has persistence, permission, version, storage, and audit implementations.
type UnavailableGateway struct{}

func (UnavailableGateway) Access(context.Context, string, string) (Access, error) {
	return Access{}, ErrForbidden
}

func (UnavailableGateway) Info(context.Context, string, string) (File, error) {
	return File{}, ErrForbidden
}

func (UnavailableGateway) OpenContent(context.Context, string, string) (io.ReadCloser, File, error) {
	return nil, File{}, ErrForbidden
}

func (UnavailableGateway) SaveNewVersion(context.Context, SaveRequest) (File, error) {
	return File{}, ErrForbidden
}
func (UnavailableGateway) ManageLock(context.Context, LockRequest) (LockResult, error) {
	return LockResult{}, ErrForbidden
}
