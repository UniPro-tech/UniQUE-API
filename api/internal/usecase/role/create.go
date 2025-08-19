package usecase

import (
	"context"
	"log/slog"

	"github.com/UniPro-tech/UniQUE-API/api/internal/domain/role"
	"github.com/UniPro-tech/UniQUE-API/api/pkg"
)

type CreateRoleUsecase struct {
	rds role.IRoleDomainService
}

func NewCreateRoleUsecase(rds role.IRoleDomainService) *CreateRoleUsecase {
	return &CreateRoleUsecase{rds: rds}
}

func (us *CreateRoleUsecase) Run(ctx context.Context, param *role.Role) error {
	value, ok := ctx.Value("ctxInfo").(pkg.CtxInfo)
	if !ok {
		return INVALID_REQUEST_ID
	}

	if err := us.rds.AddRole(ctx, param); err != nil {
		slog.Error("can not complete SaveRole Usecase", "error msg", err, "request id", value.RequestId)
		return err
	}

	slog.Info("process done SaveRole Usecase", "request id", value.RequestId)
	return nil
}
