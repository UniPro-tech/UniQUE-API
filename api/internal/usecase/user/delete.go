package usecase

import (
	"context"
	"log/slog"

	"github.com/UniPro-tech/UniQUE-API/api/internal/domain/user"
	"github.com/UniPro-tech/UniQUE-API/api/pkg"
)

type DeleteUserUsecase struct {
	uds user.IUserDomainService
}

func NewDeleteUserUsecase(uds user.IUserDomainService) *DeleteUserUsecase {
	return &DeleteUserUsecase{uds: uds}
}

func (us *DeleteUserUsecase) Run(ctx context.Context, id string) error {
	value, ok := ctx.Value("ctxInfo").(pkg.CtxInfo)
	if !ok {
		return INVALID_REQUEST_ID
	}

	if err := us.uds.DeleteUser(ctx, id); err != nil {
		slog.Error("can not complete DeleteUser Usecase", "error msg", err, "request id", value.RequestId)
		return err
	}

	slog.Info("process done DeleteUser Usecase", "request id", value.RequestId)
	return nil
}
