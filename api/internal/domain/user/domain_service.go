package user

import (
	"context"

	"github.com/UniPro-tech/UniQUE-API/api/pkg"
)

type UserDomainService struct {
	repo UserServiceRepository
}

func NewUserDomainService(repo UserServiceRepository) *UserDomainService {
	return &UserDomainService{repo: repo}
}

func (uds *UserDomainService) ListUser(ctx context.Context) ([]*User, int64, error) {
	users, count, err := uds.repo.ListUser(ctx)
	if err != nil {
		return nil, 0, err
	}
	return users, count, nil
}

func (uds *UserDomainService) FindUserById(ctx context.Context, id string) (*User, error) {
	user, err := uds.repo.FindUserById(ctx, id)
	if err != nil {
		return nil, err
	}
	return user, nil
}

func (uds *UserDomainService) EditUser(ctx context.Context, param *User) error {
	if err := param.custom_id.Valid(); err != nil {
		return ErrInvalidCustomID
	}
	if err := param.email.Valid(param.GetCustomID(), param.GetPeriod()); err != nil {
		return ErrUserEmailAddress
	}
	if err := param.external_email.Valid(); err != nil {
		return ErrUserEmailAddress
	}
	err := uds.repo.Save(ctx, param)
	if err != nil {
		return err
	}
	return nil
}

func (uds *UserDomainService) DeleteUser(ctx context.Context, id string) error {
	err := uds.repo.Delete(ctx, id)
	if err != nil {
		return err
	}
	return nil
}

func (uds *UserDomainService) SearchUser(ctx context.Context, searchParams pkg.UserParams) ([]*User, int64, error) {
	users, count, err := uds.repo.Search(ctx, searchParams)
	if err != nil {
		return nil, 0, err
	}
	return users, count, nil
}

func (uds *UserDomainService) AddUser(ctx context.Context, param *User) error {
	if err := param.custom_id.Valid(); err != nil {
		return ErrInvalidCustomID
	}
	if err := param.email.Valid(param.GetCustomID(), param.GetPeriod()); err != nil {
		return ErrUserEmailAddress
	}
	if err := param.external_email.Valid(); err != nil {
		return ErrUserEmailAddress
	}
	err := uds.repo.Create(ctx, param)
	if err != nil {
		return err
	}
	return nil
}
