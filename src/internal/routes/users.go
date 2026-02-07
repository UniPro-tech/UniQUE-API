package routes

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/UniPro-tech/UniQUE-API/internal/config"
	"github.com/UniPro-tech/UniQUE-API/internal/model"
	"github.com/UniPro-tech/UniQUE-API/internal/query"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

func RegisterUserRoutes(r *gin.Engine) {
	g := r.Group("/users")
	{
		g.GET("", listUsers)
		g.POST("", createUser)
		g.GET(":id", getUser)
		g.GET(":id/apps", listAppsForUser)
		g.POST(":id/roles", addRoleForUser)
		g.DELETE(":id/roles/:roleId", removeRoleForUser)
		g.GET(":id/roles", listRolesForUser)
		g.PUT(":id", updateUser)
		g.DELETE(":id", deleteUser)
	}
}

func getDB(c *gin.Context) *gorm.DB {
	dbi, ok := c.MustGet("db").(*gorm.DB)
	if !ok {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "database not available"})
		return nil
	}
	return dbi
}

// listUsers godoc
// @Summary List users
// @Description List users with embedded profile
// @Tags users
// @Produce json
// @Success 200 {object} routes.UserListResponse
// @Router /users [get]
func listUsers(c *gin.Context) {
	db := getDB(c)
	if db == nil {
		return
	}
	q := query.Use(db)
	users, err := q.User.Find()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if len(users) == 0 {
		c.JSON(http.StatusOK, UserListResponse{Data: []UserDTO{}})
		return
	}
	ids := make([]string, 0, len(users))
	for _, u := range users {
		ids = append(ids, u.ID)
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
			AffiliationPeriod: u.AffiliationPeriod,
			Status:            u.Status,
			CreatedAt:         u.CreatedAt,
			UpdatedAt:         u.UpdatedAt,
		}
		if p, ok := profileMap[u.ID]; ok {
			dto.Profile = &ProfileDTO{
				UserID:      p.UserID,
				DisplayName: p.DisplayName,
				Bio:         p.Bio,
				WebsiteURL:  p.WebsiteURL,
				JoinedAt:    p.JoinedAt,
			}
		}
		out = append(out, dto)
	}
	c.JSON(http.StatusOK, UserListResponse{Data: out})
}

// createUser godoc
// @Summary Create a user
// @Description Create a new user with optional profile
// @Tags users
// @Accept json
// @Produce json
// @Param user body routes.CreateUserRequest true "Create user"
// @Success 201 {object} routes.UserDTO
// @Router /users [post]
func createUser(c *gin.Context) {
	db := getDB(c)
	if db == nil {
		return
	}
	config := c.MustGet("config").(config.Config)
	var input CreateUserRequest
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	// auth-server/internal/password_hash
	req := map[string]string{
		"password": input.Password,
	}
	reqBody, err := json.Marshal(req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	resp, err := http.Post(config.IssuerURL+"/hash-password", "application/json", strings.NewReader(string(reqBody)))
	if err != nil || resp.StatusCode != http.StatusOK {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to hash password"})
		return
	}
	defer resp.Body.Close()
	var respData struct {
		PasswordHash string `json:"password_hash"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&respData); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to parse password hash response"})
		return
	}

	user := model.User{
		ID:           uuid.NewString(),
		CustomID:     input.CustomID,
		Email:        input.Email,
		PasswordHash: respData.PasswordHash,
		Status:       "established",
	}
	q := query.Use(db)
	if err := q.User.Create(&user); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	err = sendRegistrationEmailVerification(user.ID, user.Email, "", q, &config)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	var profileDTO *ProfileDTO
	if input.Profile != nil {
		p := &model.Profile{
			UserID:      user.ID,
			DisplayName: input.Profile.DisplayName,
			Bio:         input.Profile.Bio,
			WebsiteURL:  input.Profile.WebsiteURL,
		}
		if err := q.Profile.Create(p); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		profileDTO = &ProfileDTO{
			UserID:      p.UserID,
			DisplayName: p.DisplayName,
			Bio:         p.Bio,
			WebsiteURL:  p.WebsiteURL,
			JoinedAt:    p.JoinedAt,
		}
	}
	dbResp := UserDTO{
		ID:        user.ID,
		CustomID:  user.CustomID,
		Email:     user.Email,
		Status:    user.Status,
		CreatedAt: user.CreatedAt,
		UpdatedAt: user.UpdatedAt,
		Profile:   profileDTO,
	}
	c.JSON(http.StatusCreated, dbResp)
}

// getUser godoc
// @Summary Get a user
// @Description Get a single user with optional profile
// @Tags users
// @Produce json
// @Param id path string true "User ID"
// @Success 200 {object} routes.UserDTO
// @Router /users/{id} [get]
func getUser(c *gin.Context) {
	db := getDB(c)
	if db == nil {
		return
	}
	id := c.Param("id")
	q := query.Use(db)
	u, err := q.User.Where(query.User.ID.Eq(id)).First()
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
		return
	}
	dto := UserDTO{
		ID:                u.ID,
		CustomID:          u.CustomID,
		Email:             u.Email,
		ExternalEmail:     u.ExternalEmail,
		EmailVerified:     u.EmailVerified,
		AffiliationPeriod: u.AffiliationPeriod,
		Status:            u.Status,
		CreatedAt:         u.CreatedAt,
		UpdatedAt:         u.UpdatedAt,
	}
	if p, err := q.Profile.Where(query.Profile.UserID.Eq(u.ID)).First(); err == nil {
		dto.Profile = &ProfileDTO{
			UserID:      p.UserID,
			DisplayName: p.DisplayName,
			Bio:         p.Bio,
			WebsiteURL:  p.WebsiteURL,
			JoinedAt:    p.JoinedAt,
		}
	}
	c.JSON(http.StatusOK, dto)
}

// updateUser godoc
// @Summary Update a user
// @Description Update a user's fields and optional profile
// @Tags users
// @Accept json
// @Produce json
// @Param id path string true "User ID"
// @Param user body routes.UpdateUserRequest true "Update user"
// @Success 200 {object} routes.UserDTO
// @Router /users/{id} [put]
func updateUser(c *gin.Context) {
	db := getDB(c)
	if db == nil {
		return
	}
	id := c.Param("id")
	q := query.Use(db)
	user, err := q.User.Where(query.User.ID.Eq(id)).First()
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	var input UpdateUserRequest
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	// apply updates to user
	updates := map[string]interface{}{}
	if input.Email != nil {
		c.JSON(http.StatusNotImplemented, gin.H{"error": "email change not implemented"})
		return
	}
	if input.ExternalEmail != nil {
		sendEmailChangeVerification(id, *input.ExternalEmail, "", q, config.LoadConfig())
		updates["email_verified"] = false
	}
	if input.AffiliationPeriod != nil {
		updates["affiliation_period"] = *input.AffiliationPeriod
	}
	if input.Status != nil {
		updates["status"] = *input.Status
	}
	if len(updates) > 0 {
		if _, err := q.User.Where(query.User.ID.Eq(id)).Updates(updates); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
	}
	if input.Profile != nil {
		p := &model.Profile{
			UserID:      user.ID,
			DisplayName: input.Profile.DisplayName,
			Bio:         input.Profile.Bio,
			WebsiteURL:  input.Profile.WebsiteURL,
		}
		if _, err := q.Profile.Where(query.Profile.UserID.Eq(user.ID)).First(); err != nil {
			if err == gorm.ErrRecordNotFound {
				if err := q.Profile.Create(p); err != nil {
					c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
					return
				}
			} else {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
		} else {
			if _, err := q.Profile.Where(query.Profile.UserID.Eq(user.ID)).Updates(p); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
		}
	}
	// rebuild response dto
	updated, _ := q.User.Where(query.User.ID.Eq(id)).First()
	dto := UserDTO{
		ID:                updated.ID,
		CustomID:          updated.CustomID,
		Email:             updated.Email,
		ExternalEmail:     updated.ExternalEmail,
		EmailVerified:     updated.EmailVerified,
		AffiliationPeriod: updated.AffiliationPeriod,
		Status:            updated.Status,
		CreatedAt:         updated.CreatedAt,
		UpdatedAt:         updated.UpdatedAt,
	}
	if p, err := q.Profile.Where(query.Profile.UserID.Eq(updated.ID)).First(); err == nil {
		dto.Profile = &ProfileDTO{
			UserID:      p.UserID,
			DisplayName: p.DisplayName,
			Bio:         p.Bio,
			WebsiteURL:  p.WebsiteURL,
			JoinedAt:    p.JoinedAt,
		}
	}
	c.JSON(http.StatusOK, dto)
}

func deleteUser(c *gin.Context) {
	db := getDB(c)
	if db == nil {
		return
	}
	id := c.Param("id")
	q := query.Use(db)
	if _, err := q.User.Delete(&model.User{ID: id}); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.Status(http.StatusNoContent)
}

// listAppsForUser godoc
// @Summary List applications for a user
// @Description Get applications owned by a user
// @Tags users
// @Produce json
// @Param id path string true "User ID"
// @Success 200 {object} routes.ApplicationListResponse
// @Router /users/{id}/apps [get]
func listAppsForUser(c *gin.Context) {
	db := getDB(c)
	if db == nil {
		return
	}
	id := c.Param("id")
	q := query.Use(db)
	apps, err := q.Application.Where(query.Application.UserID.Eq(id)).Find()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if len(apps) == 0 {
		c.JSON(http.StatusOK, ApplicationListResponse{Data: []ApplicationDTO{}})
		return
	}
	var out []ApplicationDTO
	for _, a := range apps {
		out = append(out, ApplicationDTO{
			ID:               a.ID,
			Name:             a.Name,
			Description:      a.Description,
			WebsiteURL:       a.WebsiteURL,
			PrivacyPolicyURL: a.PrivacyPolicyURL,
			UserID:           a.UserID,
		})
	}
	c.JSON(http.StatusOK, ApplicationListResponse{Data: out})
}

// addRoleForUser godoc
// @Summary Assign a role to a user
// @Description Assign the specified role to the user
// @Tags users
// @Accept json
// @Produce json
// @Param id path string true "User ID"
// @Param body body routes.CreateUserRoleRequest true "Role assignment"
// @Success 201 {object} routes.RoleDTO
// @Router /users/{id}/roles [post]
func addRoleForUser(c *gin.Context) {
	db := getDB(c)
	if db == nil {
		return
	}
	id := c.Param("id")
	var input CreateUserRoleRequest
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	q := query.Use(db)
	// ensure user exists
	if _, err := q.User.Where(query.User.ID.Eq(id)).First(); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
		return
	}
	// ensure role exists
	role, err := q.Role.Where(query.Role.ID.Eq(input.RoleID)).First()
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "role not found"})
		return
	}
	// check existing assignment
	if _, err := q.UserRole.Where(query.UserRole.UserID.Eq(id), query.UserRole.RoleID.Eq(input.RoleID)).First(); err == nil {
		c.JSON(http.StatusConflict, gin.H{"error": "role already assigned"})
		return
	} else if err != gorm.ErrRecordNotFound {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	ur := &model.UserRole{UserID: id, RoleID: input.RoleID}
	if err := q.UserRole.Create(ur); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	resp := RoleDTO{
		ID:                role.ID,
		CustomID:          role.CustomID,
		Name:              role.Name,
		Description:       role.Description,
		PermissionBitmask: role.PermissionBitmask,
	}
	c.JSON(http.StatusCreated, resp)
}

// removeRoleForUser godoc
// @Summary Remove a role from a user
// @Description Unassign the specified role from the user
// @Tags users
// @Produce json
// @Param id path string true "User ID"
// @Param roleId path string true "Role ID"
// @Success 204
// @Router /users/{id}/roles/{roleId} [delete]
func removeRoleForUser(c *gin.Context) {
	db := getDB(c)
	if db == nil {
		return
	}
	id := c.Param("id")
	roleId := c.Param("roleId")
	q := query.Use(db)
	// ensure assignment exists
	if _, err := q.UserRole.Where(query.UserRole.UserID.Eq(id), query.UserRole.RoleID.Eq(roleId)).First(); err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "assignment not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if _, err := q.UserRole.Delete(&model.UserRole{UserID: id, RoleID: roleId}); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.Status(http.StatusNoContent)
}

// listRolesForUser godoc
// @Summary List roles for a user
// @Description Get roles assigned to a user
// @Tags users
// @Produce json
// @Param id path string true "User ID"
// @Success 200 {object} routes.RoleListResponse
// @Router /users/{id}/roles [get]
func listRolesForUser(c *gin.Context) {
	db := getDB(c)
	if db == nil {
		return
	}
	id := c.Param("id")
	q := query.Use(db)
	urs, err := q.UserRole.Where(query.UserRole.UserID.Eq(id)).Find()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if len(urs) == 0 {
		c.JSON(http.StatusOK, RoleListResponse{Data: []RoleDTO{}})
		return
	}
	ids := make([]string, 0, len(urs))
	for _, ur := range urs {
		ids = append(ids, ur.RoleID)
	}
	roles, err := q.Role.Where(query.Role.ID.In(ids...)).Find()
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
			Description:       r.Description,
			PermissionBitmask: r.PermissionBitmask,
		})
	}
	c.JSON(http.StatusOK, RoleListResponse{Data: out})
}

// EmailCodeCheck godoc
// @Summary Verify email code
// @Description 認証コードを検証する
// @Tags users
// @Accept json
// @Produce json
// @Param body body routes.EmailCodeCheckRequest true "Email code verification"
// @Success 200 {object} routes.EmailCodeCheckResponse
// @Router /users/email_verify [post]
func emailCodeCheck(c *gin.Context) {
	db := getDB(c)
	if db == nil {
		return
	}
	var input EmailCodeCheckRequest
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	q := query.Use(db)
	evc, err := q.EmailVerificationCode.Where(
		query.EmailVerificationCode.Code.Eq(input.Code),
	).First()
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_code"})
		return
	}
	if time.Now().After(evc.ExpiresAt) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "code_expired"})
		return
	}
	// mark email as verified
	switch evc.RequestType {
	case "registration":
		_, err = q.User.Where(query.User.ID.Eq(evc.UserID)).Updates(map[string]interface{}{
			"email_verified": true,
		})
	case "email_change":
		_, err = q.User.Where(query.User.ID.Eq(evc.UserID)).Updates(map[string]interface{}{
			"email":          evc.NewEmail,
			"email_verified": true,
		})
	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_request_type"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	// delete used code
	_, _ = q.EmailVerificationCode.Delete(&model.EmailVerificationCode{ID: evc.ID})
	c.JSON(http.StatusOK, EmailCodeCheckResponse{Valid: true})
}

func sendRegistrationEmailVerification(user_id, email, name string, q *query.Query, config *config.Config) error {
	// コードを生成してDBに保存する
	// 6桁のコードを生成
	code := strings.ToUpper(uuid.New().String()[:6])

	err := q.EmailVerificationCode.Create(&model.EmailVerificationCode{
		Code:        code,
		RequestType: "registration",
		UserID:      user_id,
		ExpiresAt:   time.Now().Add(10 * time.Minute),
	})
	if err != nil {
		return err
	}

	// Email APIを呼び出す
	// HTTP /register { code, email, name }
	endpoint := config.EmailSenderURL + "/register"
	payload := map[string]string{
		"code":  code,
		"email": email,
		"name":  name,
	}
	client := &http.Client{Timeout: 10 * time.Second}
	reqBody, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	req, err := http.NewRequest("POST", endpoint, strings.NewReader(string(reqBody)))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return errors.New("failed to send email verification")
	}

	return nil
}

func sendEmailChangeVerification(user_id, email, name string, q *query.Query, config *config.Config) error {
	// コードを生成してDBに保存する
	// 6桁のコードを生成
	code := strings.ToUpper(uuid.New().String()[:6])

	err := q.EmailVerificationCode.Create(&model.EmailVerificationCode{
		Code:        code,
		RequestType: "email_change",
		UserID:      user_id,
		ExpiresAt:   time.Now().Add(10 * time.Minute),
		NewEmail:    email,
	})
	if err != nil {
		return err
	}

	// Email APIを呼び出す
	// HTTP /email-change { code, email, name }
	endpoint := config.EmailSenderURL + "/email-change"
	payload := map[string]string{
		"code":  code,
		"email": email,
		"name":  name,
	}
	client := &http.Client{Timeout: 10 * time.Second}
	reqBody, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	req, err := http.NewRequest("POST", endpoint, strings.NewReader(string(reqBody)))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return errors.New("failed to send email verification")
	}

	return nil
}
