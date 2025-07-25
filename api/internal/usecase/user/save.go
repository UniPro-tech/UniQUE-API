package usecase

import (
	"context"
	"log/slog"

	"github.com/UniPro-tech/UniQUE-API/api/internal/domain/user"
	"github.com/UniPro-tech/UniQUE-API/api/pkg"
)

type SaveUserUsecase struct {
	uds user.IUserDomainService
}

func NewSaveUserUsecase(uds user.IUserDomainService) *SaveUserUsecase {
	return &SaveUserUsecase{uds: uds}
}

func (us *SaveUserUsecase) Run(ctx context.Context, param *user.User) error {
	value, ok := ctx.Value("ctxInfo").(pkg.CtxInfo)
	if !ok {
		return INVALID_REQUEST_ID
	}

	if err := us.uds.SaveUser(ctx, param); err != nil {
		slog.Error("can not complete SaveUser Usecase", "error msg", err, "request id", value.RequestId)
		return err
	}

	slog.Info("process done SaveUser Usecase", "request id", value.RequestId, "user id", param.GetID())
	return nil
}
