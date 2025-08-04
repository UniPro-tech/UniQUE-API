package role

import (
	"context"

	"github.com/UniPro-tech/UniQUE-API/api/pkg"
)

//go:generate moq -out RoleServiceRepository_mock.go . RoleServiceRepository
type RoleServiceRepository interface {
	ListRole(ctx context.Context) ([]*Role, int64, error)
	Save(ctx context.Context, param *Role) error
	Delete(ctx context.Context, id string) error
	Search(ctx context.Context, searchParams pkg.RoleParams) ([]*Role, int64, error)
	Create(ctx context.Context, param *Role) error
	Update(ctx context.Context, param *Role) error
}
