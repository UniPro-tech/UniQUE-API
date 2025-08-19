package roles

import (
	"context"
	"errors"
	"log/slog"
	"net/http"

	errorresponse "github.com/UniPro-tech/UniQUE-API/api/internal/controller/errorresponse"
	roleDomain "github.com/UniPro-tech/UniQUE-API/api/internal/domain/role"
	sqlerrors "github.com/UniPro-tech/UniQUE-API/api/internal/driver/mysql/errors"
	usecase "github.com/UniPro-tech/UniQUE-API/api/internal/usecase/role"
	"github.com/UniPro-tech/UniQUE-API/api/pkg"

	"github.com/gin-gonic/gin"
)

type RoleHandler struct {
	ListRoleUsecase     *usecase.ListRoleUsecase
	FindRoleByIdUsecase *usecase.FindRoleByIdUsecase
	SearchRoleUsecase   *usecase.SearchRoleUsecase
	AddRoleUsecase      *usecase.CreateRoleUsecase
	DeleteRoleUsecase   *usecase.DeleteRoleUsecase
	PutRoleUsecase      *usecase.PutRoleUsecase
	UpdateRoleUsecase   *usecase.UpdateRoleUsecase
}

func NewRolesHandler(
	listRoleUsecase *usecase.ListRoleUsecase,
	findRoleByIdUsecase *usecase.FindRoleByIdUsecase,
	searchRoleUsecase *usecase.SearchRoleUsecase,
	addRoleUsecase *usecase.CreateRoleUsecase,
	deleteRoleUsecase *usecase.DeleteRoleUsecase,
	putRoleUsecase *usecase.PutRoleUsecase,
	updateRoleUsecase *usecase.UpdateRoleUsecase,
) *RoleHandler {
	return &RoleHandler{
		ListRoleUsecase:     listRoleUsecase,
		FindRoleByIdUsecase: findRoleByIdUsecase,
		SearchRoleUsecase:   searchRoleUsecase,
		AddRoleUsecase:      addRoleUsecase,
		DeleteRoleUsecase:   deleteRoleUsecase,
		PutRoleUsecase:      putRoleUsecase,
		UpdateRoleUsecase:   updateRoleUsecase,
	}
}

// ListRoles godoc
// @Summary ロール一覧取得
// @Tags Roles
// @Accept json
// @Produce json
// @Success 200 {object} RolesResponse
// @Router /v1/roles [get]
func (h *RoleHandler) ListRoles(ctx *gin.Context) {
	// TODO: Authorization check
	limit := ctx.Query("limit")
	page := ctx.Query("page")
	request_id := ctx.GetHeader("X-Request-ID")

	reqCtx := context.WithValue(ctx, "ctxInfo", pkg.CtxInfo{PageLimit: limit, Pages: page, RequestId: request_id})
	res, err := h.ListRoleUsecase.Run(reqCtx)
	if err != nil {
		slog.Error("Failed to fetch role list", "error", err, "request_id", request_id)
		ctx.JSON(http.StatusInternalServerError, gin.H{"status": "Error"})
		return
	}
	roles := []RoleResponseModel{}
	for _, role := range res.Roles {
		roles = append(roles, RoleResponseModel{
			ID:         role.ID,
			CustomID:   role.CustomID,
			Name:       role.Name,
			Permission: role.PermissionBit,
			IsEnable:   role.IsEnable,
			IsSystem:   role.IsSystem,
		})
	}
	ctx.JSON(200, RolesResponse{
		TotalCount: res.TotalCount,
		Pages:      res.Pages,
		Roles:      roles,
	})
}

// GetUserById godoc
// @Summary ユーザーの詳細情報を取得
// @Tags Roles
// @Accept json
// @Produce json
// @Param request path string ture "ロールID"
// @Success 200 {object} RoleResponseModel
// @Router /v1/users/:id [get]
func (h *RoleHandler) GetRoleById(ctx *gin.Context) {
	roleId := ctx.Param("id")
	request_id := ctx.GetHeader("X-Request-ID")

	reqCtx := context.WithValue(ctx, "ctxInfo", pkg.CtxInfo{RequestId: request_id})
	res, err := h.FindRoleByIdUsecase.Run(reqCtx, roleId)
	if err != nil {
		slog.Error("can not process FindByID Usecase", "error msg", err, "request id", ctx.GetHeader("X-Request-ID"))
		ctx.JSON(http.StatusInternalServerError, Response{Status: "Internal Server Error"})
		return
	}

	if res.ID == "" {
		slog.Error("role not found", "request id", ctx.GetHeader("X-Request-ID"))
		ctx.JSON(http.StatusNotFound, Response{Status: "Role Not Found"})
		return
	}

	role := &RoleResponseModel{
		ID:         res.ID,
		CustomID:   res.CustomID,
		Name:       res.Name,
		Permission: res.PermissionBit,
		IsEnable:   res.IsEnable,
		IsSystem:   res.IsSystem,
	}
	slog.Info("process done FindByID Usecase", "request id", ctx.GetHeader("X-Request-ID"))
	ctx.JSON(http.StatusOK, role)
}

// SearchRoles godoc
// @Summary ロール検索
// @Description ロールの名前やカスタムIDで検索
// @Param name query string false "名前"
// @Param custom_id query string false "カスタムID"
// @Param is_enable query boolean false "有効フラグ"
// @Param is_system query boolean false "システムフラグ"
// @Param limit query string false "取得件数"
// @Param page query string false "ページ番号"
// @Tags Roles
// @Accept json
// @Produce json
// @Success 200 {object} RolesResponse
// @Router /v1/users [get]
func (h *RoleHandler) SearchRoles(ctx *gin.Context) {
	customID := ctx.Query("custom_id")
	name := ctx.Query("name")
	isEnable := ctx.Query("is_enable")
	isSystem := ctx.Query("is_system")
	id := ctx.Query("id")
	limit := ctx.Query("limit")
	page := ctx.Query("page")
	request_id := ctx.GetHeader("X-Request-ID")
	reqCtx := context.WithValue(context.WithValue(ctx, "ctxInfo", pkg.CtxInfo{
		PageLimit: limit,
		Pages:     page,
		RequestId: request_id,
	}), "searchParams", pkg.RoleParams{
		ID:       &id,
		CustomID: &customID,
		Name:     &name,
		IsEnable: &isEnable,
		IsSystem: &isSystem,
	})
	res, err := h.SearchRoleUsecase.Run(reqCtx)
	if err != nil {
		slog.Error("Failed to search roles", "error", err, "request_id", request_id)
		ctx.JSON(http.StatusInternalServerError, gin.H{"status": "Error"})
		return
	}
	roles := []RoleResponseModel{}
	for _, role := range res.Roles {
		roles = append(roles, RoleResponseModel{
			ID:         role.ID,
			CustomID:   role.CustomID,
			Name:       role.Name,
			Permission: role.PermissionBit,
			IsEnable:   role.IsEnable,
			IsSystem:   role.IsSystem,
		})
	}
	slog.Info("Search roles completed", "request_id", request_id)
	ctx.JSON(200, RolesResponse{
		TotalCount: res.TotalCount,
		Pages:      res.Pages,
		Roles:      roles,
	})
}

// RegisterRole godoc
// @Summary ロール情報を登録
// @Tags Roles
// @Accept json
// @Produce json
// @Param request body RoleRequestModel true "ロール情報"
// @Success 200 {object} Response
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /v1/users [post]
func (h *RoleHandler) RegisterRole(ctx *gin.Context) {
	request_id := ctx.GetHeader("X-Request-ID")
	param := &RoleRequestModel{}

	reqCtx := context.WithValue(ctx, "ctxInfo", pkg.CtxInfo{RequestId: request_id})
	err := ctx.ShouldBindJSON(&param)
	if err != nil {
		slog.Error("can not process SaveUser Usecase", "error msg", err, "request id", ctx.GetHeader("X-Request-ID"))
		ctx.JSON(http.StatusBadRequest, Response{Status: "Bad Request"})
		return
	}

	role := roleDomain.NewRole(param.ID, param.CustomID, param.Name, param.IsEnable, false, param.Permission)
	err = h.AddRoleUsecase.Run(reqCtx, role)
	if err != nil {
		if err == roleDomain.ERR_INVALID_CUSTOM_ID {
			slog.Error("Invalid role data", "error", err, "request_id", request_id)
			response := errorresponse.MissmatchedPatternError
			response.Message = "CustomID does not match the required pattern"
			ctx.JSON(http.StatusBadRequest, response)
			return
		}
		if errors.Is(err, sqlerrors.ERR_DUPLICATE_ENTRY) {
			slog.Error("Duplicate entry error", "error", err, "request_id", request_id)
			ctx.JSON(http.StatusConflict, errorresponse.AlreadyExistsError)
			return
		}
		slog.Error("Failed to save role", "error", err, "request_id", request_id)
		ctx.JSON(http.StatusInternalServerError, errorresponse.UnknownError)
		return
	}

	slog.Info("process done SaveRole Usecase", "request id", ctx.GetHeader("X-Request-ID"))
	ctx.JSON(http.StatusOK, gin.H{"status": "success"})
}

// DeleteRole godoc
// @Summary ロールを削除
// @Tags Roles
// @Accept json
// @Produce json
// @Param id path string true "ロールID"
// @Success 204 {object} nil
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /v1/roles/{id} [delete]
func (h *RoleHandler) DeleteRole(ctx *gin.Context) {
	roleId := ctx.Param("id")
	request_id := ctx.GetHeader("X-Request-ID")

	reqCtx := context.WithValue(ctx, "ctxInfo", pkg.CtxInfo{RequestId: request_id})
	err := h.DeleteRoleUsecase.Run(reqCtx, roleId)
	if err != nil {
		slog.Error("can not process DeleteRole Usecase", "error msg", err, "request id", request_id)
		ctx.JSON(http.StatusInternalServerError, Response{Status: "Internal Server Error"})
		return
	}

	slog.Info("process done DeleteRole Usecase", "request id", request_id)
	ctx.JSON(http.StatusNoContent, nil)
}

// PutRole godoc
// @Summary ロール情報を更新(置き換え)
// @Tags Roles
// @Accept json
// @Produce json
// @Param request body RoleRequestModel true "ロール情報"
// @Success 204 {object} nil
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /v1/roles/{id} [put]
func (h *RoleHandler) PutRole(ctx *gin.Context) {
	request_id := ctx.GetHeader("X-Request-ID")
	param := &RoleRequestModel{}

	reqCtx := context.WithValue(ctx, "ctxInfo", pkg.CtxInfo{RequestId: request_id})
	err := ctx.ShouldBindJSON(&param)
	if err != nil {
		slog.Error("can not process EditRole Usecase", "error msg", err, "request id", request_id)
		ctx.JSON(http.StatusBadRequest, Response{Status: "Bad Request"})
		return
	}

	role := roleDomain.NewRole(param.ID, param.CustomID, param.Name, param.IsEnable, false, param.Permission)
	err = h.PutRoleUsecase.Run(reqCtx, role)
	if err != nil {
		if err == roleDomain.ERR_INVALID_CUSTOM_ID {
			slog.Error("Invalid role data", "error", err, "request_id", request_id)
			response := errorresponse.MissmatchedPatternError
			response.Message = "CustomID does not match the required pattern"
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

// PatchRole godoc
// @Summary ロール情報を部分更新
// @Tags Roles
// @Accept json
// @Produce json
// @Param request body RoleRequestModel true "ロール情報"
// @Success 204 {object} nil
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /v1/roles [patch]
func (h *RoleHandler) PatchRole(ctx *gin.Context) {
	request_id := ctx.GetHeader("X-Request-ID")
	param := &RoleRequestModel{}

	reqCtx := context.WithValue(ctx, "ctxInfo", pkg.CtxInfo{RequestId: request_id})
	err := ctx.ShouldBindJSON(&param)
	if err != nil {
		slog.Error("can not process EditRole Usecase", "error msg", err, "request id", request_id)
		ctx.JSON(http.StatusBadRequest, Response{Status: "Bad Request"})
		return
	}

	role := roleDomain.NewRole(param.ID, param.CustomID, param.Name, param.IsEnable, false, param.Permission)
	err = h.UpdateRoleUsecase.Run(reqCtx, role)
	if err != nil {
		if err == roleDomain.ERR_INVALID_CUSTOM_ID {
			slog.Error("Invalid user data", "error", err, "request_id", request_id)
			response := errorresponse.MissmatchedPatternError
			response.Message = "CustomID does not match the required pattern"
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
