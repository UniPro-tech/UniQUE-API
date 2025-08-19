package usecase

import (
	"context"
	"errors"
	"log/slog"

	"github.com/UniPro-tech/UniQUE-API/api/internal/domain/role"
	"github.com/UniPro-tech/UniQUE-API/api/pkg"
)

var (
	INVALID_SEARCH_PARAMS = errors.New("invalid search parameters")
)

type SearchRoleUsecase struct {
	rds role.IRoleDomainService
}

type SearchRoleUsecaseDtoModel struct {
	TotalCount int64                      `json:"total_count"`
	Pages      int                        `json:"pages"`
	Roles      []SearchRoleUsecaseDtoItem `json:"roles"`
}

type SearchRoleUsecaseDtoItem struct {
	ID            string   `json:"id,omitempty"`
	CustomID      string   `json:"custom_id,omitempty"`
	Name          string   `json:"name,omitempty"`
	Permissions   []string `json:"permissions,omitempty"`
	PermissionBit uint32   `json:"permission_bit,omitempty"`
	IsEnable      bool     `json:"is_enable,omitempty"`
	IsSystem      bool     `json:"is_system,omitempty"`
}

func NewSearchRoleUsecase(rds role.IRoleDomainService) *SearchRoleUsecase {
	return &SearchRoleUsecase{rds: rds}
}

func (us *SearchRoleUsecase) Run(ctx context.Context) (*SearchRoleUsecaseDtoModel, error) {
	value, ok := ctx.Value("ctxInfo").(pkg.CtxInfo)
	if !ok {
		return nil, INVALID_REQUEST_ID
	}

	searchParams, ok := ctx.Value("searchParams").(pkg.RoleParams)
	if !ok {
		return nil, INVALID_SEARCH_PARAMS
	}

	role, count, err := us.rds.SearchRole(ctx, searchParams)
	if err != nil {
		slog.Info("can not complete FindById usecase", "request id", value.RequestId)
		return nil, err
	}

	response := &SearchRoleUsecaseDtoModel{
		TotalCount: count,
		Pages:      1, // Assuming a single page for simplicity, adjust as needed
		Roles:      []SearchRoleUsecaseDtoItem{},
	}
	for _, r := range role {
		response.Roles = append(response.Roles, SearchRoleUsecaseDtoItem{
			ID:            r.GetID(),
			CustomID:      r.GetCustomID(),
			Name:          r.GetName(),
			Permissions:   r.GetPermissionArray(),
			PermissionBit: r.GetPermissionBits(),
			IsEnable:      r.GetIsEnable(),
			IsSystem:      r.GetIsSystem(),
		})
	}

	slog.Info("process done FindById usecase", "request id", value.RequestId, "role", role)
	return response, nil
}
