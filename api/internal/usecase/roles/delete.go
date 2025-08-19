package usecase

import (
	"context"
	"log/slog"

	"github.com/UniPro-tech/UniQUE-API/api/internal/domain/role"
	"github.com/UniPro-tech/UniQUE-API/api/pkg"
)

type DeleteRoleUsecase struct {
	rds role.IRoleDomainService
}

func NewDeleteRoleUsecase(rds role.IRoleDomainService) *DeleteRoleUsecase {
	return &DeleteRoleUsecase{rds: rds}
}

func (us *DeleteRoleUsecase) Run(ctx context.Context, id string) error {
	value, ok := ctx.Value("ctxInfo").(pkg.CtxInfo)
	if !ok {
		return INVALID_REQUEST_ID
	}

	if err := us.rds.DeleteRole(ctx, id); err != nil {
		slog.Error("can not complete DeleteRole Usecase", "error msg", err, "request id", value.RequestId)
		return err
	}

	slog.Info("process done DeleteRole Usecase", "request id", value.RequestId)
	return nil
}
