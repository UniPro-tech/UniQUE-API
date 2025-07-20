package usecase

import (
	"context"
	"log/slog"

	"github.com/UniPro-tech/UniQUE-API/api/internal/domain/user"
	"github.com/UniPro-tech/UniQUE-API/api/pkg"
)

type CreateUserUsecase struct {
	uds user.IUserDomainService
}

func NewCreateUserUsecase(uds user.IUserDomainService) *CreateUserUsecase {
	return &CreateUserUsecase{uds: uds}
}

func (us *CreateUserUsecase) Run(ctx context.Context, param *user.User) error {
	value, ok := ctx.Value("ctxInfo").(pkg.CtxInfo)
	if !ok {
		return INVALID_REQUEST_ID
	}

	if err := us.uds.AddUser(ctx, param); err != nil {
		slog.Error("can not complete SaveUser Usecase", "error msg", err, "request id", value.RequestId)
		return err
	}

	slog.Info("process done SaveUser Usecase", "request id", value.RequestId)
	return nil
}
