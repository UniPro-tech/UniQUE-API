package users

import (
	"context"
	"errors"
	"log/slog"
	"net/http"

	errorresponse "github.com/UniPro-tech/UniQUE-API/api/internal/controller/errorresponse"
	userDomain "github.com/UniPro-tech/UniQUE-API/api/internal/domain/user"
	sqlerrors "github.com/UniPro-tech/UniQUE-API/api/internal/driver/mysql/errors"
	usecase "github.com/UniPro-tech/UniQUE-API/api/internal/usecase/user"
	"github.com/UniPro-tech/UniQUE-API/api/pkg"

	"github.com/gin-gonic/gin"
)

type UserHandler struct {
	ListUserUsecase     *usecase.ListUserUsecase
	FindUserByIdUsecase *usecase.FindUserByIdUsecase
	SearchUserUsecase   *usecase.SearchUsecase
	AddUserUsecase      *usecase.CreateUserUsecase
	DeleteUserUsecase   *usecase.DeleteUserUsecase
	PutUserUsecase      *usecase.PutUserUsecase
	UpdateUserUsecase   *usecase.UpdateUserUsecase
}

func NewUsersHandler(
	listUserUsecase *usecase.ListUserUsecase,
	findUserByIdUsecase *usecase.FindUserByIdUsecase,
	searchUserUsecase *usecase.SearchUsecase,
	addUserUsecase *usecase.CreateUserUsecase,
	deleteUserUsecase *usecase.DeleteUserUsecase,
	putUserUsecase *usecase.PutUserUsecase,
	updateUserUsecase *usecase.UpdateUserUsecase,
) *UserHandler {
	return &UserHandler{
		ListUserUsecase:     listUserUsecase,
		FindUserByIdUsecase: findUserByIdUsecase,
		SearchUserUsecase:   searchUserUsecase,
		AddUserUsecase:      addUserUsecase,
		DeleteUserUsecase:   deleteUserUsecase,
		PutUserUsecase:      putUserUsecase,
		UpdateUserUsecase:   updateUserUsecase,
	}
}

// ListUsers godoc
// @Summary ユーザー一覧取得
// @Tags Users
// @Accept json
// @Produce json
// @Success 200 {object} UsersResponse
// @Router /v1/users [get]
func (h *UserHandler) ListUser(ctx *gin.Context) {
	// TODO: Authorization check
	limit := ctx.Query("limit")
	page := ctx.Query("page")
	request_id := ctx.GetHeader("X-Request-ID")

	reqCtx := context.WithValue(ctx, "ctxInfo", pkg.CtxInfo{PageLimit: limit, Pages: page, RequestId: request_id})
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

// SearchUsers godoc
// @Summary ユーザー検索
// @Description ユーザーのメールアドレスやカスタムIDで検索
// @Param email query string false "メールアドレス"
// @Param custom_id query string false "カスタムID"
// @Param name query string false "名前"
// @Param external_email query string false "外部メールアドレス"
// @Param period query string false "期間"
// @Param is_enable query boolean false "有効フラグ"
// @Param limit query string false "取得件数"
// @Param page query string false "ページ番号"
// @Tags Users
// @Accept json
// @Produce json
// @Success 200 {object} UsersResponse
// @Router /v1/users [get]
func (h *UserHandler) SearchUsers(ctx *gin.Context) {
	email := ctx.Query("email")
	customID := ctx.Query("custom_id")
	name := ctx.Query("name")
	externalEmail := ctx.Query("external_email")
	period := ctx.Query("period")
	isEnable := ctx.Query("is_enable")
	id := ctx.Query("id")
	limit := ctx.Query("limit")
	page := ctx.Query("page")
	request_id := ctx.GetHeader("X-Request-ID")
	reqCtx := context.WithValue(context.WithValue(ctx, "ctxInfo", pkg.CtxInfo{
		PageLimit: limit,
		Pages:     page,
		RequestId: request_id,
	}), "searchParams", pkg.UserParams{
		ID:            &id,
		Email:         &email,
		CustomID:      &customID,
		Name:          &name,
		ExternalEmail: &externalEmail,
		Period:        &period,
		IsEnable:      &isEnable,
	})
	res, err := h.SearchUserUsecase.Run(reqCtx)
	if err != nil {
		slog.Error("Failed to search users", "error", err, "request_id", request_id)
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
	slog.Info("Search users completed", "request_id", request_id)
	ctx.JSON(200, UsersResponse{
		TotalCount: res.TotalCount,
		Pages:      res.Pages,
		Users:      users,
	})
}

// RegisterUser godoc
// @Summary ユーザー情報を登録
// @Tags Users
// @Accept json
// @Produce json
// @Param request body UserRequestModel true "ユーザー情報"
// @Success 200 {object} Response
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /v1/users [post]
func (h *UserHandler) RegisterUser(ctx *gin.Context) {
	request_id := ctx.GetHeader("X-Request-ID")
	param := &UserRequestModel{}

	reqCtx := context.WithValue(ctx, "ctxInfo", pkg.CtxInfo{RequestId: request_id})
	err := ctx.ShouldBindJSON(&param)
	if err != nil {
		slog.Error("can not process SaveUser Usecase", "error msg", err, "request id", ctx.GetHeader("X-Request-ID"))
		ctx.JSON(http.StatusBadRequest, Response{Status: "Bad Request"})
		return
	}

	user := userDomain.NewUser(param.ID, param.Name, param.Email, param.CustomID, param.ExternalEmail, param.Period, param.IsEnable, &param.PasswordHash)
	err = h.AddUserUsecase.Run(reqCtx, user)
	if err != nil {
		if err == userDomain.ERR_INVALID_CUSTOM_ID {
			slog.Error("Invalid user data", "error", err, "request_id", request_id)
			response := errorresponse.MissmatchedPatternError
			response.Message = "CustomID does not match the required pattern"
			ctx.JSON(http.StatusBadRequest, response)
			return
		}
		if err == userDomain.ERR_INVALID_EMAIL {
			slog.Error("Invalid user data", "error", err, "request_id", request_id)
			response := errorresponse.MissmatchedPatternError
			response.Message = "Email does not match the required pattern"
			slog.Info(response.Message)
			ctx.JSON(http.StatusBadRequest, response)
			return
		}
		if err == userDomain.ERR_INVALID_EXTERNAL_EMAIL {
			slog.Error("Invalid user data", "error", err, "request_id", request_id)
			response := errorresponse.MissmatchedPatternError
			response.Message = "ExternalEmail does not match the required pattern"
			ctx.JSON(http.StatusBadRequest, response)
			return
		}
		if errors.Is(err, sqlerrors.ERR_DUPLICATE_ENTRY) {
			slog.Error("Duplicate entry error", "error", err, "request_id", request_id)
			ctx.JSON(http.StatusConflict, errorresponse.AlreadyExistsError)
			return
		}
		slog.Error("Failed to save user", "error", err, "request_id", request_id)
		ctx.JSON(http.StatusInternalServerError, errorresponse.UnknownError)
		return
	}

	slog.Info("process done SaveUser Usecase", "request id", ctx.GetHeader("X-Request-ID"))
	ctx.JSON(http.StatusOK, gin.H{"status": "success"})
}

// DeleteUser godoc
// @Summary ユーザーを削除
// @Tags Users
// @Accept json
// @Produce json
// @Param id path string true "ユーザーID"
// @Success 204 {object} nil
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /v1/users/{id} [delete]
func (h *UserHandler) DeleteUser(ctx *gin.Context) {
	userId := ctx.Param("id")
	request_id := ctx.GetHeader("X-Request-ID")

	reqCtx := context.WithValue(ctx, "ctxInfo", pkg.CtxInfo{RequestId: request_id})
	err := h.DeleteUserUsecase.Run(reqCtx, userId)
	if err != nil {
		slog.Error("can not process DeleteUser Usecase", "error msg", err, "request id", request_id)
		ctx.JSON(http.StatusInternalServerError, Response{Status: "Internal Server Error"})
		return
	}

	slog.Info("process done DeleteUser Usecase", "request id", request_id)
	ctx.JSON(http.StatusNoContent, nil)
}

// PutUser godoc
// @Summary ユーザー情報を更新(置き換え)
// @Tags Users
// @Accept json
// @Produce json
// @Param request body UserRequestModel true "ユーザー情報"
// @Success 204 {object} nil
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /v1/users [put]
func (h *UserHandler) PutUser(ctx *gin.Context) {
	request_id := ctx.GetHeader("X-Request-ID")
	param := &UserRequestModel{}

	reqCtx := context.WithValue(ctx, "ctxInfo", pkg.CtxInfo{RequestId: request_id})
	err := ctx.ShouldBindJSON(&param)
	if err != nil {
		slog.Error("can not process EditUser Usecase", "error msg", err, "request id", request_id)
		ctx.JSON(http.StatusBadRequest, Response{Status: "Bad Request"})
		return
	}

	user := userDomain.NewUser(param.ID, param.Name, param.Email, param.CustomID, param.ExternalEmail, param.Period, param.IsEnable, &param.PasswordHash)
	err = h.PutUserUsecase.Run(reqCtx, user)
	if err != nil {
		if err == userDomain.ERR_INVALID_CUSTOM_ID {
			slog.Error("Invalid user data", "error", err, "request_id", request_id)
			response := errorresponse.MissmatchedPatternError
			response.Message = "CustomID does not match the required pattern"
			ctx.JSON(http.StatusBadRequest, response)
			return
		}
		if err == userDomain.ERR_INVALID_EMAIL {
			slog.Error("Invalid user data", "error", err, "request_id", request_id)
			response := errorresponse.MissmatchedPatternError
			response.Message = "Email does not match the required pattern"
			slog.Info(response.Message)
			ctx.JSON(http.StatusBadRequest, response)
			return
		}
		if err == userDomain.ERR_INVALID_EXTERNAL_EMAIL {
			slog.Error("Invalid user data", "error", err, "request_id", request_id)
			response := errorresponse.MissmatchedPatternError
			response.Message = "ExternalEmail does not match the required pattern"
			ctx.JSON(http.StatusBadRequest, response)
			return
		}
		if errors.Is(err, sqlerrors.ERR_DUPLICATE_ENTRY) {
			slog.Error("Duplicate entry error", "error", err, "request_id", request_id)
			ctx.JSON(http.StatusConflict, errorresponse.AlreadyExistsError)
			return
		}
		slog.Error("Failed to save user", "error", err, "request_id", request_id)
		ctx.JSON(http.StatusInternalServerError, errorresponse.UnknownError)
		return
	}

	slog.Info("process done EditUser Usecase", "request id", request_id)
	ctx.JSON(http.StatusNoContent, nil)
}

// PatchUser godoc
// @Summary ユーザー情報を部分更新
// @Tags Users
// @Accept json
// @Produce json
// @Param request body UserRequestModel true "ユーザー情報"
// @Success 204 {object} nil
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /v1/users [patch]
func (h *UserHandler) PatchUser(ctx *gin.Context) {
	request_id := ctx.GetHeader("X-Request-ID")
	param := &UserRequestModel{}

	reqCtx := context.WithValue(ctx, "ctxInfo", pkg.CtxInfo{RequestId: request_id})
	err := ctx.ShouldBindJSON(&param)
	if err != nil {
		slog.Error("can not process EditUser Usecase", "error msg", err, "request id", request_id)
		ctx.JSON(http.StatusBadRequest, Response{Status: "Bad Request"})
		return
	}

	user := userDomain.NewUser(param.ID, param.Name, param.Email, param.CustomID, param.ExternalEmail, param.Period, param.IsEnable, &param.PasswordHash)
	err = h.UpdateUserUsecase.Run(reqCtx, user)
	if err != nil {
		if err == userDomain.ERR_INVALID_CUSTOM_ID {
			slog.Error("Invalid user data", "error", err, "request_id", request_id)
			response := errorresponse.MissmatchedPatternError
			response.Message = "CustomID does not match the required pattern"
			ctx.JSON(http.StatusBadRequest, response)
			return
		}
		if err == userDomain.ERR_INVALID_EMAIL {
			slog.Error("Invalid user data", "error", err, "request_id", request_id)
			response := errorresponse.MissmatchedPatternError
			response.Message = "Email does not match the required pattern"
			slog.Info(response.Message)
			ctx.JSON(http.StatusBadRequest, response)
			return
		}
		if err == userDomain.ERR_INVALID_EXTERNAL_EMAIL {
			slog.Error("Invalid user data", "error", err, "request_id", request_id)
			response := errorresponse.MissmatchedPatternError
			response.Message = "ExternalEmail does not match the required pattern"
			ctx.JSON(http.StatusBadRequest, response)
			return
		}
		if errors.Is(err, sqlerrors.ERR_DUPLICATE_ENTRY) {
			slog.Error("Duplicate entry error", "error", err, "request_id", request_id)
			ctx.JSON(http.StatusConflict, errorresponse.AlreadyExistsError)
			return
		}
		slog.Error("Failed to save user", "error", err, "request_id", request_id)
		ctx.JSON(http.StatusInternalServerError, errorresponse.UnknownError)
		return
	}

	slog.Info("process done EditUser Usecase", "request id", request_id)
	ctx.JSON(http.StatusNoContent, nil)
}
