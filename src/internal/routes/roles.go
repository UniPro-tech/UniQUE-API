package routes

import (
	"net/http"
	"strings"
	"time"

	"github.com/UniPro-tech/UniQUE-API/internal/constants"
	"github.com/UniPro-tech/UniQUE-API/internal/middleware"
	"github.com/UniPro-tech/UniQUE-API/internal/model"
	"github.com/UniPro-tech/UniQUE-API/internal/query"
	"github.com/gin-gonic/gin"
	"github.com/go-sql-driver/mysql"
	"github.com/oklog/ulid/v2"
)

func RegisterRoleRoutes(r *gin.Engine) {
	g := r.Group("/roles")
	{
		// ロール一覧・詳細の取得はROLE_MANAGE権限が必要
		g.GET("", middleware.RequirePermission(constants.ROLE_MANAGE), listRoles)
		g.GET(":id", middleware.RequirePermission(constants.ROLE_MANAGE), getRole)
		g.GET(":id/users", middleware.RequirePermission(constants.ROLE_MANAGE), listUsersForRole)

		// ロールの作成・更新・削除はROLE_MANAGE権限が必要
		g.POST("", middleware.RequirePermission(constants.ROLE_MANAGE), createRole)
		g.PUT(":id", middleware.RequirePermission(constants.ROLE_MANAGE), updateRole)
		g.DELETE(":id", middleware.RequirePermission(constants.ROLE_MANAGE), deleteRole)
	}
}

// listUsersForRole godoc
// @Summary List users for a role
// @Description Get users assigned to a role
// @Tags roles
// @Produce json
// @Param id path string true "Role ID"
// @Success 200 {object} routes.UserListResponse
// @Router /roles/{id}/users [get]
func listUsersForRole(c *gin.Context) {
	db := getDB(c)
	if db == nil {
		return
	}
	id := c.Param("id")
	q := query.Use(db)
	// Use JOIN to fetch users with optional profile in a single query.
	type row struct {
		ID                string     `json:"id"`
		CustomID          string     `json:"custom_id"`
		Email             string     `json:"email"`
		ExternalEmail     string     `json:"external_email"`
		EmailVerified     bool       `json:"email_verified"`
		AffiliationPeriod string     `json:"affiliation_period"`
		Status            string     `json:"status"`
		CreatedAt         time.Time  `json:"created_at"`
		UpdatedAt         time.Time  `json:"updated_at"`
		DisplayName       *string    `json:"display_name"`
		Bio               *string    `json:"bio"`
		WebsiteURL        *string    `json:"website_url"`
		JoinedAt          *time.Time `json:"joined_at"`
	}

	// Batch fetch via user_roles -> collect user IDs -> IN queries to avoid N+1
	urs, err := q.UserRole.Where(query.UserRole.RoleID.Eq(id)).Find()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if len(urs) == 0 {
		c.JSON(http.StatusOK, UserListResponse{Data: []UserDTO{}})
		return
	}
	ids := make([]string, 0, len(urs))
	for _, ur := range urs {
		ids = append(ids, ur.UserID)
	}
	users, err := q.User.Where(query.User.ID.In(ids...)).Find()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	profiles, _ := q.Profile.Where(query.Profile.UserID.In(ids...)).Find()
	profileMap := make(map[string]*model.Profile)
	for _, p := range profiles {
		profileMap[p.UserID] = p
	}
	var out []UserDTO
	for _, u := range users {
		dto := UserDTO{
			ID:                u.ID,
			CustomID:          u.CustomID,
			Email:             u.Email,
			ExternalEmail:     u.ExternalEmail,
			EmailVerified:     u.EmailVerified,
			AffiliationPeriod: ptrToString(u.AffiliationPeriod),
			Status:            u.Status,
			CreatedAt:         u.CreatedAt,
			UpdatedAt:         u.UpdatedAt,
		}
		if p, ok := profileMap[u.ID]; ok {
			dto.Profile = &ProfileDTO{
				UserID:           p.UserID,
				DisplayName:      p.DisplayName,
				Bio:              ptrToString(p.Bio),
				WebsiteURL:       ptrToString(p.WebsiteURL),
				TwitterHandle:    ptrToString(p.TwitterHandle),
				Birthdate:        formatBirthdate(p.Birthdate),
				BirthdateVisible: &p.BirthdateVisible,
				JoinedAt:         timeToTime(p.JoinedAt),
			}
		}
		out = append(out, dto)
	}
	c.JSON(http.StatusOK, UserListResponse{Data: out})
}

// listRoles godoc
// @Summary List roles
// @Description List roles
// @Tags roles
// @Produce json
// @Success 200 {object} routes.RoleListResponse
// @Router /roles [get]
func listRoles(c *gin.Context) {
	db := getDB(c)
	if db == nil {
		return
	}
	q := query.Use(db)
	roles, err := q.Role.Find()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	var out []RoleDTO
	for _, r := range roles {
		out = append(out, RoleDTO{
			ID:                r.ID,
			CustomID:          r.CustomID,
			Name:              r.Name,
			Description:       ptrToString(r.Description),
			PermissionBitmask: r.PermissionBitmask,
		})
	}
	c.JSON(http.StatusOK, RoleListResponse{Data: out})
}

// createRole godoc
// @Summary Create a role
// @Description Create a new role
// @Tags roles
// @Accept json
// @Produce json
// @Param role body routes.CreateRoleRequest true "Create role"
// @Success 201 {object} routes.RoleDTO
// @Router /roles [post]
func createRole(c *gin.Context) {
	db := getDB(c)
	if db == nil {
		return
	}
	var input CreateRoleRequest
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	role := model.Role{
		ID:                ulid.Make().String(),
		CustomID:          input.CustomID,
		Name:              input.Name,
		Description:       stringToPtr(input.Description),
		PermissionBitmask: input.PermissionBitmask,
	}
	q := query.Use(db)
	if err := q.Role.Create(&role); err != nil {
		// MySQLの重複エラーをチェック
		if mysqlErr, ok := err.(*mysql.MySQLError); ok && mysqlErr.Number == 1062 {
			// エラーメッセージから重複したキーを判定
			errMsg := mysqlErr.Message
			if strings.Contains(errMsg, "custom_id") || strings.Contains(errMsg, "name") {
				c.JSON(http.StatusConflict, gin.H{"error": "role already exists", "code": "R0002"})
				return
			}
			// その他の重複エラー
			c.JSON(http.StatusConflict, gin.H{"error": "duplicate entry", "code": "R0002"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	resp := RoleDTO{
		ID:                role.ID,
		CustomID:          role.CustomID,
		Name:              role.Name,
		Description:       ptrToString(role.Description),
		PermissionBitmask: role.PermissionBitmask,
	}
	c.JSON(http.StatusCreated, resp)
}

// getRole godoc
// @Summary Get a role
// @Description Get a single role
// @Tags roles
// @Produce json
// @Param id path string true "Role ID"
// @Success 200 {object} routes.RoleDTO
// @Router /roles/{id} [get]
func getRole(c *gin.Context) {
	db := getDB(c)
	if db == nil {
		return
	}
	id := c.Param("id")
	q := query.Use(db)
	r, err := q.Role.Where(query.Role.ID.Eq(id)).First()
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
		return
	}
	resp := RoleDTO{
		ID:                r.ID,
		CustomID:          r.CustomID,
		Name:              r.Name,
		Description:       ptrToString(r.Description),
		PermissionBitmask: r.PermissionBitmask,
	}
	c.JSON(http.StatusOK, resp)
}

// updateRole godoc
// @Summary Update a role
// @Description Update role fields
// @Tags roles
// @Accept json
// @Produce json
// @Param id path string true "Role ID"
// @Param role body routes.UpdateRoleRequest true "Update role"
// @Success 200 {object} routes.RoleDTO
// @Router /roles/{id} [put]
func updateRole(c *gin.Context) {
	db := getDB(c)
	if db == nil {
		return
	}
	id := c.Param("id")
	var input UpdateRoleRequest
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	updates := map[string]interface{}{}
	if input.Name != nil {
		updates["name"] = *input.Name
	}
	if input.Description != nil {
		updates["description"] = *input.Description
	}
	if input.PermissionBitmask != nil {
		updates["permission_bitmask"] = *input.PermissionBitmask
	}
	q := query.Use(db)
	if len(updates) > 0 {
		if _, err := q.Role.Where(query.Role.ID.Eq(id)).Updates(updates); err != nil {
			// MySQLの重複エラーをチェック
			if mysqlErr, ok := err.(*mysql.MySQLError); ok && mysqlErr.Number == 1062 {
				c.JSON(http.StatusConflict, gin.H{"error": "role name already exists", "code": "R0002"})
				return
			}
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
	}
	updated, _ := q.Role.Where(query.Role.ID.Eq(id)).First()
	resp := RoleDTO{
		ID:                updated.ID,
		CustomID:          updated.CustomID,
		Name:              updated.Name,
		Description:       ptrToString(updated.Description),
		PermissionBitmask: updated.PermissionBitmask,
	}
	c.JSON(http.StatusOK, resp)
}

func deleteRole(c *gin.Context) {
	db := getDB(c)
	if db == nil {
		return
	}
	id := c.Param("id")
	q := query.Use(db)
	if _, err := q.Role.Delete(&model.Role{ID: id}); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.Status(http.StatusNoContent)
}
