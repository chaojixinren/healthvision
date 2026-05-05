package handlers

import (
	"context"
	"net/http"
	"strconv"
	"time"

	"healthvision/backend/internal/httputil"
	"healthvision/backend/internal/models"
	"healthvision/backend/internal/services"

	"github.com/gin-gonic/gin"
)

type ConfirmationMedicineLookup interface {
	FindByIDAny(ctx context.Context, id uint) (*models.Medicine, error)
}

type ConfirmationUserLookup interface {
	FindByID(ctx context.Context, id uint) (*models.User, error)
}

type ConfirmationHandler struct {
	svc       *services.ConfirmationService
	medLookup ConfirmationMedicineLookup
	userRepo  ConfirmationUserLookup
}

func NewConfirmationHandler(svc *services.ConfirmationService, medLookup ConfirmationMedicineLookup, userRepo ConfirmationUserLookup) *ConfirmationHandler {
	return &ConfirmationHandler{svc: svc, medLookup: medLookup, userRepo: userRepo}
}

type confirmationResponse struct {
	ID            uint    `json:"id"`
	ReminderID    uint    `json:"reminder_id"`
	UserID        uint    `json:"user_id"`
	MedicineID    uint    `json:"medicine_id"`
	MedicineName  string  `json:"medicine_name"`
	ScheduledDate string  `json:"scheduled_date"`
	ScheduledTime string  `json:"scheduled_time"`
	ConfirmedAt   *string `json:"confirmed_at"`
	ConfirmedBy   uint    `json:"confirmed_by"`
	ElderName     string  `json:"elder_name,omitempty"`
}

func (h *ConfirmationHandler) medicineName(ctx context.Context, medicineID uint) string {
	if h.medLookup == nil {
		return ""
	}
	m, err := h.medLookup.FindByIDAny(ctx, medicineID)
	if err != nil {
		return ""
	}
	return m.Name
}

func (h *ConfirmationHandler) List(c *gin.Context) {
	user, ok := CurrentUser(c)
	if !ok {
		httputil.Unauthorized(c, "authentication required")
		return
	}

	date := c.DefaultQuery("date", time.Now().Format("2006-01-02"))

	var confirmations []models.Confirmation
	var err error
	var elderNames = make(map[uint]string)

	if user.IsOld {
		confirmations, err = h.svc.ListByUser(c.Request.Context(), user.ID, date)
	} else {
		elderIDs, e := h.svc.ListBoundElderIDs(c.Request.Context(), user.ID)
		if e != nil {
			httputil.ErrorJSON(c, http.StatusInternalServerError, "list_failed", "failed to list bindings")
			return
		}
		if len(elderIDs) == 0 {
			c.JSON(http.StatusOK, gin.H{"data": []confirmationResponse{}})
			return
		}
		confirmations, err = h.svc.ListByElderIDs(c.Request.Context(), elderIDs, date)
		if h.userRepo != nil {
			for _, eid := range elderIDs {
				u, e2 := h.userRepo.FindByID(c.Request.Context(), eid)
				if e2 == nil && u != nil {
					elderNames[eid] = u.Name
				}
			}
		}
	}
	if err != nil {
		httputil.ErrorJSON(c, http.StatusInternalServerError, "list_failed", "failed to list confirmations")
		return
	}

	resp := make([]confirmationResponse, len(confirmations))
	for i, cf := range confirmations {
		r := confirmationResponse{
			ID:            cf.ID,
			ReminderID:    cf.ReminderID,
			UserID:        cf.UserID,
			MedicineID:    cf.MedicineID,
			MedicineName:  h.medicineName(c.Request.Context(), cf.MedicineID),
			ScheduledDate: cf.ScheduledDate,
			ScheduledTime: cf.ScheduledTime,
			ConfirmedBy:   cf.ConfirmedBy,
		}
		if cf.ConfirmedAt != nil {
			s := cf.ConfirmedAt.Format(time.RFC3339)
			r.ConfirmedAt = &s
		}
		if name, ok2 := elderNames[cf.UserID]; ok2 {
			r.ElderName = name
		}
		resp[i] = r
	}

	c.JSON(http.StatusOK, gin.H{"data": resp})
}

func (h *ConfirmationHandler) Confirm(c *gin.Context) {
	user, ok := CurrentUser(c)
	if !ok {
		httputil.Unauthorized(c, "authentication required")
		return
	}

	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		httputil.ErrorJSON(c, http.StatusBadRequest, "invalid_id", "invalid confirmation id")
		return
	}

	cf, err := h.svc.Confirm(c.Request.Context(), uint(id), user.ID, user.IsOld)
	if err != nil {
		switch err {
		case services.ErrAlreadyConfirmed:
			httputil.ErrorJSON(c, http.StatusConflict, "already_confirmed", "already confirmed")
		case services.ErrConfirmForbidden:
			httputil.ErrorJSON(c, http.StatusForbidden, "forbidden", "not allowed to confirm this dose")
		case services.ErrNotBoundToElder:
			httputil.ErrorJSON(c, http.StatusForbidden, "forbidden", "not bound to this elder")
		default:
			httputil.ErrorJSON(c, http.StatusInternalServerError, "confirm_failed", "failed to confirm")
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"id":           cf.ID,
		"confirmed_at": cf.ConfirmedAt.Format(time.RFC3339),
		"confirmed_by": cf.ConfirmedBy,
	})
}
