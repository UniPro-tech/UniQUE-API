package routes

import (
	"net/http"

	"github.com/UniPro-tech/UniQUE-API/internal/constants"
	"github.com/UniPro-tech/UniQUE-API/internal/middleware"
	"github.com/UniPro-tech/UniQUE-API/internal/model"
	"github.com/UniPro-tech/UniQUE-API/internal/query"
	"github.com/gin-gonic/gin"
	"github.com/oklog/ulid/v2"
	"gorm.io/gorm"
)

func RegisterApplicationRoutes(r *gin.Engine) {
	// 公開ルート（読み取り専用）
	g := r.Group("/applications")
	{
		g.GET("", listApplications)
		g.GET(":id", getApplication)
		g.GET(":id/redirect_uris", listRedirectURIsForApplication)
		g.POST(":id/redirect_uris", createRedirectURIForApplication)
		g.DELETE(":id/redirect_uris", deleteRedirectURIForApplication)
		g.GET(":id/owners", listOwnersForApplication)
		g.PUT(":id", updateApplication)
		g.DELETE(":id", deleteApplication)
	}

	// 内部用ルート（作成系）
	ig := r.Group("/internal/applications")
	{
		ig.POST("", createApplication)
		ig.POST(":id/owners", addOwnerForApplication)
	}
}

// listApplications godoc
// @Summary List applications
// @Description List third-party applications
// @Tags applications
// @Produce json
// @Success 200 {object} routes.ApplicationListResponse
// @Router /applications [get]
func listApplications(c *gin.Context) {
	db := getDB(c)
	if db == nil {
		return
	}
	q := query.Use(db)

	// Check if the requester has APP_READ permission. If not,
	// only return applications owned by the authenticated user.
	hasAppReadPermission := false
	var authUser *model.User
	if ui, exists := c.Get("user"); exists {
		if su, ok := ui.(*model.User); ok && su != nil {
			authUser = su
			perms, _ := middleware.GetUserPermissions(su.ID, db)
			hasAppReadPermission = perms.HasPermission(constants.APP_READ)
		}
	}

	var apps []*model.Application
	var err error
	if hasAppReadPermission {
		apps, err = q.Application.Find()
	} else if authUser != nil {
		apps, err = q.Application.Where(query.Application.UserID.Eq(authUser.ID)).Find()
	} else {
		// unauthenticated and no APP_READ -> return empty list
		c.JSON(http.StatusOK, ApplicationListResponse{Data: []ApplicationDTO{}})
		return
	}

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
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

// createApplication godoc
// @Summary Create an application
// @Description Register a new third-party application
// @Tags applications
// @Accept json
// @Produce json
// @Param app body routes.CreateApplicationRequest true "Create application"
// @Success 201 {object} routes.ApplicationDTO
// @Router /applications [post]
func createApplication(c *gin.Context) {
	db := getDB(c)
	if db == nil {
		return
	}
	var input CreateApplicationRequest
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	app := model.Application{
		ID:               ulid.Make().String(),
		Name:             input.Name,
		Description:      stringToPtr(input.Description),
		WebsiteURL:       stringToPtr(input.WebsiteURL),
		PrivacyPolicyURL: stringToPtr(input.PrivacyPolicyURL),
		ClientSecret:     input.ClientSecret,
		UserID:           input.UserID,
	}
	q := query.Use(db)
	// If a user is present in the context (session), use it as the owner.
	if ui, exists := c.Get("user"); exists {
		if su, ok := ui.(*model.User); ok && su != nil {
			app.UserID = su.ID
		}
	}
	if err := q.Application.Create(&app); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	resp := ApplicationDTO{
		ID:               app.ID,
		Name:             app.Name,
		Description:      ptrToString(app.Description),
		WebsiteURL:       ptrToString(app.WebsiteURL),
		PrivacyPolicyURL: ptrToString(app.PrivacyPolicyURL),
		UserID:           app.UserID,
	}
	c.JSON(http.StatusCreated, resp)
}

// getApplication godoc
// @Summary Get an application
// @Description Get a single application
// @Tags applications
// @Produce json
// @Param id path string true "Application ID"
// @Success 200 {object} routes.ApplicationDTO
// @Router /applications/{id} [get]
func getApplication(c *gin.Context) {
	db := getDB(c)
	if db == nil {
		return
	}
	id := c.Param("id")
	q := query.Use(db)
	a, err := q.Application.Where(query.Application.ID.Eq(id)).First()
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
		return
	}
	resp := ApplicationDTO{
		ID:               a.ID,
		Name:             a.Name,
		Description:      ptrToString(a.Description),
		WebsiteURL:       ptrToString(a.WebsiteURL),
		PrivacyPolicyURL: ptrToString(a.PrivacyPolicyURL),
		UserID:           a.UserID,
	}
	c.JSON(http.StatusOK, resp)
}

// updateApplication godoc
// @Summary Update an application
// @Description Update application fields
// @Tags applications
// @Accept json
// @Produce json
// @Param id path string true "Application ID"
// @Param app body routes.UpdateApplicationRequest true "Update application"
// @Success 200 {object} routes.ApplicationDTO
// @Router /applications/{id} [put]
func updateApplication(c *gin.Context) {
	db := getDB(c)
	if db == nil {
		return
	}
	id := c.Param("id")
	var input UpdateApplicationRequest
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
	if input.WebsiteURL != nil {
		updates["website_url"] = *input.WebsiteURL
	}
	if input.PrivacyPolicyURL != nil {
		updates["privacy_policy_url"] = *input.PrivacyPolicyURL
	}
	if input.ClientSecret != nil {
		updates["client_secret"] = *input.ClientSecret
	}
	q := query.Use(db)

	// fetch application to check ownership
	appModel, err := q.Application.Where(query.Application.ID.Eq(id)).First()
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// auth user required for owner check
	ui, exists := c.Get("user")
	if !exists {
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "認証が必要です"})
		return
	}
	authUser, ok := ui.(*model.User)
	if !ok || authUser == nil {
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "ユーザー情報が取得できませんでした"})
		return
	}

	// allow if user has APP_UPDATE permission or is owner
	perms, _ := middleware.GetUserPermissions(authUser.ID, db)
	if !perms.HasPermission(constants.APP_UPDATE) && appModel.UserID != authUser.ID {
		c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "この操作を実行する権限がありません"})
		return
	}

	if len(updates) > 0 {
		if _, err := q.Application.Where(query.Application.ID.Eq(id)).Updates(updates); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
	}
	updated, _ := q.Application.Where(query.Application.ID.Eq(id)).First()
	resp := ApplicationDTO{
		ID:               updated.ID,
		Name:             updated.Name,
		Description:      ptrToString(updated.Description),
		WebsiteURL:       ptrToString(updated.WebsiteURL),
		PrivacyPolicyURL: ptrToString(updated.PrivacyPolicyURL),
		UserID:           updated.UserID,
	}
	c.JSON(http.StatusOK, resp)
}

func deleteApplication(c *gin.Context) {
	db := getDB(c)
	if db == nil {
		return
	}
	id := c.Param("id")
	q := query.Use(db)

	// fetch application to check ownership
	appModel, err := q.Application.Where(query.Application.ID.Eq(id)).First()
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// auth user required for owner check
	ui, exists := c.Get("user")
	if !exists {
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "認証が必要です"})
		return
	}
	authUser, ok := ui.(*model.User)
	if !ok || authUser == nil {
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "ユーザー情報が取得できませんでした"})
		return
	}

	// allow if user has APP_DELETE permission or is owner
	perms, _ := middleware.GetUserPermissions(authUser.ID, db)
	if !perms.HasPermission(constants.APP_DELETE) && appModel.UserID != authUser.ID {
		c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "この操作を実行する権限がありません"})
		return
	}

	if _, err := q.Application.Delete(&model.Application{ID: id}); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.Status(http.StatusNoContent)
}

// listOwnersForApplication godoc
// @Summary List owners for an application
// @Description Get the user(s) who own the application
// @Tags applications
// @Produce json
// @Param id path string true "Application ID"
// @Success 200 {object} routes.UserListResponse
// @Router /applications/{id}/owners [get]
func listOwnersForApplication(c *gin.Context) {
	db := getDB(c)
	if db == nil {
		return
	}
	id := c.Param("id")
	q := query.Use(db)
	a, err := q.Application.Where(query.Application.ID.Eq(id)).First()
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
		return
	}
	// application has a single UserID owner; return as list for compatibility
	user, err := q.User.Where(query.User.ID.Eq(a.UserID)).First()
	if err != nil {
		// owner not found -> empty list
		c.JSON(http.StatusOK, UserListResponse{Data: []UserDTO{}})
		return
	}
	dto := UserDTO{
		ID:                user.ID,
		CustomID:          user.CustomID,
		Email:             user.Email,
		ExternalEmail:     user.ExternalEmail,
		EmailVerified:     user.EmailVerified,
		AffiliationPeriod: ptrToString(user.AffiliationPeriod),
		Status:            user.Status,
		CreatedAt:         user.CreatedAt,
		UpdatedAt:         user.UpdatedAt,
	}
	if p, err := q.Profile.Where(query.Profile.UserID.Eq(user.ID)).First(); err == nil {
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
	c.JSON(http.StatusOK, UserListResponse{Data: []UserDTO{dto}})
}

// addOwnerForApplication godoc
// @Summary Assign an owner to an application
// @Description Set or replace the owner (user) of the application
// @Tags applications
// @Accept json
// @Produce json
// @Param id path string true "Application ID"
// @Param body body routes.CreateApplicationOwnerRequest true "Assign owner"
// @Success 200 {object} routes.ApplicationDTO
// @Router /applications/{id}/owners [post]
func addOwnerForApplication(c *gin.Context) {
	db := getDB(c)
	if db == nil {
		return
	}
	id := c.Param("id")
	var input CreateApplicationOwnerRequest
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	q := query.Use(db)
	app, err := q.Application.Where(query.Application.ID.Eq(id)).First()
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "application not found"})
		return
	}
	// ensure target user exists
	if _, err := q.User.Where(query.User.ID.Eq(input.UserID)).First(); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
		return
	}
	// if already owner, return 409
	if app.UserID == input.UserID {
		c.JSON(http.StatusConflict, gin.H{"error": "user already owner"})
		return
	}
	// update owner
	if _, err := q.Application.Where(query.Application.ID.Eq(id)).Updates(map[string]interface{}{"user_id": input.UserID}); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	updated, _ := q.Application.Where(query.Application.ID.Eq(id)).First()
	resp := ApplicationDTO{
		ID:               updated.ID,
		Name:             updated.Name,
		Description:      ptrToString(updated.Description),
		WebsiteURL:       ptrToString(updated.WebsiteURL),
		PrivacyPolicyURL: ptrToString(updated.PrivacyPolicyURL),
		UserID:           updated.UserID,
	}
	c.JSON(http.StatusOK, resp)
}

// listRedirectURIsForApplication godoc
// @Summary List redirect URIs for an application
// @Description Get redirect URIs registered for a given application
// @Tags applications
// @Produce json
// @Param id path string true "Application ID"
// @Success 200 {array} RedirectURIDTO
// @Router /applications/{id}/redirect_uris [get]
func listRedirectURIsForApplication(c *gin.Context) {
	db := getDB(c)
	if db == nil {
		return
	}
	id := c.Param("id")
	q := query.Use(db)
	results, err := q.RedirectURI.Where(q.RedirectURI.ApplicationID.Eq(id)).Find()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	response := make([]RedirectURIDTO, len(results))
	for i, r := range results {
		response[i] = RedirectURIDTO{
			ApplicationID: r.ApplicationID,
			URI:           r.URI,
			CreatedAt:     r.CreatedAt,
			UpdatedAt:     r.UpdatedAt,
		}
	}
	c.JSON(http.StatusOK, response)
}

// createRedirectURIForApplication godoc
// @Summary Create redirect URI for an application
// @Description Register a new redirect URI for the application
// @Tags applications
// @Accept json
// @Produce json
// @Param id path string true "Application ID"
// @Param body body map[string]string true "payload: {\"uri\": \"https://...\" }"
// @Success 201 {object} RedirectURIDTO
// @Router /applications/{id}/redirect_uris [post]
func createRedirectURIForApplication(c *gin.Context) {
	db := getDB(c)
	if db == nil {
		return
	}
	id := c.Param("id")
	var body struct {
		URI string `json:"uri"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if body.URI == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "uri required"})
		return
	}
	q := query.Use(db)
	// 重複チェック: 同じURIが既に登録されている場合は409返す
	if existing, err := q.RedirectURI.Where(q.RedirectURI.ApplicationID.Eq(id), q.RedirectURI.URI.Eq(body.URI)).First(); err == nil && existing != nil {
		c.JSON(http.StatusConflict, gin.H{"error": "redirect uri already exists"})
		return
	} else if err != nil && err != gorm.ErrRecordNotFound {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	r := &model.RedirectURI{ApplicationID: id, URI: body.URI}
	if err := q.RedirectURI.Create(r); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	response := RedirectURIDTO{
		ApplicationID: r.ApplicationID,
		URI:           r.URI,
		CreatedAt:     r.CreatedAt,
		UpdatedAt:     r.UpdatedAt,
	}
	c.JSON(http.StatusCreated, response)
}

// deleteRedirectURIForApplication godoc
// @Summary Delete redirect URI for an application
// @Description Delete a registered redirect URI by application id and uri (query param)
// @Tags applications
// @Produce json
// @Param id path string true "Application ID"
// @Param uri query string true "Redirect URI"
// @Success 200 {object} map[string]string
// @Router /applications/{id}/redirect_uris [delete]
func deleteRedirectURIForApplication(c *gin.Context) {
	db := getDB(c)
	if db == nil {
		return
	}
	id := c.Param("id")
	uri := c.Query("uri")
	if uri == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "uri required"})
		return
	}
	q := query.Use(db)
	r, err := q.RedirectURI.Where(q.RedirectURI.ApplicationID.Eq(id), q.RedirectURI.URI.Eq(uri)).First()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if r == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "redirect uri not found"})
		return
	}
	if _, err := q.RedirectURI.Delete(r); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "deleted"})
}
