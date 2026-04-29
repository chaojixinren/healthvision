package handlers

import (
	"net/http"
	"strconv"

	"healthvision/backend/internal/httputil"
	"healthvision/backend/internal/models"
	"healthvision/backend/internal/services"

	"github.com/gin-gonic/gin"
)

type MedicineHandler struct {
	svc *services.MedicineService
}

func NewMedicineHandler(svc *services.MedicineService) *MedicineHandler {
	return &MedicineHandler{svc: svc}
}

type medicineRequest struct {
	Name        string `json:"name" binding:"required,max=100"`
	ImageURL    string `json:"image_url" binding:"max=500"`
	Description string `json:"description"`
	Notes       string `json:"notes"`
}

type medicineResponse struct {
	ID          uint   `json:"id"`
	Name        string `json:"name"`
	ImageURL    string `json:"image_url"`
	Description string `json:"description"`
	Notes       string `json:"notes"`
	CreatedAt   string `json:"created_at"`
	UpdatedAt   string `json:"updated_at"`
}

type listMedicinesResponse struct {
	Data       []medicineResponse `json:"data"`
	Pagination paginationInfo       `json:"pagination"`
}

type paginationInfo struct {
	Page     int   `json:"page"`
	PerPage  int   `json:"per_page"`
	Total    int64 `json:"total"`
}

func (h *MedicineHandler) Create(c *gin.Context) {
	var req medicineRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httputil.ErrorJSON(c, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}

	user, ok := CurrentUser(c)
	if !ok {
		httputil.Unauthorized(c, "authentication required")
		return
	}

	medicine, err := h.svc.Create(c.Request.Context(), user.ID, req.Name, req.ImageURL, req.Description, req.Notes)
	if err != nil {
		httputil.ErrorJSON(c, http.StatusInternalServerError, "create_failed", "failed to create medicine")
		return
	}

	c.JSON(http.StatusCreated, toMedicineResponse(medicine))
}

func (h *MedicineHandler) List(c *gin.Context) {
	user, ok := CurrentUser(c)
	if !ok {
		httputil.Unauthorized(c, "authentication required")
		return
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	perPage, _ := strconv.Atoi(c.DefaultQuery("per_page", "20"))

	medicines, total, err := h.svc.List(c.Request.Context(), user.ID, page, perPage)
	if err != nil {
		httputil.ErrorJSON(c, http.StatusInternalServerError, "list_failed", "failed to list medicines")
		return
	}

	resp := listMedicinesResponse{
		Data: make([]medicineResponse, len(medicines)),
		Pagination: paginationInfo{
			Page:    page,
			PerPage: perPage,
			Total:   total,
		},
	}
	for i, m := range medicines {
		resp.Data[i] = toMedicineResponse(&m)
	}

	c.JSON(http.StatusOK, resp)
}

func (h *MedicineHandler) Get(c *gin.Context) {
	user, ok := CurrentUser(c)
	if !ok {
		httputil.Unauthorized(c, "authentication required")
		return
	}

	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		httputil.ErrorJSON(c, http.StatusBadRequest, "invalid_id", "invalid medicine id")
		return
	}

	medicine, err := h.svc.GetByID(c.Request.Context(), uint(id), user.ID)
	if err != nil {
		if err == services.ErrMedicineNotFound {
			httputil.ErrorJSON(c, http.StatusNotFound, "not_found", "medicine not found")
			return
		}
		httputil.ErrorJSON(c, http.StatusInternalServerError, "get_failed", "failed to get medicine")
		return
	}

	c.JSON(http.StatusOK, toMedicineResponse(medicine))
}

func (h *MedicineHandler) Update(c *gin.Context) {
	user, ok := CurrentUser(c)
	if !ok {
		httputil.Unauthorized(c, "authentication required")
		return
	}

	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		httputil.ErrorJSON(c, http.StatusBadRequest, "invalid_id", "invalid medicine id")
		return
	}

	var req medicineRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httputil.ErrorJSON(c, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}

	medicine, err := h.svc.Update(c.Request.Context(), uint(id), user.ID, req.Name, req.ImageURL, req.Description, req.Notes)
	if err != nil {
		if err == services.ErrMedicineNotFound {
			httputil.ErrorJSON(c, http.StatusNotFound, "not_found", "medicine not found")
			return
		}
		httputil.ErrorJSON(c, http.StatusInternalServerError, "update_failed", "failed to update medicine")
		return
	}

	c.JSON(http.StatusOK, toMedicineResponse(medicine))
}

func (h *MedicineHandler) Delete(c *gin.Context) {
	user, ok := CurrentUser(c)
	if !ok {
		httputil.Unauthorized(c, "authentication required")
		return
	}

	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		httputil.ErrorJSON(c, http.StatusBadRequest, "invalid_id", "invalid medicine id")
		return
	}

	if err := h.svc.Delete(c.Request.Context(), uint(id), user.ID); err != nil {
		httputil.ErrorJSON(c, http.StatusInternalServerError, "delete_failed", "failed to delete medicine")
		return
	}

	c.Status(http.StatusNoContent)
}

func toMedicineResponse(m *models.Medicine) medicineResponse {
	return medicineResponse{
		ID:          m.ID,
		Name:        m.Name,
		ImageURL:    m.ImageURL,
		Description: m.Description,
		Notes:       m.Notes,
		CreatedAt:   m.CreatedAt.Format(http.TimeFormat),
		UpdatedAt:   m.UpdatedAt.Format(http.TimeFormat),
	}
}
