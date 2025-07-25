package usecase

import (
	"context"
	"log/slog"

	"github.com/UniPro-tech/UniQUE-API/api/internal/domain/user"
	"github.com/UniPro-tech/UniQUE-API/api/pkg"
)

type UpdateUserUsecase struct {
	uds user.IUserDomainService
}

func NewUpdateUserUsecase(uds user.IUserDomainService) *UpdateUserUsecase {
	return &UpdateUserUsecase{uds: uds}
}

func (us *UpdateUserUsecase) Run(ctx context.Context, param *user.User) error {
	value, ok := ctx.Value("ctxInfo").(pkg.CtxInfo)
	if !ok {
		return INVALID_REQUEST_ID
	}

	if err := us.uds.UpdateUser(ctx, param); err != nil {
		slog.Error("can not complete UpdateUser Usecase", "error msg", err, "request id", value.RequestId)
		return err
	}

	slog.Info("process done UpdateUser Usecase", "request id", value.RequestId, "user id", param.GetID())
	return nil
}
