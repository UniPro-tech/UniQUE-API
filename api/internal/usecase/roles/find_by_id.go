package usecase

import (
	"context"
	"log/slog"

	"github.com/UniPro-tech/UniQUE-API/api/internal/domain/role"
	"github.com/UniPro-tech/UniQUE-API/api/pkg"
)

type FindRoleByIdUsecase struct {
	rds role.IRoleDomainService
}

type GetRoleByIdUsecaseDtoModel struct {
	ID            string   `json:"id,omitempty"`
	CustomID      string   `json:"custom_id,omitempty"`
	Name          string   `json:"name,omitempty"`
	Permissions   []string `json:"permissions,omitempty"`
	PermissionBit uint32   `json:"permission_bit,omitempty"`
	IsEnable      bool     `json:"is_enable,omitempty"`
	IsSystem      bool     `json:"is_system,omitempty"`
}

func NewFindRoleByIdUsecase(rds role.IRoleDomainService) *FindRoleByIdUsecase {
	return &FindRoleByIdUsecase{rds: rds}
}

func (us *FindRoleByIdUsecase) Run(ctx context.Context, id string) (*GetRoleByIdUsecaseDtoModel, error) {
	value, ok := ctx.Value("ctxInfo").(pkg.CtxInfo)
	if !ok {
		return nil, INVALID_REQUEST_ID
	}

	role, err := us.rds.FindRoleById(ctx, id)
	if err != nil {
		slog.Info("can not complete FindById usecase", "request id", value.RequestId)
		return nil, err
	}

	response := &GetRoleByIdUsecaseDtoModel{
		ID:            role.GetID(),
		CustomID:      role.GetCustomID(),
		Name:          role.GetName(),
		Permissions:   role.GetPermissionArray(),
		PermissionBit: role.GetPermissionBits(),
		IsEnable:      role.GetIsEnable(),
		IsSystem:      role.GetIsSystem(),
	}

	slog.Info("process done FindById usecase", "request id", value.RequestId, "role", role)
	return response, nil
}
