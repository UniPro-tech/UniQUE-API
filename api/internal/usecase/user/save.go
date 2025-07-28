package usecase

import (
	"context"
	"log/slog"

	"github.com/UniPro-tech/UniQUE-API/api/internal/domain/user"
	"github.com/UniPro-tech/UniQUE-API/api/pkg"
)

type PutUserUsecase struct {
	uds user.IUserDomainService
}

func NewPutUserUsecase(uds user.IUserDomainService) *PutUserUsecase {
	return &PutUserUsecase{uds: uds}
}

func (us *PutUserUsecase) Run(ctx context.Context, param *user.User) error {
	value, ok := ctx.Value("ctxInfo").(pkg.CtxInfo)
	if !ok {
		return INVALID_REQUEST_ID
	}

	if err := param.Valid(); err != nil {
		slog.Error("can not complete PutUser Usecase", "error msg", err, "request id", value.RequestId)
		return err
	}

	if err := us.uds.UpdateUser(ctx, param); err != nil {
		slog.Error("can not complete PutUser Usecase", "error msg", err, "request id", value.RequestId)
		return err
	}

	slog.Info("process done PutUser Usecase", "request id", value.RequestId, "user id", param.GetID())
	return nil
}
