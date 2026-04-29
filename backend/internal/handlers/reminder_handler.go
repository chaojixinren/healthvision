package handlers

import (
	"net/http"
	"strconv"

	"healthvision/backend/internal/httputil"
	"healthvision/backend/internal/models"
	"healthvision/backend/internal/services"

	"github.com/gin-gonic/gin"
)

type ReminderHandler struct {
	svc *services.ReminderService
}

func NewReminderHandler(svc *services.ReminderService) *ReminderHandler {
	return &ReminderHandler{svc: svc}
}

type reminderRequest struct {
	MedicineID uint   `json:"medicine_id" binding:"required"`
	Time       string `json:"time" binding:"required"`
}

type reminderUpdateRequest struct {
	Time    string `json:"time" binding:"required"`
	Enabled *bool  `json:"enabled" binding:"required"`
}

type reminderResponse struct {
	ID         uint   `json:"id"`
	MedicineID uint   `json:"medicine_id"`
	Time       string `json:"time"`
	Enabled    bool   `json:"enabled"`
	CreatedAt  string `json:"created_at"`
	UpdatedAt  string `json:"updated_at"`
}

type listRemindersResponse struct {
	Data       []reminderResponse `json:"data"`
	Pagination paginationInfo      `json:"pagination"`
}

func (h *ReminderHandler) Create(c *gin.Context) {
	var req reminderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httputil.ErrorJSON(c, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}

	user, ok := CurrentUser(c)
	if !ok {
		httputil.Unauthorized(c, "authentication required")
		return
	}

	reminder, err := h.svc.Create(c.Request.Context(), user.ID, req.MedicineID, req.Time)
	if err != nil {
		if err == services.ErrMedicineNotFound {
			httputil.ErrorJSON(c, http.StatusNotFound, "not_found", "medicine not found")
			return
		}
		if err == services.ErrInvalidTime {
			httputil.ErrorJSON(c, http.StatusBadRequest, "invalid_time", err.Error())
			return
		}
		httputil.ErrorJSON(c, http.StatusInternalServerError, "create_failed", "failed to create reminder")
		return
	}

	c.JSON(http.StatusCreated, toReminderResponse(reminder))
}

func (h *ReminderHandler) List(c *gin.Context) {
	user, ok := CurrentUser(c)
	if !ok {
		httputil.Unauthorized(c, "authentication required")
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

	reminders, total, err := h.svc.List(c.Request.Context(), user.ID, medicineID, page, perPage)
	if err != nil {
		httputil.ErrorJSON(c, http.StatusInternalServerError, "list_failed", "failed to list reminders")
		return
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
		resp.Data[i] = toReminderResponse(&r)
	}

	c.JSON(http.StatusOK, resp)
}

func (h *ReminderHandler) Get(c *gin.Context) {
	user, ok := CurrentUser(c)
	if !ok {
		httputil.Unauthorized(c, "authentication required")
		return
	}

	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		httputil.ErrorJSON(c, http.StatusBadRequest, "invalid_id", "invalid reminder id")
		return
	}

	reminder, err := h.svc.GetByID(c.Request.Context(), uint(id), user.ID)
	if err != nil {
		if err == services.ErrReminderNotFound {
			httputil.ErrorJSON(c, http.StatusNotFound, "not_found", "reminder not found")
			return
		}
		httputil.ErrorJSON(c, http.StatusInternalServerError, "get_failed", "failed to get reminder")
		return
	}

	c.JSON(http.StatusOK, toReminderResponse(reminder))
}

func (h *ReminderHandler) Update(c *gin.Context) {
	user, ok := CurrentUser(c)
	if !ok {
		httputil.Unauthorized(c, "authentication required")
		return
	}

	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		httputil.ErrorJSON(c, http.StatusBadRequest, "invalid_id", "invalid reminder id")
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
			httputil.ErrorJSON(c, http.StatusNotFound, "not_found", "reminder not found")
			return
		}
		if err == services.ErrInvalidTime {
			httputil.ErrorJSON(c, http.StatusBadRequest, "invalid_time", err.Error())
			return
		}
		httputil.ErrorJSON(c, http.StatusInternalServerError, "update_failed", "failed to update reminder")
		return
	}

	c.JSON(http.StatusOK, toReminderResponse(reminder))
}

func (h *ReminderHandler) Delete(c *gin.Context) {
	user, ok := CurrentUser(c)
	if !ok {
		httputil.Unauthorized(c, "authentication required")
		return
	}

	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		httputil.ErrorJSON(c, http.StatusBadRequest, "invalid_id", "invalid reminder id")
		return
	}

	if err := h.svc.Delete(c.Request.Context(), uint(id), user.ID); err != nil {
		httputil.ErrorJSON(c, http.StatusInternalServerError, "delete_failed", "failed to delete reminder")
		return
	}

	c.Status(http.StatusNoContent)
}

func toReminderResponse(r *models.Reminder) reminderResponse {
	return reminderResponse{
		ID:         r.ID,
		MedicineID: r.MedicineID,
		Time:       r.Time,
		Enabled:    r.Enabled,
		CreatedAt:  r.CreatedAt.Format(http.TimeFormat),
		UpdatedAt:  r.UpdatedAt.Format(http.TimeFormat),
	}
}
