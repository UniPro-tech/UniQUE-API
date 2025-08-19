package usecase

import (
	"context"
	"log/slog"

	"github.com/UniPro-tech/UniQUE-API/api/internal/domain/role"
	"github.com/UniPro-tech/UniQUE-API/api/pkg"
)

type UpdateRoleUsecase struct {
	rds role.IRoleDomainService
}

func NewUpdateRoleUsecase(rds role.IRoleDomainService) *UpdateRoleUsecase {
	return &UpdateRoleUsecase{rds: rds}
}

func (us *UpdateRoleUsecase) Run(ctx context.Context, param *role.Role) error {
	value, ok := ctx.Value("ctxInfo").(pkg.CtxInfo)
	if !ok {
		return INVALID_REQUEST_ID
	}

	if err := us.rds.UpdateRole(ctx, param); err != nil {
		slog.Error("can not complete UpdateRole Usecase", "error msg", err, "request id", value.RequestId)
		return err
	}

	slog.Info("process done UpdateRole Usecase", "request id", value.RequestId, "role id", param.GetID())
	return nil
}
