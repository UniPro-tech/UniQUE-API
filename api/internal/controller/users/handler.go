package users

import (
	"context"
	"log/slog"
	"net/http"

	usecase "github.com/UniPro-tech/UniQUE-API/api/internal/usecase/user"
	"github.com/UniPro-tech/UniQUE-API/api/pkg"

	"github.com/gin-gonic/gin"
)

type UsersHandler struct {
	ListUserUsecase     *usecase.ListUserUsecase
	FindUserByIdUsecase *usecase.FindUserByIdUsecase
}

func NewUsersHandler(
	listUserUsecase *usecase.ListUserUsecase,
	findUserByIdUsecase *usecase.FindUserByIdUsecase,
) *UsersHandler {
	return &UsersHandler{
		ListUserUsecase:     listUserUsecase,
		FindUserByIdUsecase: findUserByIdUsecase,
	}
}

// HealthCheck godoc
// @Summary 死活監視用
// @Tags healthcheck
// @Accept json
// @Produce json
// @Success 200 {object} Response
// @Router /v1/health [get]
func (h *UsersHandler) ListUser(ctx *gin.Context) {
	// TODO: Authorization check
	limit := ctx.Query("limit")
	page := ctx.Query("page")
	userId := ctx.Query("userid")
	request_id := ctx.GetHeader("X-Request-ID")

	reqCtx := context.WithValue(ctx, "ctxInfo", pkg.CtxInfo{PageLimit: limit, Pages: page, UserId: userId, RequestId: request_id})
	res, err := h.ListUserUsecase.Run(reqCtx)
	if err != nil {
		slog.Error("Failed to fetch user list", "error", err, "request_id", request_id)
		ctx.JSON(http.StatusInternalServerError, gin.H{"status": "Error"})
		return
	}
	users := []UserResponseModel{}
	for _, user := range res.Users {
		users = append(users, UserResponseModel{
			ID:            user.ID,
			Email:         user.Email,
			CustomID:      user.CustomID,
			Name:          user.Name,
			ExternalEmail: user.ExternalEmail,
			Period:        user.Period,
			IsEnable:      user.IsEnable,
		})
	}
	ctx.JSON(200, UsersResponse{
		TotalCount: res.TotalCount,
		Pages:      res.Pages,
		Users:      users,
	})
}
