package handlers

import (
	"errors"
	"net/http"
	"time"

	"healthvision/backend/internal/httputil"
	"healthvision/backend/internal/models"
	"healthvision/backend/internal/services"

	"github.com/gin-gonic/gin"
)

type LocationHandler struct {
	svc *services.LocationService
}

func NewLocationHandler(svc *services.LocationService) *LocationHandler {
	return &LocationHandler{svc: svc}
}

// --- request / response types ---

type reportLocationRequest struct {
	Latitude  float64 `json:"latitude" binding:"required"`
	Longitude float64 `json:"longitude" binding:"required"`
	Altitude  float64 `json:"altitude"`
	Timestamp string  `json:"timestamp"`
}

type locationResponse struct {
	ID        uint    `json:"id"`
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
	Altitude  float64 `json:"altitude"`
	Timestamp string  `json:"timestamp"`
	CreatedAt string  `json:"created_at"`
	UpdatedAt string  `json:"updated_at"`
}

// --- handlers ---

func (h *LocationHandler) Report(c *gin.Context) {
	user, ok := CurrentUser(c)
	if !ok {
		httputil.Unauthorized(c, "请先登录")
		return
	}

	var req reportLocationRequest
	if !bindJSON(c, &req) {
		return
	}

	// Parse timestamp: accept RFC3339 or "2006-01-02 15:04:05" from ESP32.
	var ts time.Time
	if req.Timestamp != "" {
		var err error
		ts, err = time.Parse(time.RFC3339, req.Timestamp)
		if err != nil {
			ts, err = time.Parse("2006-01-02 15:04:05", req.Timestamp)
			if err != nil {
				httputil.ErrorJSON(c, http.StatusBadRequest, "invalid_timestamp", "时间格式无效，请使用 RFC3339 或 YYYY-MM-DD HH:MM:SS")
				return
			}
		}
	} else {
		ts = time.Now()
	}

	loc, err := h.svc.Report(c.Request.Context(), user.ID, req.Latitude, req.Longitude, req.Altitude, ts)
	if errors.Is(err, services.ErrInvalidCoordinates) {
		httputil.ErrorJSON(c, http.StatusBadRequest, "invalid_coordinates", "经纬度无效")
		return
	}
	if err != nil {
		httputil.ErrorJSON(c, http.StatusInternalServerError, "report_failed", "上报位置失败")
		return
	}

	c.JSON(http.StatusOK, toLocationResponse(loc))
}

func (h *LocationHandler) GetLatest(c *gin.Context) {
	user, ok := CurrentUser(c)
	if !ok {
		httputil.Unauthorized(c, "请先登录")
		return
	}

	loc, err := h.svc.GetLatest(c.Request.Context(), user.ID)
	if errors.Is(err, services.ErrLocationNotFound) {
		httputil.ErrorJSON(c, http.StatusNotFound, "not_found", "暂无位置记录")
		return
	}
	if err != nil {
		httputil.ErrorJSON(c, http.StatusInternalServerError, "get_failed", "获取位置失败")
		return
	}

	c.JSON(http.StatusOK, toLocationResponse(loc))
}

// --- helpers ---

func toLocationResponse(loc *models.Location) locationResponse {
	return locationResponse{
		ID:        loc.ID,
		Latitude:  loc.Latitude,
		Longitude: loc.Longitude,
		Altitude:  loc.Altitude,
		Timestamp: loc.Timestamp.Format(time.RFC3339),
		CreatedAt: loc.CreatedAt.Format(time.RFC3339),
		UpdatedAt: loc.UpdatedAt.Format(time.RFC3339),
	}
}
