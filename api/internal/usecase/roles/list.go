package usecase

import (
	"context"
	"errors"
	"log/slog"
	"strconv"

	"github.com/UniPro-tech/UniQUE-API/api/internal/domain/role"
	"github.com/UniPro-tech/UniQUE-API/api/pkg"
)

var (
	INVALID_ROLE_ID    = errors.New("invalid role id")
	INVALID_REQUEST_ID = errors.New("invalid request id")
)

type ListRoleUsecase struct {
	rds role.IRoleDomainService
}

type ListRoleUsecaseDto struct {
	TotalCount int64                     `json:"total_count,omitempty"`
	Pages      int                       `json:"pages,omitempty"`
	Roles      []ListRoleUsecaseDtoModel `json:"roles,omitempty"`
}

type ListRoleUsecaseDtoModel struct {
	ID            string   `json:"id,omitempty"`
	CustomID      string   `json:"custom_id,omitempty"`
	Name          string   `json:"name,omitempty"`
	Permissions   []string `json:"permissions,omitempty"`
	PermissionBit uint32   `json:"permission,omitempty"`
	IsEnable      bool     `json:"is_enable,omitempty"`
	IsSystem      bool     `json:"is_system,omitempty"`
}

func NewListRoleUsecase(rds role.IRoleDomainService) *ListRoleUsecase {
	return &ListRoleUsecase{rds: rds}
}

func (us *ListRoleUsecase) Run(ctx context.Context) (*ListRoleUsecaseDto, error) {
	// context.Contextの値を取り出す
	value, ok := ctx.Value("ctxInfo").(pkg.CtxInfo)
	if !ok {
		return nil, INVALID_REQUEST_ID
	}

	roles, count, err := us.rds.ListRole(ctx)
	if err != nil {
		slog.Error("can not process FindUser Usecase", "error msg", err, "request id", value.RequestId)
		return nil, err
	}

	dtorole := []ListRoleUsecaseDtoModel{}
	dto := ListRoleUsecaseDto{}
	for _, r := range roles {
		r := ListRoleUsecaseDtoModel{
			ID:            r.GetID(),
			CustomID:      r.GetCustomID(),
			Name:          r.GetName(),
			Permissions:   r.GetPermissionArray(),
			PermissionBit: r.GetPermissionBits(),
			IsEnable:      r.GetIsEnable(),
			IsSystem:      r.GetIsSystem(),
		}
		dtorole = append(dtorole, r)
	}
	limit, _ := strconv.Atoi(value.PageLimit)
	page, _ := strconv.Atoi(value.Pages)

	dto.TotalCount = count
	dto.Pages = limit + (page - 1)
	dto.Roles = dtorole

	slog.Info("FindUserUsecase processing done ", "request_id", value.RequestId, "total_count", count, "pages", dto.Pages, "roles", dtorole)
	return &dto, nil
}
