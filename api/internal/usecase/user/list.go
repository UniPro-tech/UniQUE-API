package usecase

import (
	"context"
	"errors"
	"log/slog"
	"strconv"

	"github.com/UniPro-tech/UniQUE-API/api/internal/domain/user"
	"github.com/UniPro-tech/UniQUE-API/api/pkg"
)

var (
	INVALID_USER_ID    = errors.New("invalid user id")
	INVALID_REQUEST_ID = errors.New("invalid request id")
)

type ListUserUsecase struct {
	uds user.IUserDomainService
}

type ListUserUsecaseDto struct {
	TotalCount int64                     `json:"total_count,omitempty"`
	Pages      int                       `json:"pages,omitempty"`
	Users      []ListUserUsecaseDtoModel `json:"user,omitempty"`
}

type ListUserUsecaseDtoModel struct {
	ID            string `json:"id,omitempty"`
	Email         string `json:"email,omitempty"`
	CustomID      string `json:"custom_id,omitempty"`
	Name          string `json:"name,omitempty"`
	Period        string `json:"period,omitempty"`
	ExternalEmail string `json:"external_email,omitempty"`
	IsEnable      bool   `json:"is_enable,omitempty"`
}

func NewFindUserUsecase(uds user.IUserDomainService) *ListUserUsecase {
	return &ListUserUsecase{uds: uds}
}

func (us *ListUserUsecase) Run(ctx context.Context) (*ListUserUsecaseDto, error) {
	// context.Contextの値を取り出す
	value, ok := ctx.Value("ctxInfo").(pkg.CtxInfo)
	if !ok {
		return nil, INVALID_USER_ID
	}

	users, count, err := us.uds.ListUser(ctx)
	if err != nil {
		slog.Error("can not process FindUser Usecase", "error msg", err, "request id", value.RequestId)
		return nil, err
	}

	dtouser := []ListUserUsecaseDtoModel{}
	dto := ListUserUsecaseDto{}
	for _, u := range users {
		r := ListUserUsecaseDtoModel{
			ID:            u.GetID(),
			Email:         u.GetEmail(),
			CustomID:      u.GetCustomID(),
			Name:          u.GetName(),
			Period:        u.GetPeriod(),
			IsEnable:      u.GetIsEnable(),
			ExternalEmail: u.GetExternalEmail(),
		}
		dtouser = append(dtouser, r)
	}
	limit, _ := strconv.Atoi(value.PageLimit)
	page, _ := strconv.Atoi(value.Pages)

	dto.TotalCount = count
	dto.Pages = limit + (page - 1)
	dto.Users = dtouser

	slog.Info("FindUserUsecase processing done ", "request_id", value.RequestId, "total_count", count, "pages", dto.Pages, "users", dtouser)
	return &dto, nil
}
