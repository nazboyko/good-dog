package animal

import "context"

// Provider is the only door to animal data.
type Provider interface {
	// Search returns the pool the game can play: every real dog that
	// passes the pool filter, whatever the listing says now. A dog who
	// went home stays in and the reveal tells the truth about her.
	Search(ctx context.Context) ([]Animal, error)
	GetAnimal(ctx context.Context, id string) (*Animal, error)
	GetOrganization(ctx context.Context, id string) (*Organization, error)
	// GetStatus reports REMOVED_UNKNOWN for ids the provider no longer sees.
	GetStatus(ctx context.Context, id string) (Status, error)
}
