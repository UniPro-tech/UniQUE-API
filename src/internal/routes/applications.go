package routes

import (
	"net/http"

	"github.com/UniPro-tech/UniQUE-API/internal/model"
	"github.com/UniPro-tech/UniQUE-API/internal/query"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

func RegisterApplicationRoutes(r *gin.Engine) {
	g := r.Group("/applications")
	{
		g.GET("", listApplications)
		g.POST("", createApplication)
		g.GET(":id", getApplication)
		g.GET(":id/owners", listOwnersForApplication)
		g.POST(":id/owners", addOwnerForApplication)
		g.PUT(":id", updateApplication)
		g.DELETE(":id", deleteApplication)
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
	apps, err := q.Application.Find()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
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
		ID:               uuid.NewString(),
		Name:             input.Name,
		Description:      input.Description,
		WebsiteURL:       input.WebsiteURL,
		PrivacyPolicyURL: input.PrivacyPolicyURL,
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
		Description:      app.Description,
		WebsiteURL:       app.WebsiteURL,
		PrivacyPolicyURL: app.PrivacyPolicyURL,
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
		Description:      a.Description,
		WebsiteURL:       a.WebsiteURL,
		PrivacyPolicyURL: a.PrivacyPolicyURL,
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
		Description:      updated.Description,
		WebsiteURL:       updated.WebsiteURL,
		PrivacyPolicyURL: updated.PrivacyPolicyURL,
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
		AffiliationPeriod: user.AffiliationPeriod,
		Status:            user.Status,
		CreatedAt:         user.CreatedAt,
		UpdatedAt:         user.UpdatedAt,
	}
	if p, err := q.Profile.Where(query.Profile.UserID.Eq(user.ID)).First(); err == nil {
		dto.Profile = &ProfileDTO{
			UserID:      p.UserID,
			DisplayName: p.DisplayName,
			Bio:         p.Bio,
			WebsiteURL:  p.WebsiteURL,
			JoinedAt:    p.JoinedAt,
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
		Description:      updated.Description,
		WebsiteURL:       updated.WebsiteURL,
		PrivacyPolicyURL: updated.PrivacyPolicyURL,
		UserID:           updated.UserID,
	}
	c.JSON(http.StatusOK, resp)
}
