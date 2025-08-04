package user

import (
	"context"

	"github.com/UniPro-tech/UniQUE-API/api/pkg"
)

//go:generate moq -out UserServiceRepository_mock.go . UserServiceRepository
type UserServiceRepository interface {
	ListUser(ctx context.Context) ([]*User, int64, error)
	Save(ctx context.Context, param *User) error
	Delete(ctx context.Context, id string) error
	Search(ctx context.Context, searchParams pkg.UserParams) ([]*User, int64, error)
	Create(ctx context.Context, param *User) error
	Update(ctx context.Context, param *User) error
}
