package usecase

import (
	"context"
	"errors"
	"log/slog"

	"github.com/UniPro-tech/UniQUE-API/api/internal/domain/user"
	"github.com/UniPro-tech/UniQUE-API/api/pkg"
)

var (
	INVALID_SEARCH_PARAMS = errors.New("invalid search parameters")
)

type SearchUserUsecase struct {
	uds user.IUserDomainService
}

type SearchUsecaseDtoModel struct {
	TotalCount int64                       `json:"total_count"`
	Pages      int                         `json:"pages"`
	Users      []SearchUserUsecaseDtoModel `json:"users"`
}

type SearchUserUsecaseDtoModel struct {
	ID            string `json:"id,omitempty"`
	Email         string `json:"email,omitempty"`
	CustomID      string `json:"custom_id,omitempty"`
	Name          string `json:"name,omitempty"`
	Period        string `json:"period,omitempty"`
	ExternalEmail string `json:"external_email,omitempty"`
	IsEnable      bool   `json:"is_enable,omitempty"`
}

func NewSearchUserUsecase(uds user.IUserDomainService) *SearchUserUsecase {
	return &SearchUserUsecase{uds: uds}
}

func (us *SearchUserUsecase) Run(ctx context.Context) (*SearchUsecaseDtoModel, error) {
	value, ok := ctx.Value("ctxInfo").(pkg.CtxInfo)
	if !ok {
		return nil, INVALID_REQUEST_ID
	}

	searchParams, ok := ctx.Value("searchParams").(pkg.UserParams)
	if !ok {
		return nil, INVALID_SEARCH_PARAMS
	}

	user, _, err := us.uds.SearchUser(ctx, searchParams)
	if err != nil {
		slog.Info("can not complete FindById usecase", "request id", value.RequestId)
		return nil, err
	}

	response := &SearchUsecaseDtoModel{
		TotalCount: int64(len(user)),
		Pages:      1, // Assuming a single page for simplicity, adjust as needed
		Users:      []SearchUserUsecaseDtoModel{},
	}
	for _, u := range user {
		response.Users = append(response.Users, SearchUserUsecaseDtoModel{
			ID:            u.GetID(),
			Email:         u.GetEmail(),
			CustomID:      u.GetCustomID(),
			Name:          u.GetName(),
			Period:        u.GetPeriod(),
			ExternalEmail: u.GetExternalEmail(),
			IsEnable:      u.GetIsEnable(),
		})
	}

	slog.Info("process done FindById usecase", "request id", value.RequestId, "user", user)
	return response, nil
}
