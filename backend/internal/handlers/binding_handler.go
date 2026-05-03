package handlers

import (
	"errors"
	"net/http"
	"strconv"

	"healthvision/backend/internal/httputil"
	"healthvision/backend/internal/models"
	"healthvision/backend/internal/services"

	"github.com/gin-gonic/gin"
)

type BindingHandler struct {
	bindings *services.BindingService
}

func NewBindingHandler(bindings *services.BindingService) *BindingHandler {
	return &BindingHandler{bindings: bindings}
}

type createBindingRequest struct {
	ToEmail string `json:"to_email" binding:"required,email"`
}

type respondBindingRequest struct {
	Accept bool `json:"accept"`
}

func (h *BindingHandler) Create(c *gin.Context) {
	user, ok := CurrentUser(c)
	if !ok {
		httputil.Unauthorized(c, "authentication required")
		return
	}

	var req createBindingRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httputil.ErrorJSON(c, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}

	binding, err := h.bindings.Create(c.Request.Context(), user.ID, req.ToEmail)
	if errors.Is(err, services.ErrBindingSelf) {
		httputil.ErrorJSON(c, http.StatusBadRequest, "binding_self", "cannot bind to yourself")
		return
	}
	if errors.Is(err, services.ErrBindingSameType) {
		httputil.ErrorJSON(c, http.StatusBadRequest, "binding_same_type", "can only bind between elder and child accounts")
		return
	}
	if errors.Is(err, services.ErrBindingDuplicate) {
		httputil.ErrorJSON(c, http.StatusConflict, "binding_duplicate", "binding already exists")
		return
	}
	if err != nil {
		httputil.ErrorJSON(c, http.StatusInternalServerError, "binding_failed", err.Error())
		return
	}

	c.JSON(http.StatusCreated, gin.H{"binding": binding})
}

func (h *BindingHandler) List(c *gin.Context) {
	user, ok := CurrentUser(c)
	if !ok {
		httputil.Unauthorized(c, "authentication required")
		return
	}

	bindings, err := h.bindings.ListByUser(c.Request.Context(), user.ID, user.IsOld)
	if err != nil {
		httputil.ErrorJSON(c, http.StatusInternalServerError, "list_bindings_failed", "failed to list bindings")
		return
	}

	if bindings == nil {
		bindings = []models.Binding{}
	}

	c.JSON(http.StatusOK, gin.H{"bindings": bindings})
}

func (h *BindingHandler) Respond(c *gin.Context) {
	user, ok := CurrentUser(c)
	if !ok {
		httputil.Unauthorized(c, "authentication required")
		return
	}

	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		httputil.ErrorJSON(c, http.StatusBadRequest, "invalid_id", "invalid binding id")
		return
	}

	var req respondBindingRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httputil.ErrorJSON(c, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}

	binding, err := h.bindings.Respond(c.Request.Context(), user.ID, uint(id), req.Accept)
	if errors.Is(err, services.ErrBindingNotFound) {
		httputil.ErrorJSON(c, http.StatusNotFound, "binding_not_found", "binding not found")
		return
	}
	if errors.Is(err, services.ErrBindingNotPending) {
		httputil.ErrorJSON(c, http.StatusBadRequest, "binding_not_pending", "binding is not in pending status")
		return
	}
	if errors.Is(err, services.ErrBindingNotPermitted) {
		httputil.ErrorJSON(c, http.StatusForbidden, "not_permitted", "only the elder can accept or reject a binding")
		return
	}
	if err != nil {
		httputil.ErrorJSON(c, http.StatusInternalServerError, "respond_failed", "failed to respond to binding")
		return
	}

	c.JSON(http.StatusOK, gin.H{"binding": binding})
}

func (h *BindingHandler) Delete(c *gin.Context) {
	user, ok := CurrentUser(c)
	if !ok {
		httputil.Unauthorized(c, "authentication required")
		return
	}

	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		httputil.ErrorJSON(c, http.StatusBadRequest, "invalid_id", "invalid binding id")
		return
	}

	err = h.bindings.Delete(c.Request.Context(), user.ID, uint(id))
	if errors.Is(err, services.ErrBindingNotFound) {
		httputil.ErrorJSON(c, http.StatusNotFound, "binding_not_found", "binding not found")
		return
	}
	if errors.Is(err, services.ErrBindingNotPermitted) {
		httputil.ErrorJSON(c, http.StatusForbidden, "not_permitted", "not permitted to delete this binding")
		return
	}
	if err != nil {
		httputil.ErrorJSON(c, http.StatusInternalServerError, "delete_failed", "failed to delete binding")
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "binding deleted"})
}

func (h *BindingHandler) ChangeIdentity(c *gin.Context) {
	user, ok := CurrentUser(c)
	if !ok {
		httputil.Unauthorized(c, "authentication required")
		return
	}

	if err := h.bindings.ChangeIdentity(c.Request.Context(), user); err != nil {
		if errors.Is(err, services.ErrHasActiveBindings) {
			httputil.ErrorJSON(c, http.StatusBadRequest, "has_active_bindings", "must unbind all relationships before changing identity")
			return
		}
		httputil.ErrorJSON(c, http.StatusInternalServerError, "identity_change_failed", "failed to change identity")
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"user":   toUserResponse(user),
		"message": "identity changed successfully",
	})
}

func (h *BindingHandler) SearchUsers(c *gin.Context) {
	user, ok := CurrentUser(c)
	if !ok {
		httputil.Unauthorized(c, "authentication required")
		return
	}

	query := c.Query("q")
	if query == "" {
		httputil.ErrorJSON(c, http.StatusBadRequest, "invalid_request", "search query is required")
		return
	}

	users, err := h.bindings.SearchUsers(c.Request.Context(), query, user.ID)
	if err != nil {
		httputil.ErrorJSON(c, http.StatusInternalServerError, "search_failed", "failed to search users")
		return
	}

	results := make([]userResponse, 0, len(users))
	for _, u := range users {
		results = append(results, toUserResponse(&u))
	}

	c.JSON(http.StatusOK, gin.H{"users": results})
}
