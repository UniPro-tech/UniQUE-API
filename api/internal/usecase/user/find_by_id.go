package usecase

import (
	"context"
	"log/slog"

	"github.com/UniPro-tech/UniQUE-API/api/internal/domain/user"
	"github.com/UniPro-tech/UniQUE-API/api/pkg"
)

type FindUserByIdUsecase struct {
	uds user.IUserDomainService
}

func NewFindUserByIdUsecase(uds user.IUserDomainService) *FindUserByIdUsecase {
	return &FindUserByIdUsecase{uds: uds}
}

func (us *FindUserByIdUsecase) Run(ctx context.Context, id string) (*user.User, error) {
	value, ok := ctx.Value("ctxInfo").(pkg.CtxInfo)
	if !ok {
		return nil, INVALID_REQUEST_ID
	}

	user, err := us.uds.FindUserById(ctx, id)
	if err != nil {
		slog.Info("can not complete FindById usecase", "request id", value.RequestId)
		return nil, err
	}

	slog.Info("process done FindById usecase", "request id", value.RequestId, "user", user)
	return user, nil
}
