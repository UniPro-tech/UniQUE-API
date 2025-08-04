package role

import (
	"context"
)

//go:generate moq -out RoleServiceRepository_mock.go . RoleServiceRepository
type RoleServiceRepository interface {
	ListRole(ctx context.Context) ([]*Role, int64, error)
	FindRoleById(ctx context.Context, id string) (*Role, error)
	Save(ctx context.Context, param *Role) error
	Delete(ctx context.Context, id string) error
	// TODO: 直す
	//	Search(ctx context.Context, searchParams pkg.RoleParams) ([]*Role, int64, error)
	Create(ctx context.Context, param *Role) error
	Update(ctx context.Context, param *Role) error
}
