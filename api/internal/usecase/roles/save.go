package usecase

import (
	"context"
	"log/slog"

	"github.com/UniPro-tech/UniQUE-API/api/internal/domain/role"
	"github.com/UniPro-tech/UniQUE-API/api/pkg"
)

type PutRoleUsecase struct {
	rds role.IRoleDomainService
}

func NewPutRoleUsecase(rds role.IRoleDomainService) *PutRoleUsecase {
	return &PutRoleUsecase{rds: rds}
}

func (us *PutRoleUsecase) Run(ctx context.Context, param *role.Role) error {
	value, ok := ctx.Value("ctxInfo").(pkg.CtxInfo)
	if !ok {
		return INVALID_REQUEST_ID
	}

	if err := param.Valid(); err != nil {
		slog.Error("can not complete PutUser Usecase", "error msg", err, "request id", value.RequestId)
		return err
	}

	if err := us.rds.UpdateRole(ctx, param); err != nil {
		slog.Error("can not complete PutUser Usecase", "error msg", err, "request id", value.RequestId)
		return err
	}

	slog.Info("process done PutUser Usecase", "request id", value.RequestId, "user id", param.GetID())
	return nil
}
