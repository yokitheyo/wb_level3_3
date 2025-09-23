package http

import (
	"net/http"
	"strconv"

	"github.com/wb-go/wbf/ginext"
	"github.com/wb-go/wbf/zlog"
	"github.com/yokitheyo/wb_level3_3/internal/domain"
	"github.com/yokitheyo/wb_level3_3/internal/dto"
)

// CommentHandler обрабатывает HTTP-запросы по комментариям.
type CommentHandler struct {
	service domain.CommentService
}

// NewCommentHandler создаёт новый CommentHandler.
func NewCommentHandler(service domain.CommentService) *CommentHandler {
	return &CommentHandler{service: service}
}

// RegisterRoutes регистрирует маршруты комментариев в Engine.
func (h *CommentHandler) RegisterRoutes(engine *ginext.Engine) {
	group := engine.Group("/comments")
	group.POST("", h.CreateComment)
	group.GET("", h.GetComments)
	group.DELETE("/:id", h.DeleteComment)
	group.GET("/search", h.SearchComments)
}

// CreateComment POST /comments
func (h *CommentHandler) CreateComment(c *ginext.Context) {
	var req dto.CreateCommentRequest
	if err := c.BindJSON(&req); err != nil {
		zlog.Logger.Warn().Err(err).Msg("invalid request body")
		c.JSON(http.StatusBadRequest, ginext.H{"error": "invalid request"})
		return
	}

	comment, err := h.service.CreateComment(c, req.ParentID, req.Author, req.Content)
	if err != nil {
		zlog.Logger.Error().Err(err).Msg("CreateComment failed")
		c.JSON(http.StatusInternalServerError, ginext.H{"error": "failed to create comment"})
		return
	}

	c.JSON(http.StatusCreated, mapToCommentResponse(comment))

}

// GetComments GET /comments?parent={id}&limit=&offset=&sort=
func (h *CommentHandler) GetComments(c *ginext.Context) {
	var parentID *int64
	if parentStr := c.Query("parent"); parentStr != "" {
		id, err := strconv.ParseInt(parentStr, 10, 64)
		if err != nil {
			c.JSON(http.StatusBadRequest, ginext.H{"error": "invalid parent id"})
			return
		}
		parentID = &id
	}

	limit := 10
	if l := c.Query("limit"); l != "" {
		if val, err := strconv.Atoi(l); err == nil {
			limit = val
		}
	}
	offset := 0
	if o := c.Query("offset"); o != "" {
		if val, err := strconv.Atoi(o); err == nil {
			offset = val
		}
	}
	sort := c.Query("sort")

	comments, err := h.service.GetThread(c, parentID, limit, offset, sort)
	if err != nil {
		zlog.Logger.Error().Err(err).Msg("GetThread failed")
		c.JSON(http.StatusInternalServerError, ginext.H{"error": "failed to get comments"})
		return
	}

	c.JSON(http.StatusOK, mapToCommentResponses(comments))
}

// DeleteComment DELETE /comments/:id
func (h *CommentHandler) DeleteComment(c *ginext.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, ginext.H{"error": "invalid id"})
		return
	}

	if err := h.service.DeleteThread(c, id); err != nil {
		zlog.Logger.Error().Err(err).Msg("DeleteThread failed")
		c.JSON(http.StatusInternalServerError, ginext.H{"error": "failed to delete comment"})
		return
	}

	c.Status(http.StatusNoContent)
}

// SearchComments GET /comments/search?query=&limit=&offset=
func (h *CommentHandler) SearchComments(c *ginext.Context) {
	query := c.Query("query")
	if query == "" {
		c.JSON(http.StatusBadRequest, ginext.H{"error": "query cannot be empty"})
		return
	}

	limit := 10
	if l := c.Query("limit"); l != "" {
		if val, err := strconv.Atoi(l); err == nil {
			limit = val
		}
	}
	offset := 0
	if o := c.Query("offset"); o != "" {
		if val, err := strconv.Atoi(o); err == nil {
			offset = val
		}
	}

	comments, err := h.service.SearchComment(c, query, limit, offset)
	if err != nil {
		zlog.Logger.Error().Err(err).Msg("SearchComment failed")
		c.JSON(http.StatusInternalServerError, ginext.H{"error": "search failed"})
		return
	}

	c.JSON(http.StatusOK, comments)
}

func mapToCommentResponse(c *domain.Comment) *dto.CommentResponse {
	if c == nil {
		return nil
	}

	children := make([]*dto.CommentResponse, 0, len(c.Children))
	for _, ch := range c.Children {
		children = append(children, mapToCommentResponse(ch))
	}

	return &dto.CommentResponse{
		ID:        c.ID,
		ParentID:  c.ParentID,
		Content:   c.Content,
		Author:    c.Author,
		CreatedAt: c.CreatedAt,
		UpdatedAt: c.UpdatedAt,
		Deleted:   c.Deleted,
		Children:  children,
	}
}

func mapToCommentResponses(list []*domain.Comment) []*dto.CommentResponse {
	out := make([]*dto.CommentResponse, 0, len(list))
	for _, c := range list {
		out = append(out, mapToCommentResponse(c))
	}
	return out
}
