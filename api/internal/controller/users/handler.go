package users

import (
	"context"
	"log/slog"
	"net/http"

	usecase "github.com/UniPro-tech/UniQUE-API/api/internal/usecase/user"
	"github.com/UniPro-tech/UniQUE-API/api/pkg"

	"github.com/gin-gonic/gin"
)

type UserHandler struct {
	ListUserUsecase     *usecase.ListUserUsecase
	FindUserByIdUsecase *usecase.FindUserByIdUsecase
}

func NewUsersHandler(
	listUserUsecase *usecase.ListUserUsecase,
	findUserByIdUsecase *usecase.FindUserByIdUsecase,
) *UserHandler {
	return &UserHandler{
		ListUserUsecase:     listUserUsecase,
		FindUserByIdUsecase: findUserByIdUsecase,
	}
}

// ListUsers godoc
// @Summary ユーザー一覧取得
// @Tags users
// @Accept json
// @Produce json
// @Success 200 {object} UsersResponse
// @Router /v1/users [get]
func (h *UserHandler) ListUser(ctx *gin.Context) {
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

// GetUserById godoc
// @Summary ユーザーの詳細情報を取得
// @Tags Users
// @Accept json
// @Produce json
// @Param request path string ture "ユーザーID"
// @Success 200 {object} UserResponseModel
// @Router /v1/users/:id [get]
func (h *UserHandler) GetUserById(ctx *gin.Context) {
	userId := ctx.Param("id")
	request_id := ctx.GetHeader("X-Request-ID")

	reqCtx := context.WithValue(ctx, "ctxInfo", pkg.CtxInfo{RequestId: request_id})
	res, err := h.FindUserByIdUsecase.Run(reqCtx, userId)
	if err != nil {
		slog.Error("can not process FindByID Usecase", "error msg", err, "request id", ctx.GetHeader("X-Request-ID"))
		ctx.JSON(http.StatusInternalServerError, Response{Status: "Internal Server Error"})
		return
	}

	if res.ID == "" {
		slog.Error("user not found", "request id", ctx.GetHeader("X-Request-ID"))
		ctx.JSON(http.StatusNotFound, Response{Status: "User Not Found"})
		return
	}

	user := &UserResponseModel{
		ID:            res.ID,
		Email:         res.Email,
		CustomID:      res.CustomID,
		Name:          res.Name,
		ExternalEmail: res.ExternalEmail,
		Period:        res.Period,
		IsEnable:      res.IsEnable,
	}
	slog.Info("process done FindByID Usecase", "request id", ctx.GetHeader("X-Request-ID"))
	ctx.JSON(http.StatusOK, user)
}
