package git

import "context"

type Provider interface {
	CreateRepository(ctx context.Context, r *Repository) (bool, error)
	AddTeam(ctx context.Context, r *Repository, name, permission string) (bool, error)
	AddDeployKey(ctx context.Context, r *Repository, key, keyName string) (bool, error)
}
