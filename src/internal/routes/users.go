package routes

import (
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"crypto/sha256"
	"encoding/hex"

	"golang.org/x/crypto/bcrypt"

	"github.com/UniPro-tech/UniQUE-API/internal/config"
	"github.com/UniPro-tech/UniQUE-API/internal/constants"
	"github.com/UniPro-tech/UniQUE-API/internal/middleware"
	"github.com/UniPro-tech/UniQUE-API/internal/model"
	"github.com/UniPro-tech/UniQUE-API/internal/query"
	"github.com/UniPro-tech/UniQUE-API/internal/utils"
	"github.com/gin-gonic/gin"
	"github.com/go-sql-driver/mysql"
	"github.com/oklog/ulid/v2"
	"gorm.io/gorm"
)

func RegisterUserRoutes(r *gin.Engine) {
	// 公開ルート
	g := r.Group("/users")
	{
		// ユーザー一覧の取得は認証不要（基本情報のみ公開）
		g.GET("", listUsers)

		// ユーザー情報の取得は認証不要（基本情報のみ公開、詳細は自分自身のみ）
		g.GET(":id", getUser)

		// ユーザーのアプリ一覧は自分自身 OR APP_READ権限
		g.GET(":id/apps", middleware.RequirePermissionOrSelf(constants.APP_READ), listAppsForUser)

		// ロールの追加・削除はPERMISSION_MANAGE権限が必要
		g.POST(":id/roles", middleware.RequirePermission(constants.PERMISSION_MANAGE), addRoleForUser)
		g.DELETE(":id/roles/:roleId", middleware.RequirePermission(constants.PERMISSION_MANAGE), removeRoleForUser)

		// ロール一覧の取得は自分自身 OR PERMISSION_MANAGE権限
		g.GET(":id/roles", middleware.RequirePermissionOrSelf(constants.PERMISSION_MANAGE), listRolesForUser)

		// 権限一覧の取得は自分自身のみ
		g.GET(":id/permissions", middleware.RequirePermissionOrSelf(constants.USER_READ), getUserPermissions)

		// 外部ID連携の閲覧は自分自身 OR USER_READ権限(EXTERNAL_IDENTITY_READと同等)
		g.GET(":id/external_identities", middleware.RequirePermissionOrSelf(constants.USER_READ), listExternalIdentities)

		// 外部ID連携の追加は自分自身 OR USER_UPDATE権限(EXTERNAL_IDENTITY_WRITEと同等)
		g.POST(":id/external_identities", middleware.RequirePermissionOrSelf(constants.USER_UPDATE), addExternalIdentity)

		// 外部ID連携の削除は自分自身 OR USER_UPDATE権限(EXTERNAL_IDENTITY_DELETEと同等)
		g.DELETE(":id/external_identities/:eid", middleware.RequirePermissionOrSelf(constants.USER_UPDATE), removeExternalIdentity)

		// ユーザー情報の更新は自分自身 OR USER_UPDATE権限
		g.PUT(":id", middleware.RequirePermissionOrSelf(constants.USER_UPDATE), updateUser)

		// パスワード変更: 自分自身またはUSER_UPDATE権限
		g.PUT(":id/password/change", middleware.RequirePermissionOrSelf(constants.USER_UPDATE), changePassword)

		// ユーザーの削除はUSER_DELETE権限が必要
		g.DELETE(":id", middleware.RequirePermission(constants.USER_DELETE), deleteUser)

		// ユーザー登録の承認はフロントと同じくUSER_CREATE権限が必要
		g.POST(":id/approve", middleware.RequirePermission(constants.USER_CREATE), approveUserRegist)

		// ユーザー登録の却下もUSER_CREATE権限が必要（フロントと整合）
		g.POST(":id/reject", middleware.RequirePermission(constants.USER_CREATE), rejectUserRegist)

		// メール認証の再送は自分自身 OR USER_UPDATE権限
		g.POST(":id/resend_email_verification", middleware.RequirePermissionOrSelf(constants.USER_UPDATE), resendEmailVerification)
	}

	// 内部用ルート（作成系）
	ig := r.Group("/internal/users")
	{
		ig.POST("", createUser)
		ig.GET("email_verify/:code", getEmailVerificationCode)
		ig.POST("email_verify/discord_link", linkDiscordByEmailCode)
		ig.POST("email_verify", emailCodeCheck)
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

// getPendingEmail は認証待ちのメールアドレスを取得する
func getPendingEmail(userID string, q *query.Query) string {
	evc, err := q.EmailVerificationCode.Where(
		query.EmailVerificationCode.UserID.Eq(userID),
		query.EmailVerificationCode.RequestType.Eq("email_change"),
	).Order(query.EmailVerificationCode.CreatedAt.Desc()).First()
	if err != nil {
		return ""
	}
	return ptrToString(evc.NewEmail)
}

// listUsers godoc
// @Summary List users
// @Description List users with embedded profile. Returns all data if USER_READ permission, otherwise basic info only
// @Tags users
// @Produce json
// @Success 200 {object} routes.UserListResponse
// @Router /users [get]
func listUsers(c *gin.Context) {
	if isOAuth := IsOAuth(c); isOAuth {
		// 403
		c.JSON(http.StatusForbidden, gin.H{"error": "You are not allowed to list applications with an access token"})
		return
	}
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

	// USER_READ権限があるかチェック
	hasUserReadPermission := false
	if user, exists := c.Get("user"); exists {
		if userModel, ok := user.(*model.User); ok && userModel != nil {
			permissions, _ := middleware.GetUserPermissions(userModel.ID, db)
			log.Printf("permission:%s", string(rune(permissions)))
			hasUserReadPermission = permissions.HasPermission(constants.USER_READ)
		}
	} else {
		log.Printf("no authenticated user found in context")
	}

	var out []UserDTO
	for _, u := range users {
		dto := UserDTO{
			ID:                u.ID,
			CustomID:          u.CustomID,
			Email:             u.Email,
			AffiliationPeriod: ptrToString(u.AffiliationPeriod),
			Status:            u.Status,
			CreatedAt:         u.CreatedAt,
			UpdatedAt:         u.UpdatedAt,
		}

		if p, ok := profileMap[u.ID]; ok {
			profileDTO := &ProfileDTO{
				UserID:           p.UserID,
				DisplayName:      p.DisplayName,
				Bio:              ptrToString(p.Bio),
				WebsiteURL:       ptrToString(p.WebsiteURL),
				TwitterHandle:    ptrToString(p.TwitterHandle),
				JoinedAt:         timeToTime(p.JoinedAt),
				BirthdateVisible: &p.BirthdateVisible,
			}
			// USER_READ権限があればExternalEmailを返す
			if hasUserReadPermission {
				dto.ExternalEmail = u.ExternalEmail
			}
			// birthdateはUSER_READ権限があるか、birthdateVisibleがtrueの場合のみ返す
			if hasUserReadPermission || p.BirthdateVisible {
				profileDTO.Birthdate = formatBirthdate(p.Birthdate)
			}
			dto.Profile = profileDTO
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
	if isOAuth := IsOAuth(c); isOAuth {
		// 403
		c.JSON(http.StatusForbidden, gin.H{"error": "You are not allowed to list applications with an access token"})
		return
	}
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
	resp, err := http.Post(config.IssuerInternalURL+"/internal/password_hash", "application/json", strings.NewReader(string(reqBody)))
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

	status := "established"
	if input.Status != "" {
		status = input.Status
	}

	user := model.User{
		ID:                ulid.Make().String(),
		CustomID:          input.CustomID,
		Email:             input.Email,
		PasswordHash:      respData.PasswordHash,
		ExternalEmail:     input.ExternalEmail,
		Status:            status,
		AffiliationPeriod: stringToPtr(input.AffiliationPeriod),
	}
	q := query.Use(db)
	if err := q.User.Create(&user); err != nil {
		// MySQLの重複エラーをチェック
		if mysqlErr, ok := err.(*mysql.MySQLError); ok && mysqlErr.Number == 1062 {
			// エラーメッセージから重複したキーを判定
			errMsg := mysqlErr.Message
			if strings.Contains(errMsg, "custom_id") {
				c.JSON(http.StatusConflict, gin.H{"error": "username already exists", "code": "R0006"})
				return
			} else if strings.Contains(errMsg, "email") {
				c.JSON(http.StatusConflict, gin.H{"error": "email already exists", "code": "R0007"})
				return
			}
			// その他の重複エラー
			c.JSON(http.StatusConflict, gin.H{"error": "duplicate entry", "code": "R0002"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	// 新規作成ユーザーに対して is_default=true のロールを付与
	if defaultRoles, derr := q.Role.Where(query.Role.IsDefault.Is(true)).Find(); derr == nil {
		for _, dr := range defaultRoles {
			ur := &model.UserRole{UserID: user.ID, RoleID: dr.ID}
			if err := q.UserRole.Create(ur); err != nil {
				// ロール付与失敗は致命的ではないのでログに残す
				log.Printf("failed to assign default role %s to user %s: %v", dr.ID, user.ID, err)
			}
		}
	} else {
		log.Printf("failed to fetch default roles: %v", derr)
	}
	err = sendRegistrationEmailVerification(user.ID, user.ExternalEmail, "", q, &config)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	var profileDTO *ProfileDTO
	if input.Profile != nil {
		p := &model.Profile{
			UserID:      user.ID,
			DisplayName: input.Profile.DisplayName,
		}
		if input.Profile.Bio != "" {
			p.Bio = &input.Profile.Bio
		}
		if input.Profile.WebsiteURL != "" {
			p.WebsiteURL = &input.Profile.WebsiteURL
		}
		if err := q.Profile.Create(p); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		profileDTO = &ProfileDTO{
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
	dbResp := UserDTO{
		ID:                user.ID,
		CustomID:          user.CustomID,
		Email:             user.Email,
		ExternalEmail:     user.ExternalEmail,
		EmailVerified:     user.EmailVerified,
		AffiliationPeriod: ptrToString(user.AffiliationPeriod),
		Status:            user.Status,
		CreatedAt:         user.CreatedAt,
		UpdatedAt:         user.UpdatedAt,
		Profile:           profileDTO,
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

	// 認証ユーザーを取得して自分かどうかを判定
	user, exists := c.Get("user")
	isSelf := false
	hasUserReadPermission := false
	isOAuth := IsOAuth(c)
	scopeStr := GetScope(c)
	if exists {
		userModel, ok := user.(*model.User)
		if ok && userModel != nil {
			if userModel.ID == id {
				isSelf = true
			}
			// OAuth トークンの場合は scope に基づく自己情報取得のみ許可
			if !isOAuth {
				permissions, _ := middleware.GetUserPermissions(userModel.ID, db)
				hasUserReadPermission = permissions.HasPermission(constants.USER_READ)
			} else {
				// OAuthかつisSelf=falseの場合は情報取得不可
				if !isSelf {
					c.JSON(http.StatusForbidden, gin.H{"error": "You are not allowed to access other users' information with an access token"})
					return
				}
			}
		}
	}

	q := query.Use(db)
	u, err := q.User.Where(query.User.ID.Eq(id)).First()
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
		return
	}

	// 基本情報は常に返す
	dto := UserDTO{
		ID:       u.ID,
		CustomID: u.CustomID,
	}

	// 自分自身またはUSER_READ権限がある場合はセンシティブ情報を含める
	sensitiveAllowed := false
	if isOAuth {
		// OAuth の場合は自身かつ scope に email が含まれている必要がある
		if isSelf && strings.Contains(scopeStr, "email") {
			sensitiveAllowed = true
		}
	} else {
		if isSelf || hasUserReadPermission {
			sensitiveAllowed = true
		}
	}
	if sensitiveAllowed {
		dto.Email = u.Email
		dto.ExternalEmail = u.ExternalEmail
		dto.EmailVerified = u.EmailVerified
		dto.AffiliationPeriod = ptrToString(u.AffiliationPeriod)
		dto.Status = u.Status
		dto.CreatedAt = u.CreatedAt
		dto.UpdatedAt = u.UpdatedAt
	}

	// PendingEmailは自分自身のみ（OAuth の場合は scope に email が必要）
	if isSelf {
		if isOAuth {
			if strings.Contains(scopeStr, "email") {
				dto.PendingEmail = getPendingEmail(u.ID, q)
			}
		} else {
			dto.PendingEmail = getPendingEmail(u.ID, q)
		}
	}

	// プロフィールは常に返す（birthdateはbirthdateVisibleで制御）
	if p, err := q.Profile.Where(query.Profile.UserID.Eq(u.ID)).First(); err == nil {
		profileDTO := &ProfileDTO{
			UserID:           p.UserID,
			DisplayName:      p.DisplayName,
			Bio:              ptrToString(p.Bio),
			WebsiteURL:       ptrToString(p.WebsiteURL),
			TwitterHandle:    ptrToString(p.TwitterHandle),
			JoinedAt:         timeToTime(p.JoinedAt),
			BirthdateVisible: &p.BirthdateVisible,
		}
		// birthdateはUSER_READ権限があるか、自分自身、またはbirthdateVisible、OAuth で scope に profile があれば返す
		if hasUserReadPermission || isSelf || p.BirthdateVisible || (isOAuth && strings.Contains(scopeStr, "profile")) {
			profileDTO.Birthdate = formatBirthdate(p.Birthdate)
		}
		dto.Profile = profileDTO
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
	if isOAuth := IsOAuth(c); isOAuth {
		// 403
		c.JSON(http.StatusForbidden, gin.H{"error": "You are not allowed to update users with an access token"})
		return
	}
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
	if input.Email != nil && *input.Email != user.Email {
		// メールアドレスの変更は管理者のみ可能
		permissions, exists := c.Get("permissions")
		if !exists {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "permissions not found"})
			return
		}

		perms, ok := permissions.(constants.Permission)
		if !ok || !perms.HasPermission(constants.USER_UPDATE) {
			c.JSON(http.StatusNotImplemented, gin.H{"error": "email change not implemented"})
			return
		}

		// 管理者ならメールアドレスを更新
		updates["email"] = *input.Email
	}
	if input.ExternalEmail != nil && *input.ExternalEmail != user.ExternalEmail {
		// 既存の未使用コードを削除
		_, _ = q.EmailVerificationCode.Where(
			query.EmailVerificationCode.UserID.Eq(id),
			query.EmailVerificationCode.RequestType.Eq("email_change"),
		).Delete()
		// external_emailは更新せず、認証コードのnew_emailに保存
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
		profileUpdates := map[string]interface{}{}
		if input.Profile.DisplayName != "" {
			profileUpdates["display_name"] = input.Profile.DisplayName
		}
		if input.Profile.Bio != "" {
			profileUpdates["bio"] = input.Profile.Bio
		}
		if input.Profile.WebsiteURL != "" {
			profileUpdates["website_url"] = input.Profile.WebsiteURL
		}
		if input.Profile.TwitterHandle != "" {
			profileUpdates["twitter_handle"] = input.Profile.TwitterHandle
		}
		if input.Profile.BirthdateVisible != nil {
			profileUpdates["birthdate_visible"] = *input.Profile.BirthdateVisible
		}
		if input.Profile.Birthdate != "" {
			t, err := time.Parse("2006-01-02", input.Profile.Birthdate)
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "invalid birthdate format, expected YYYY-MM-DD"})
				return
			}
			profileUpdates["birthdate"] = t
		}
		existing, err := q.Profile.Where(query.Profile.UserID.Eq(user.ID)).First()
		if err != nil {
			if err == gorm.ErrRecordNotFound {
				// 新規作成: 必須フィールドだけセット
				newProfile := &model.Profile{
					UserID: user.ID,
				}
				if v, ok := profileUpdates["display_name"]; ok {
					newProfile.DisplayName = v.(string)
				}
				if v, ok := profileUpdates["bio"]; ok {
					bioStr := v.(string)
					newProfile.Bio = &bioStr
				}
				if v, ok := profileUpdates["website_url"]; ok {
					urlStr := v.(string)
					newProfile.WebsiteURL = &urlStr
				}
				if v, ok := profileUpdates["twitter_handle"]; ok {
					twitterStr := v.(string)
					newProfile.TwitterHandle = &twitterStr
				}
				if v, ok := profileUpdates["birthdate_visible"]; ok {
					newProfile.BirthdateVisible = v.(bool)
				}
				if v, ok := profileUpdates["birthdate"]; ok {
					bdTime := v.(time.Time)
					newProfile.Birthdate = &bdTime
				}
				now := time.Now()
				newProfile.JoinedAt = &now
				if err := q.Profile.Create(newProfile); err != nil {
					c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
					return
				}
			} else {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
		} else if len(profileUpdates) > 0 {
			_ = existing // profile exists
			if _, err := q.Profile.Where(query.Profile.UserID.Eq(user.ID)).Updates(profileUpdates); err != nil {
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
		PendingEmail:      getPendingEmail(updated.ID, q),
		EmailVerified:     updated.EmailVerified,
		AffiliationPeriod: ptrToString(updated.AffiliationPeriod),
		Status:            updated.Status,
		CreatedAt:         updated.CreatedAt,
		UpdatedAt:         updated.UpdatedAt,
	}
	if p, err := q.Profile.Where(query.Profile.UserID.Eq(updated.ID)).First(); err == nil {
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
	c.JSON(http.StatusOK, dto)
}

func deleteUser(c *gin.Context) {
	if isOAuth := IsOAuth(c); isOAuth {
		// 403
		c.JSON(http.StatusForbidden, gin.H{"error": "You are not allowed to list applications with an access token"})
		return
	}
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
	if isOAuth := IsOAuth(c); isOAuth {
		// 403
		c.JSON(http.StatusForbidden, gin.H{"error": "You are not allowed to list applications with an access token"})
		return
	}
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
			Description:      ptrToString(a.Description),
			WebsiteURL:       ptrToString(a.WebsiteURL),
			PrivacyPolicyURL: ptrToString(a.PrivacyPolicyURL),
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
	if isOAuth := IsOAuth(c); isOAuth {
		// 403
		c.JSON(http.StatusForbidden, gin.H{"error": "You are not allowed to list applications with an access token"})
		return
	}
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
		Description:       ptrToString(role.Description),
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
	if isOAuth := IsOAuth(c); isOAuth {
		// 403
		c.JSON(http.StatusForbidden, gin.H{"error": "You are not allowed to list applications with an access token"})
		return
	}
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
	if isOAuth := IsOAuth(c); isOAuth {
		// 403
		c.JSON(http.StatusForbidden, gin.H{"error": "You are not allowed to list applications with an access token"})
		return
	}
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
			Description:       ptrToString(r.Description),
			PermissionBitmask: r.PermissionBitmask,
		})
	}
	c.JSON(http.StatusOK, RoleListResponse{Data: out})
}

// getUserPermissions godoc
// @Summary Get user permissions
// @Description Get the combined permissions for a user based on their roles
// @Tags users
// @Produce json
// @Param id path string true "User ID"
// @Success 200 {object} routes.PermissionsResponse
// @Router /users/{id}/permissions [get]
func getUserPermissions(c *gin.Context) {
	if isOAuth := IsOAuth(c); isOAuth {
		// 403
		c.JSON(http.StatusForbidden, gin.H{"error": "You are not allowed to list applications with an access token"})
		return
	}
	db := getDB(c)
	if db == nil {
		return
	}
	id := c.Param("id")

	// ユーザーが存在するか確認
	q := query.Use(db)
	_, err := q.User.Where(q.User.ID.Eq(id)).First()
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// 権限を取得
	permissions, err := middleware.GetUserPermissions(id, db)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// 権限のテキスト表現を取得
	permissionsText := constants.GetPermissionsText(int64(permissions))

	c.JSON(http.StatusOK, PermissionsResponse{
		PermissionBitmask: int64(permissions),
		PermissionsText:   permissionsText,
	})
}

// listExternalIdentities godoc
// @Summary List external identities for a user
// @Description Get external identities linked to a user
// @Tags users
// @Produce json
// @Param id path string true "User ID"
// @Success 200 {object} routes.ExternalIdentityListResponse
// @Router /users/{id}/external_identities [get]
func listExternalIdentities(c *gin.Context) {
	if isOAuth := IsOAuth(c); isOAuth {
		// 403
		c.JSON(http.StatusForbidden, gin.H{"error": "You are not allowed to list applications with an access token"})
		return
	}
	db := getDB(c)
	if db == nil {
		return
	}
	cfg := c.MustGet("config").(config.Config)
	id := c.Param("id")
	q := query.Use(db)
	eis, err := q.ExternalIdentity.Where(query.ExternalIdentity.UserID.Eq(id)).Find()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if len(eis) == 0 {
		c.JSON(http.StatusOK, ExternalIdentityListResponse{Data: []ExternalIdentityDTO{}})
		return
	}
	var out []ExternalIdentityDTO
	for _, e := range eis {
		// トークンの有効期限が切れていたらリフレッシュ
		refreshed, _ := utils.RefreshExternalToken(e, q, &cfg)

		dto := ExternalIdentityDTO{
			ID:             refreshed.ID,
			UserID:         refreshed.UserID,
			Provider:       refreshed.Provider,
			ExternalUserID: refreshed.ExternalUserID,
			CreatedAt:      refreshed.CreatedAt,
			UpdatedAt:      refreshed.UpdatedAt,
		}

		// ID Token をデコードして claims を取得
		if refreshed.IDToken != nil {
			if claims, err := utils.DecodeIDTokenClaims(*refreshed.IDToken); err == nil {
				dto.IDTokenClaims = claims
			}
		}

		// プロバイダの userinfo API を叩いて共通フィールド＋生データを取得
		if info, err := utils.FetchProviderUserInfo(refreshed); err == nil && info != nil {
			dto.Username = info.Username
			dto.DisplayName = info.DisplayName
			dto.AvatarURL = info.AvatarURL
			dto.Email = info.Email
			dto.ProviderData = info.ProviderData
		}

		out = append(out, dto)
	}
	c.JSON(http.StatusOK, ExternalIdentityListResponse{Data: out})
}

// addExternalIdentity godoc
// @Summary Link an external account
// @Description Link an external identity to the user
// @Tags users
// @Accept json
// @Produce json
// @Param id path string true "User ID"
// @Param body body routes.CreateExternalIdentityRequest true "External identity"
// @Success 201 {object} routes.ExternalIdentityDTO
// @Router /users/{id}/external_identities [post]
func addExternalIdentity(c *gin.Context) {
	if isOAuth := IsOAuth(c); isOAuth {
		// 403
		c.JSON(http.StatusForbidden, gin.H{"error": "You are not allowed to list applications with an access token"})
		return
	}
	db := getDB(c)
	if db == nil {
		return
	}
	id := c.Param("id")
	var input CreateExternalIdentityRequest
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
	ei := &model.ExternalIdentity{
		ID:             ulid.Make().String(),
		UserID:         id,
		Provider:       input.Provider,
		ExternalUserID: input.ExternalUserID,
		IDToken:        stringToPtr(input.IDToken),
		AccessToken:    input.AccessToken,
		RefreshToken:   input.RefreshToken,
	}
	if input.TokenExpiresAt != nil {
		ei.TokenExpiresAt = timeToTimePtr(*input.TokenExpiresAt)
	}
	if err := q.ExternalIdentity.Create(ei); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	resp := ExternalIdentityDTO{
		ID:             ei.ID,
		UserID:         ei.UserID,
		Provider:       ei.Provider,
		ExternalUserID: ei.ExternalUserID,
		CreatedAt:      ei.CreatedAt,
		UpdatedAt:      ei.UpdatedAt,
	}

	// ID Token をデコードして claims を取得
	if ei.IDToken != nil {
		if claims, err := utils.DecodeIDTokenClaims(*ei.IDToken); err == nil {
			resp.IDTokenClaims = claims
		}
	}

	// プロバイダの userinfo API を叩いて共通フィールド＋生データを取得
	if info, err := utils.FetchProviderUserInfo(ei); err == nil && info != nil {
		resp.Username = info.Username
		resp.DisplayName = info.DisplayName
		resp.AvatarURL = info.AvatarURL
		resp.Email = info.Email
		resp.ProviderData = info.ProviderData
	}

	c.JSON(http.StatusCreated, resp)
}

// removeExternalIdentity godoc
// @Summary Unlink an external account
// @Description Remove an external identity linked to a user
// @Tags users
// @Param id path string true "User ID"
// @Param eid path string true "External identity ID"
// @Success 204
// @Router /users/{id}/external_identities/{eid} [delete]
func removeExternalIdentity(c *gin.Context) {
	if isOAuth := IsOAuth(c); isOAuth {
		// 403
		c.JSON(http.StatusForbidden, gin.H{"error": "You are not allowed to list applications with an access token"})
		return
	}
	db := getDB(c)
	if db == nil {
		return
	}
	id := c.Param("id")
	eid := c.Param("eid")
	q := query.Use(db)
	// ensure exists and belongs to user
	if _, err := q.ExternalIdentity.Where(query.ExternalIdentity.ID.Eq(eid), query.ExternalIdentity.UserID.Eq(id)).First(); err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if _, err := q.ExternalIdentity.Delete(&model.ExternalIdentity{ID: eid}); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.Status(http.StatusNoContent)
}

// getEmailVerificationCode godoc
// @Summary Get email verification code info
// @Description メール検証コードからユーザーIDを取得する（メール認証フロー用）
// @Tags users
// @Produce json
// @Param code path string true "Email verification code"
// @Success 200 {object} map[string]string
// @Router /internal/users/email_verify/{code} [get]
func getEmailVerificationCode(c *gin.Context) {
	if isOAuth := IsOAuth(c); isOAuth {
		// 403
		c.JSON(http.StatusForbidden, gin.H{"error": "You are not allowed to list applications with an access token"})
		return
	}
	db := getDB(c)
	if db == nil {
		return
	}
	code := c.Param("code")
	q := query.Use(db)
	evc, err := q.EmailVerificationCode.Where(
		query.EmailVerificationCode.Code.Eq(code),
	).First()
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "code_not_found"})
		return
	}
	if time.Now().After(evc.ExpiresAt) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "code_expired"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"user_id": evc.UserID})
}

// linkDiscordByEmailCode godoc
// @Summary Link Discord account by email verification code
// @Description メール検証コードを使ってDiscord連携を行う
// @Tags users
// @Accept json
// @Produce json
// @Param body body routes.EmailVerifyDiscordLinkRequest true "Discord link request"
// @Success 201 {object} routes.ExternalIdentityDTO
// @Router /internal/users/email_verify/discord_link [post]
func linkDiscordByEmailCode(c *gin.Context) {
	if isOAuth := IsOAuth(c); isOAuth {
		// 403
		c.JSON(http.StatusForbidden, gin.H{"error": "You are not allowed to list applications with an access token"})
		return
	}
	db := getDB(c)
	if db == nil {
		return
	}
	var input EmailVerifyDiscordLinkRequest
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	q := query.Use(db)
	evc, err := q.EmailVerificationCode.Where(
		query.EmailVerificationCode.Code.Eq(input.Code),
	).First()
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "code_not_found"})
		return
	}
	if time.Now().After(evc.ExpiresAt) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "code_expired"})
		return
	}
	if evc.RequestType != "registration" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_request_type"})
		return
	}
	// ensure user exists
	if _, err := q.User.Where(query.User.ID.Eq(evc.UserID)).First(); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
		return
	}
	// already linked for this user
	if _, err := q.ExternalIdentity.Where(
		query.ExternalIdentity.UserID.Eq(evc.UserID),
		query.ExternalIdentity.Provider.Eq("discord"),
	).First(); err == nil {
		c.JSON(http.StatusConflict, gin.H{"error": "discord_already_linked"})
		return
	} else if err != gorm.ErrRecordNotFound {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	// prevent linking the same Discord account to another user
	if existing, err := q.ExternalIdentity.Where(
		query.ExternalIdentity.Provider.Eq("discord"),
		query.ExternalIdentity.ExternalUserID.Eq(input.ExternalUserID),
	).First(); err == nil {
		if existing.UserID != evc.UserID {
			c.JSON(http.StatusConflict, gin.H{"error": "discord_already_linked"})
			return
		}
		c.JSON(http.StatusConflict, gin.H{"error": "discord_already_linked"})
		return
	} else if err != gorm.ErrRecordNotFound {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	ei := &model.ExternalIdentity{
		ID:             ulid.Make().String(),
		UserID:         evc.UserID,
		Provider:       "discord",
		ExternalUserID: input.ExternalUserID,
		AccessToken:    input.AccessToken,
		RefreshToken:   input.RefreshToken,
	}
	if input.TokenExpiresAt != nil {
		ei.TokenExpiresAt = timeToTimePtr(*input.TokenExpiresAt)
	}
	if err := q.ExternalIdentity.Create(ei); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	resp := ExternalIdentityDTO{
		ID:             ei.ID,
		UserID:         ei.UserID,
		Provider:       ei.Provider,
		ExternalUserID: ei.ExternalUserID,
		CreatedAt:      ei.CreatedAt,
		UpdatedAt:      ei.UpdatedAt,
	}
	if info, err := utils.FetchProviderUserInfo(ei); err == nil && info != nil {
		resp.Username = info.Username
		resp.DisplayName = info.DisplayName
		resp.AvatarURL = info.AvatarURL
		resp.Email = info.Email
		resp.ProviderData = info.ProviderData
	}
	c.JSON(http.StatusCreated, resp)
}

// EmailCodeCheck godoc
// @Summary Verify email code
// @Description 認証コードを検証する
// @Tags users
// @Accept json
// @Produce json
// @Param body body routes.EmailCodeCheckRequest true "Email code verification"
// @Success 200 {object} routes.EmailCodeCheckResponse
// @Router /internal/users/email_verify [post]
func emailCodeCheck(c *gin.Context) {
	if isOAuth := IsOAuth(c); isOAuth {
		// 403
		c.JSON(http.StatusForbidden, gin.H{"error": "You are not allowed to list applications with an access token"})
		return
	}
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

	// Get user to determine verification type
	user, err := q.User.Where(query.User.ID.Eq(evc.UserID)).First()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "user_not_found"})
		return
	}

	// Determine verification type based on request_type and user status
	var verificationType string
	switch evc.RequestType {
	case "registration":
		if user.Status == "established" {
			verificationType = "signup"
		} else {
			verificationType = "migration"
		}
	case "email_change":
		if evc.NewEmail != nil && *evc.NewEmail != "" {
			verificationType = "change"
		} else {
			verificationType = "migration"
		}
	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_request_type"})
		return
	}

	// Check Discord account linkage for signup type
	if verificationType == "signup" {
		_, err := q.ExternalIdentity.Where(
			query.ExternalIdentity.UserID.Eq(evc.UserID),
			query.ExternalIdentity.Provider.Eq("discord"),
		).First()
		if err != nil {
			if err == gorm.ErrRecordNotFound {
				c.JSON(http.StatusBadRequest, gin.H{"error": "discord_not_linked"})
				return
			}
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
	}

	// mark email as verified
	switch evc.RequestType {
	case "registration":
		_, err = q.User.Where(query.User.ID.Eq(evc.UserID)).Updates(map[string]interface{}{
			"email_verified": true,
		})
	case "email_change":
		_, err = q.User.Where(query.User.ID.Eq(evc.UserID)).Updates(map[string]interface{}{
			"external_email": evc.NewEmail,
			"email_verified": true,
		})
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	// delete used code
	_, _ = q.EmailVerificationCode.Delete(&model.EmailVerificationCode{ID: evc.ID})
	c.JSON(http.StatusOK, EmailCodeCheckResponse{Valid: true, Type: verificationType})
}

// approveUserRegist godoc
// @Summary Approve user registration
// @Description ユーザ登録を承認する
// @Tags users
// @Produce json
// @Param id path string true "User ID"
// @Success 200
// @Router /users/{id}/approve [post]
func approveUserRegist(c *gin.Context) {
	if isOAuth := IsOAuth(c); isOAuth {
		// 403
		c.JSON(http.StatusForbidden, gin.H{"error": "You are not allowed to list applications with an access token"})
		return
	}
	db := getDB(c)
	if db == nil {
		return
	}
	user_id := c.Param("id")
	q := query.Use(db)
	_, err := q.User.Where(query.User.ID.Eq(user_id)).Updates(map[string]interface{}{
		"status": "active",
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.Status(http.StatusOK)
}

// rejectUserRegist godoc
// @Summary Reject user registration
// @Description ユーザ登録を却下し、ユーザを物理削除する
// @Tags users
// @Produce json
// @Param id path string true "User ID"
// @Success 200
// @Router /users/{id}/reject [post]
func rejectUserRegist(c *gin.Context) {
	if isOAuth := IsOAuth(c); isOAuth {
		c.JSON(http.StatusForbidden, gin.H{"error": "You are not allowed to list applications with an access token"})
		return
	}
	db := getDB(c)
	if db == nil {
		return
	}
	user_id := c.Param("id")
	q := query.Use(db)
	// Perform physical delete using Unscoped()
	if _, err := q.User.Unscoped().Delete(&model.User{ID: user_id}); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.Status(http.StatusOK)
}

// resendEmailVerification godoc
// @Summary Resend email verification
// @Description メール認証メールを再送する
// @Tags users
// @Produce json
// @Param id path string true "User ID"
// @Success 200
// @Router /users/{id}/resend_email_verification [post]
func resendEmailVerification(c *gin.Context) {
	if isOAuth := IsOAuth(c); isOAuth {
		// 403
		c.JSON(http.StatusForbidden, gin.H{"error": "You are not allowed to list applications with an access token"})
		return
	}
	db := getDB(c)
	if db == nil {
		return
	}
	id := c.Param("id")
	q := query.Use(db)
	user, err := q.User.Where(query.User.ID.Eq(id)).First()
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if user.EmailVerified {
		c.JSON(http.StatusBadRequest, gin.H{"error": "email already verified"})
		return
	}
	if user.ExternalEmail == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "no external email set"})
		return
	}
	// 既存のコードを取得
	existingCodes, err := q.EmailVerificationCode.Where(
		query.EmailVerificationCode.UserID.Eq(id),
		query.EmailVerificationCode.RequestType.Eq("email_change"),
	).Find()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if len(existingCodes) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "no existing verification request"})
		return
	}
	externalEmail := existingCodes[0].NewEmail
	if externalEmail == nil || *externalEmail == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "no external email set in existing request"})
		return
	}
	// 既存の未使用コードを削除
	_, _ = q.EmailVerificationCode.Where(
		query.EmailVerificationCode.UserID.Eq(id),
		query.EmailVerificationCode.RequestType.Eq("email_change"),
	).Delete()
	// プロフィールから名前を取得
	name := ""
	if p, err := q.Profile.Where(query.Profile.UserID.Eq(id)).First(); err == nil {
		name = p.DisplayName
	}
	cfg := config.LoadConfig()
	err = sendEmailChangeVerification(id, *externalEmail, name, q, cfg)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to send verification email: " + err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "verification email sent"})
}

// changePassword godoc
// @Summary Change user password
// @Description Change a user's password. The requester must be the user themself or have USER_UPDATE permission. If the requester is the user, the current password must be provided.
// @Tags users
// @Accept json
// @Produce json
// @Param id path string true "User ID"
// @Param body body map[string]string true "Password change request"
// @Success 200 {object} map[string]string
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Failure 403 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /users/{id}/password/change [put]
func changePassword(c *gin.Context) {
	if isOAuth := IsOAuth(c); isOAuth {
		// 403
		c.JSON(http.StatusForbidden, gin.H{"error": "You are not allowed to list applications with an access token"})
		return
	}
	db := getDB(c)
	if db == nil {
		return
	}
	id := c.Param("id")
	q := query.Use(db)

	userModel, err := q.User.Where(query.User.ID.Eq(id)).First()
	if err != nil || userModel == nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	var input struct {
		CurrentPassword string `json:"current_password"`
		NewPassword     string `json:"new_password" binding:"required"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Determine if requester is self or has USER_UPDATE
	isSelf := false
	hasPerm := false
	if u, exists := c.Get("user"); exists {
		if um, ok := u.(*model.User); ok && um != nil {
			if um.ID == id {
				isSelf = true
			}
			if perms, err := middleware.GetUserPermissions(um.ID, db); err == nil {
				hasPerm = perms.HasPermission(constants.USER_UPDATE)
			}
		}
	}

	if !isSelf && !hasPerm {
		c.JSON(http.StatusForbidden, gin.H{"error": "access denied"})
		return
	}

	// If requester is self, verify current password
	if isSelf {
		if input.CurrentPassword == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "current_password required"})
			return
		}
		if err := bcrypt.CompareHashAndPassword([]byte(userModel.PasswordHash), []byte(input.CurrentPassword)); err != nil {
			// If bcrypt hash is malformed, fall back to legacy SHA256 hex(password)
			if _, ok := err.(bcrypt.InvalidHashPrefixError); ok {
				sum := sha256.Sum256([]byte(input.CurrentPassword))
				if hex.EncodeToString(sum[:]) != userModel.PasswordHash {
					c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid current password"})
					return
				}
				// matched legacy hash; continue
			} else {
				c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid current password"})
				return
			}
		}
	}

	// Generate hash for new password via issuer internal API
	cfg := c.MustGet("config").(config.Config)
	req := map[string]string{"password": input.NewPassword}
	reqBody, jerr := json.Marshal(req)
	if jerr != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": jerr.Error()})
		return
	}
	resp, err := http.Post(cfg.IssuerInternalURL+"/internal/password_hash", "application/json", strings.NewReader(string(reqBody)))
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

	if _, err := q.User.Where(query.User.ID.Eq(id)).Update(q.User.PasswordHash, respData.PasswordHash); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update password"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "password changed"})
}

func sendRegistrationEmailVerification(user_id, email, name string, q *query.Query, config *config.Config) error {
	// コードを生成してDBに保存する
	// 6桁のコードを生成
	b := make([]byte, 3)
	_, _ = rand.Read(b)
	code := fmt.Sprintf("%06X", b)

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
	b := make([]byte, 3)
	_, _ = rand.Read(b)
	code := fmt.Sprintf("%06X", b)

	err := q.EmailVerificationCode.Create(&model.EmailVerificationCode{
		Code:        code,
		RequestType: "email_change",
		UserID:      user_id,
		ExpiresAt:   time.Now().Add(10 * time.Minute),
		NewEmail:    stringToPtr(email),
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
