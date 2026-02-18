package routes

import (
	"net/http"
	"strconv"
	"time"

	"github.com/UniPro-tech/UniQUE-API/internal/constants"
	"github.com/UniPro-tech/UniQUE-API/internal/middleware"
	"github.com/UniPro-tech/UniQUE-API/internal/model"
	"github.com/UniPro-tech/UniQUE-API/internal/query"
	"github.com/gin-gonic/gin"
	"github.com/oklog/ulid/v2"
)

// RegisterAnnouncementRoutes registers announcement routes
func RegisterAnnouncementRoutes(r *gin.Engine) {
	g := r.Group("/announcements")
	{
		g.GET("", listAnnouncements)
		g.GET(":id", getAnnouncement)

		// 管理系
		g.POST("", middleware.RequirePermission(constants.ANNOUNCEMENT_CREATE), createAnnouncement)
		g.PUT(":id", middleware.RequirePermission(constants.ANNOUNCEMENT_UPDATE), updateAnnouncement)
		g.DELETE(":id", middleware.RequirePermission(constants.ANNOUNCEMENT_DELETE), deleteAnnouncement)
		g.POST(":id/pin", middleware.RequirePermission(constants.ANNOUNCEMENT_PIN), pinAnnouncement)
	}
}

type CreateAnnouncementRequest struct {
	Title    string `json:"title" binding:"required"`
	Content  string `json:"content" binding:"required"`
	IsPinned *bool  `json:"is_pinned,omitempty"`
}

type UpdateAnnouncementRequest struct {
	Title    *string `json:"title,omitempty"`
	Content  *string `json:"content,omitempty"`
	IsPinned *bool   `json:"is_pinned,omitempty"`
}

type AnnouncementDTO struct {
	ID        string    `json:"id"`
	Title     string    `json:"title"`
	Content   string    `json:"content"`
	CreatedBy string    `json:"created_by"`
	IsPinned  bool      `json:"is_pinned"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type AnnouncementListResponse struct {
	Data []AnnouncementDTO `json:"data"`
}

// listAnnouncements godoc
// @Summary List announcements
// @Description List announcements, pinned first. Use `limit` query to limit results and `deleted` to include deleted records.
// @Tags announcements
// @Produce json
// @Param limit query int false "Limit number of announcements"
// @Param deleted query bool false "Include deleted announcements"
// @Success 200 {object} routes.AnnouncementListResponse
// @Router /announcements [get]
func listAnnouncements(c *gin.Context) {
	db := getDB(c)
	if db == nil {
		return
	}
	q := query.Use(db)
	// limit パラメータの読み取り（省略時は100件）
	limitStr := c.Query("limit")
	if limitStr == "" {
		limitStr = "100"
	}
	// deletedパラメータの読み取り（省略時はfalse）
	deletedStr := c.Query("deleted")
	var (
		dao = q.Announcement.Order(query.Announcement.IsPinned.Desc(), query.Announcement.CreatedAt.Desc())
	)
	if deletedStr != "" {
		deleted, errp := strconv.ParseBool(deletedStr)
		if errp != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid deleted parameter"})
			return
		}
		if deleted {
			// deleted=true は ANNOUNCEMENT_UPDATE 権限があるユーザーのみ許可
			ui, exists := c.Get("user")
			if !exists {
				c.JSON(http.StatusForbidden, gin.H{"error": "permission denied"})
				return
			}
			su, ok := ui.(*model.User)
			if !ok || su == nil {
				c.JSON(http.StatusForbidden, gin.H{"error": "permission denied"})
				return
			}
			perms, _ := middleware.GetUserPermissions(su.ID, db)
			if !perms.HasPermission(constants.ANNOUNCEMENT_UPDATE) {
				c.JSON(http.StatusForbidden, gin.H{"error": "permission denied"})
				return
			}
			// 権限あり -> 削除済み含めて返す（フィルタなし）
		} else {
			dao = dao.Where(query.Announcement.DeletedAt.IsNull())
		}
	} else {
		dao = dao.Where(query.Announcement.DeletedAt.IsNull())
	}
	if limitStr != "" {
		limit, errp := strconv.Atoi(limitStr)
		if errp != nil || limit < 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid limit"})
			return
		}
		if limit > 0 {
			dao = dao.Limit(limit)
		}
	}
	anns, err := dao.Find()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	out := make([]AnnouncementDTO, 0, len(anns))
	for _, a := range anns {
		out = append(out, AnnouncementDTO{
			ID:        a.ID,
			Title:     a.Title,
			Content:   a.Content,
			CreatedBy: a.CreatedBy,
			IsPinned:  a.IsPinned,
			CreatedAt: a.CreatedAt,
			UpdatedAt: a.UpdatedAt,
		})
	}
	c.JSON(http.StatusOK, AnnouncementListResponse{Data: out})
}

// getAnnouncement godoc
// @Summary Get an announcement
// @Description Get a single announcement by ID
// @Tags announcements
// @Produce json
// @Param id path string true "Announcement ID"
// @Success 200 {object} routes.AnnouncementDTO
// @Router /announcements/{id} [get]
func getAnnouncement(c *gin.Context) {
	db := getDB(c)
	if db == nil {
		return
	}
	id := c.Param("id")
	q := query.Use(db)
	a, err := q.Announcement.Where(query.Announcement.ID.Eq(id)).First()
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
		return
	}
	dto := AnnouncementDTO{
		ID:        a.ID,
		Title:     a.Title,
		Content:   a.Content,
		CreatedBy: a.CreatedBy,
		IsPinned:  a.IsPinned,
		CreatedAt: a.CreatedAt,
		UpdatedAt: a.UpdatedAt,
	}
	c.JSON(http.StatusOK, dto)
}

// createAnnouncement godoc
// @Summary Create an announcement
// @Description Create a new announcement
// @Tags announcements
// @Accept json
// @Produce json
// @Param announcement body routes.CreateAnnouncementRequest true "Create announcement"
// @Success 201 {object} routes.AnnouncementDTO
// @Router /announcements [post]
func createAnnouncement(c *gin.Context) {
	db := getDB(c)
	if db == nil {
		return
	}
	var input CreateAnnouncementRequest
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	// 作成者
	userObj, _ := c.Get("user")
	createdBy := ""
	if um, ok := userObj.(*model.User); ok && um != nil {
		createdBy = um.ID
	}
	ann := &model.Announcement{
		ID:        ulid.Make().String(),
		Title:     input.Title,
		Content:   input.Content,
		CreatedBy: createdBy,
	}
	if input.IsPinned != nil {
		ann.IsPinned = *input.IsPinned
	}
	q := query.Use(db)
	if err := q.Announcement.Create(ann); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	dto := AnnouncementDTO{
		ID:        ann.ID,
		Title:     ann.Title,
		Content:   ann.Content,
		CreatedBy: ann.CreatedBy,
		IsPinned:  ann.IsPinned,
		CreatedAt: ann.CreatedAt,
		UpdatedAt: ann.UpdatedAt,
	}
	c.JSON(http.StatusCreated, dto)
}

// updateAnnouncement godoc
// @Summary Update an announcement
// @Description Update announcement fields
// @Tags announcements
// @Accept json
// @Produce json
// @Param id path string true "Announcement ID"
// @Param announcement body routes.UpdateAnnouncementRequest true "Update announcement"
// @Success 200 {object} routes.AnnouncementDTO
// @Router /announcements/{id} [put]
func updateAnnouncement(c *gin.Context) {
	db := getDB(c)
	if db == nil {
		return
	}
	id := c.Param("id")
	var input UpdateAnnouncementRequest
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	updates := map[string]interface{}{}
	if input.Title != nil {
		updates["title"] = *input.Title
	}
	if input.Content != nil {
		updates["content"] = *input.Content
	}
	if input.IsPinned != nil {
		updates["is_pinned"] = *input.IsPinned
	}
	q := query.Use(db)
	if len(updates) > 0 {
		if _, err := q.Announcement.Where(query.Announcement.ID.Eq(id)).Updates(updates); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
	}
	a, err := q.Announcement.Where(query.Announcement.ID.Eq(id)).First()
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
		return
	}
	dto := AnnouncementDTO{
		ID:        a.ID,
		Title:     a.Title,
		Content:   a.Content,
		CreatedBy: a.CreatedBy,
		IsPinned:  a.IsPinned,
		CreatedAt: a.CreatedAt,
		UpdatedAt: a.UpdatedAt,
	}
	c.JSON(http.StatusOK, dto)
}

// deleteAnnouncement godoc
// @Summary Delete an announcement
// @Description Delete an announcement by ID
// @Tags announcements
// @Produce json
// @Param id path string true "Announcement ID"
// @Success 204 {string} string
// @Router /announcements/{id} [delete]
func deleteAnnouncement(c *gin.Context) {
	db := getDB(c)
	if db == nil {
		return
	}
	id := c.Param("id")
	q := query.Use(db)
	if _, err := q.Announcement.Delete(&model.Announcement{ID: id}); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.Status(http.StatusNoContent)
}

// pinAnnouncement godoc
// @Summary Pin or unpin an announcement
// @Description Toggle pin state for an announcement
// @Tags announcements
// @Accept json
// @Produce json
// @Param id path string true "Announcement ID"
// @Param body body map[string]bool true "{\"pin\": true}"
// @Success 200 {object} routes.AnnouncementDTO
// @Router /announcements/{id}/pin [post]
func pinAnnouncement(c *gin.Context) {
	db := getDB(c)
	if db == nil {
		return
	}
	id := c.Param("id")
	// toggle pin or set true if absent
	var input struct {
		Pin bool `json:"pin"`
	}
	_ = c.ShouldBindJSON(&input)
	q := query.Use(db)
	if _, err := q.Announcement.Where(query.Announcement.ID.Eq(id)).Update(query.Announcement.IsPinned, input.Pin); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	a, err := q.Announcement.Where(query.Announcement.ID.Eq(id)).First()
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
		return
	}
	dto := AnnouncementDTO{ID: a.ID, Title: a.Title, Content: a.Content, CreatedBy: a.CreatedBy, IsPinned: a.IsPinned, CreatedAt: a.CreatedAt, UpdatedAt: a.UpdatedAt}
	c.JSON(http.StatusOK, dto)
}
