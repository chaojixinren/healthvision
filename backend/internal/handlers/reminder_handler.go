package handlers

import (
	"context"
	"net/http"
	"strconv"

	"healthvision/backend/internal/httputil"
	"healthvision/backend/internal/models"
	"healthvision/backend/internal/services"

	"github.com/gin-gonic/gin"
)

type MedicineNameLookup interface {
	FindByIDAny(ctx context.Context, id uint) (*models.Medicine, error)
}

type ReminderHandler struct {
	svc       *services.ReminderService
	medLookup MedicineNameLookup
}

func NewReminderHandler(svc *services.ReminderService, medLookup MedicineNameLookup) *ReminderHandler {
	return &ReminderHandler{svc: svc, medLookup: medLookup}
}

type reminderRequest struct {
	MedicineID   uint   `json:"medicine_id" binding:"required"`
	Time         string `json:"time" binding:"required"`
	TargetUserID uint   `json:"target_user_id"`
}

type reminderUpdateRequest struct {
	Time    string `json:"time" binding:"required"`
	Enabled *bool  `json:"enabled" binding:"required"`
}

type reminderResponse struct {
	ID           uint   `json:"id"`
	UserID       uint   `json:"user_id"`
	MedicineID   uint   `json:"medicine_id"`
	MedicineName string `json:"medicine_name"`
	Time         string `json:"time"`
	Enabled      bool   `json:"enabled"`
	CreatedBy    uint   `json:"created_by"`
	CreatedAt    string `json:"created_at"`
	UpdatedAt    string `json:"updated_at"`
}

type listRemindersResponse struct {
	Data       []reminderResponse `json:"data"`
	Pagination paginationInfo      `json:"pagination"`
}

func (h *ReminderHandler) medicineName(ctx context.Context, id uint) string {
	if h.medLookup == nil {
		return ""
	}
	m, err := h.medLookup.FindByIDAny(ctx, id)
	if err != nil {
		return "未知药品"
	}
	return m.Name
}

func (h *ReminderHandler) Create(c *gin.Context) {
	var req reminderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httputil.ErrorJSON(c, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}

	user, ok := CurrentUser(c)
	if !ok {
		httputil.Unauthorized(c, "请先登录")
		return
	}

	targetUserID := req.TargetUserID
	if targetUserID == 0 {
		targetUserID = user.ID
	}

	reminder, err := h.svc.Create(c.Request.Context(), user.ID, targetUserID, req.MedicineID, req.Time)
	if err != nil {
		if err == services.ErrMedicineNotFound {
			httputil.ErrorJSON(c, http.StatusNotFound, "not_found", "药品不存在")
			return
		}
		if err == services.ErrInvalidTime {
			httputil.ErrorJSON(c, http.StatusBadRequest, "invalid_time", err.Error())
			return
		}
		if err == services.ErrNotBound {
			httputil.ErrorJSON(c, http.StatusBadRequest, "not_bound", "未与该用户建立绑定关系")
			return
		}
		httputil.ErrorJSON(c, http.StatusInternalServerError, "create_failed", "创建提醒失败")
		return
	}

	name := h.medicineName(c.Request.Context(), reminder.MedicineID)
	c.JSON(http.StatusCreated, h.toResponse(reminder, name))
}

func (h *ReminderHandler) List(c *gin.Context) {
	user, ok := CurrentUser(c)
	if !ok {
		httputil.Unauthorized(c, "请先登录")
		return
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	perPage, _ := strconv.Atoi(c.DefaultQuery("per_page", "20"))

	var medicineID *uint
	if raw := c.Query("medicine_id"); raw != "" {
		id, err := strconv.ParseUint(raw, 10, 32)
		if err == nil {
			v := uint(id)
			medicineID = &v
		}
	}

	var reminders []models.Reminder
	var total int64
	var err error

	if user.IsOld {
		reminders, total, err = h.svc.List(c.Request.Context(), user.ID, medicineID, page, perPage)
	} else {
		reminders, total, err = h.svc.ListByCreator(c.Request.Context(), user.ID, medicineID, page, perPage)
	}
	if err != nil {
		httputil.ErrorJSON(c, http.StatusInternalServerError, "list_failed", "获取提醒列表失败")
		return
	}

	// batch lookup medicine names
	medNames := make(map[uint]string)
	for _, r := range reminders {
		if _, ok := medNames[r.MedicineID]; ok {
			continue
		}
		medNames[r.MedicineID] = h.medicineName(c.Request.Context(), r.MedicineID)
	}

	resp := listRemindersResponse{
		Data: make([]reminderResponse, len(reminders)),
		Pagination: paginationInfo{
			Page:    page,
			PerPage: perPage,
			Total:   total,
		},
	}
	for i, r := range reminders {
		resp.Data[i] = h.toResponse(&r, medNames[r.MedicineID])
	}

	c.JSON(http.StatusOK, resp)
}

func (h *ReminderHandler) Get(c *gin.Context) {
	user, ok := CurrentUser(c)
	if !ok {
		httputil.Unauthorized(c, "请先登录")
		return
	}

	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		httputil.ErrorJSON(c, http.StatusBadRequest, "invalid_id", "无效的提醒 ID")
		return
	}

	reminder, err := h.svc.GetByID(c.Request.Context(), uint(id), user.ID)
	if err != nil {
		if err == services.ErrReminderNotFound {
			httputil.ErrorJSON(c, http.StatusNotFound, "not_found", "提醒不存在")
			return
		}
		httputil.ErrorJSON(c, http.StatusInternalServerError, "get_failed", "获取提醒失败")
		return
	}

	name := h.medicineName(c.Request.Context(), reminder.MedicineID)
	c.JSON(http.StatusOK, h.toResponse(reminder, name))
}

func (h *ReminderHandler) Update(c *gin.Context) {
	user, ok := CurrentUser(c)
	if !ok {
		httputil.Unauthorized(c, "请先登录")
		return
	}
	if user.IsOld {
		httputil.ErrorJSON(c, http.StatusForbidden, "forbidden", "只有子女可以编辑提醒")
		return
	}

	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		httputil.ErrorJSON(c, http.StatusBadRequest, "invalid_id", "无效的提醒 ID")
		return
	}

	var req reminderUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httputil.ErrorJSON(c, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}

	reminder, err := h.svc.Update(c.Request.Context(), uint(id), user.ID, req.Time, *req.Enabled)
	if err != nil {
		if err == services.ErrReminderNotFound {
			httputil.ErrorJSON(c, http.StatusNotFound, "not_found", "提醒不存在")
			return
		}
		if err == services.ErrInvalidTime {
			httputil.ErrorJSON(c, http.StatusBadRequest, "invalid_time", err.Error())
			return
		}
		httputil.ErrorJSON(c, http.StatusInternalServerError, "update_failed", "更新提醒失败")
		return
	}

	name := h.medicineName(c.Request.Context(), reminder.MedicineID)
	c.JSON(http.StatusOK, h.toResponse(reminder, name))
}

func (h *ReminderHandler) Delete(c *gin.Context) {
	user, ok := CurrentUser(c)
	if !ok {
		httputil.Unauthorized(c, "请先登录")
		return
	}
	if user.IsOld {
		httputil.ErrorJSON(c, http.StatusForbidden, "forbidden", "只有子女可以删除提醒")
		return
	}

	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		httputil.ErrorJSON(c, http.StatusBadRequest, "invalid_id", "无效的提醒 ID")
		return
	}

	if err := h.svc.Delete(c.Request.Context(), uint(id), user.ID); err != nil {
		httputil.ErrorJSON(c, http.StatusInternalServerError, "delete_failed", "删除提醒失败")
		return
	}

	c.Status(http.StatusNoContent)
}

func (h *ReminderHandler) toResponse(r *models.Reminder, medicineName string) reminderResponse {
	return reminderResponse{
		ID:           r.ID,
		UserID:       r.UserID,
		MedicineID:   r.MedicineID,
		MedicineName: medicineName,
		Time:         r.Time,
		Enabled:      r.Enabled,
		CreatedBy:    r.CreatedBy,
		CreatedAt:    r.CreatedAt.Format(http.TimeFormat),
		UpdatedAt:    r.UpdatedAt.Format(http.TimeFormat),
	}
}
