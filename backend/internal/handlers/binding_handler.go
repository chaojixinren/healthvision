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
		httputil.Unauthorized(c, "请先登录")
		return
	}

	var req createBindingRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httputil.ErrorJSON(c, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}

	binding, err := h.bindings.Create(c.Request.Context(), user.ID, req.ToEmail)
	if errors.Is(err, services.ErrBindingSelf) {
		httputil.ErrorJSON(c, http.StatusBadRequest, "binding_self", "不能绑定自己")
		return
	}
	if errors.Is(err, services.ErrBindingSameType) {
		httputil.ErrorJSON(c, http.StatusBadRequest, "binding_same_type", "只能在老人和子女账户之间绑定")
		return
	}
	if errors.Is(err, services.ErrBindingDuplicate) {
		httputil.ErrorJSON(c, http.StatusConflict, "binding_duplicate", "绑定关系已存在")
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
		httputil.Unauthorized(c, "请先登录")
		return
	}

	bindings, err := h.bindings.ListByUser(c.Request.Context(), user.ID, user.IsOld)
	if err != nil {
		httputil.ErrorJSON(c, http.StatusInternalServerError, "list_bindings_failed", "获取绑定列表失败")
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
		httputil.Unauthorized(c, "请先登录")
		return
	}

	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		httputil.ErrorJSON(c, http.StatusBadRequest, "invalid_id", "无效的绑定 ID")
		return
	}

	var req respondBindingRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httputil.ErrorJSON(c, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}

	binding, err := h.bindings.Respond(c.Request.Context(), user.ID, uint(id), req.Accept)
	if errors.Is(err, services.ErrBindingNotFound) {
		httputil.ErrorJSON(c, http.StatusNotFound, "binding_not_found", "绑定关系不存在")
		return
	}
	if errors.Is(err, services.ErrBindingNotPending) {
		httputil.ErrorJSON(c, http.StatusBadRequest, "binding_not_pending", "绑定不是待处理状态")
		return
	}
	if errors.Is(err, services.ErrBindingNotPermitted) {
		httputil.ErrorJSON(c, http.StatusForbidden, "not_permitted", "只有老人本人可以接受或拒绝绑定")
		return
	}
	if err != nil {
		httputil.ErrorJSON(c, http.StatusInternalServerError, "respond_failed", "响应绑定失败")
		return
	}

	c.JSON(http.StatusOK, gin.H{"binding": binding})
}

func (h *BindingHandler) Delete(c *gin.Context) {
	user, ok := CurrentUser(c)
	if !ok {
		httputil.Unauthorized(c, "请先登录")
		return
	}

	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		httputil.ErrorJSON(c, http.StatusBadRequest, "invalid_id", "无效的绑定 ID")
		return
	}

	err = h.bindings.Delete(c.Request.Context(), user.ID, uint(id))
	if errors.Is(err, services.ErrBindingNotFound) {
		httputil.ErrorJSON(c, http.StatusNotFound, "binding_not_found", "绑定关系不存在")
		return
	}
	if errors.Is(err, services.ErrBindingNotPermitted) {
		httputil.ErrorJSON(c, http.StatusForbidden, "not_permitted", "无权删除此绑定")
		return
	}
	if err != nil {
		httputil.ErrorJSON(c, http.StatusInternalServerError, "delete_failed", "删除绑定失败")
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "绑定已删除"})
}

func (h *BindingHandler) ChangeIdentity(c *gin.Context) {
	user, ok := CurrentUser(c)
	if !ok {
		httputil.Unauthorized(c, "请先登录")
		return
	}

	if err := h.bindings.ChangeIdentity(c.Request.Context(), user); err != nil {
		if errors.Is(err, services.ErrHasActiveBindings) {
			httputil.ErrorJSON(c, http.StatusBadRequest, "has_active_bindings", "切换身份前必须先解绑所有关系")
			return
		}
		httputil.ErrorJSON(c, http.StatusInternalServerError, "identity_change_failed", "身份切换失败")
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"user":   toUserResponse(user),
		"message": "身份已切换",
	})
}

func (h *BindingHandler) SearchUsers(c *gin.Context) {
	user, ok := CurrentUser(c)
	if !ok {
		httputil.Unauthorized(c, "请先登录")
		return
	}

	query := c.Query("q")
	if query == "" {
		httputil.ErrorJSON(c, http.StatusBadRequest, "invalid_request", "请输入搜索关键词")
		return
	}

	users, err := h.bindings.SearchUsers(c.Request.Context(), query, user.ID)
	if err != nil {
		httputil.ErrorJSON(c, http.StatusInternalServerError, "search_failed", "搜索用户失败")
		return
	}

	results := make([]userResponse, 0, len(users))
	for _, u := range users {
		results = append(results, toUserResponse(&u))
	}

	c.JSON(http.StatusOK, gin.H{"users": results})
}
