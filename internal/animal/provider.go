package animal

import "context"

// Provider is the only door to animal data.
type Provider interface {
	// Search returns the adoptable pool: active dogs that pass the pool filter.
	Search(ctx context.Context) ([]Animal, error)
	GetAnimal(ctx context.Context, id string) (*Animal, error)
	GetOrganization(ctx context.Context, id string) (*Organization, error)
	// GetStatus reports REMOVED_UNKNOWN for ids the provider no longer sees.
	GetStatus(ctx context.Context, id string) (Status, error)
}
