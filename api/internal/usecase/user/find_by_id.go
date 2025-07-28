package usecase

import (
	"context"
	"log/slog"

	"github.com/UniPro-tech/UniQUE-API/api/internal/domain/user"
	"github.com/UniPro-tech/UniQUE-API/api/pkg"
)

type FindUserByIdUsecase struct {
	uds user.IUserDomainService
}

type GetUserByIdUsecaseDtoModel struct {
	ID            string `json:"id,omitempty"`
	Email         string `json:"email,omitempty"`
	CustomID      string `json:"custom_id,omitempty"`
	Name          string `json:"name,omitempty"`
	Period        string `json:"period,omitempty"`
	ExternalEmail string `json:"external_email,omitempty"`
	IsEnable      bool   `json:"is_enable,omitempty"`
	CreatedAt     string `json:"created_at,omitempty"`
	UpdatedAt     string `json:"updated_at,omitempty"`
	JoinedAt      string `json:"joined_at,omitempty"`
}

func NewFindUserByIdUsecase(uds user.IUserDomainService) *FindUserByIdUsecase {
	return &FindUserByIdUsecase{uds: uds}
}

func (us *FindUserByIdUsecase) Run(ctx context.Context, id string) (*GetUserByIdUsecaseDtoModel, error) {
	value, ok := ctx.Value("ctxInfo").(pkg.CtxInfo)
	if !ok {
		return nil, INVALID_REQUEST_ID
	}

	user, err := us.uds.FindUserById(ctx, id)
	if err != nil {
		slog.Info("can not complete FindById usecase", "request id", value.RequestId)
		return nil, err
	}

	response := &GetUserByIdUsecaseDtoModel{
		ID:            user.GetID(),
		Email:         user.GetEmail(),
		CustomID:      user.GetCustomID(),
		Name:          user.GetName(),
		Period:        user.GetPeriod(),
		ExternalEmail: user.GetExternalEmail(),
		IsEnable:      user.GetIsEnable(),
	}

	slog.Info("process done FindById usecase", "request id", value.RequestId, "user", user)
	return response, nil
}
