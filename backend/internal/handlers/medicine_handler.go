package handlers

import (
	"errors"
	"net/http"

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
	Name        string `json:"name" binding:"required"`
	ImageURL    string `json:"image_url"`
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
	Pagination paginationInfo     `json:"pagination"`
}

type paginationInfo struct {
	Page    int   `json:"page"`
	PerPage int   `json:"per_page"`
	Total   int64 `json:"total"`
}

func (h *MedicineHandler) Create(c *gin.Context) {
	var req medicineRequest
	if !bindJSON(c, &req) {
		return
	}
	if !validateMedicineRequest(c, &req) {
		return
	}

	user, ok := CurrentUser(c)
	if !ok {
		httputil.Unauthorized(c, "请先登录")
		return
	}

	medicine, err := h.svc.Create(c.Request.Context(), user.ID, req.Name, req.ImageURL, req.Description, req.Notes)
	if err != nil {
		if isInvalidMedicineInput(err) {
			httputil.ErrorJSON(c, http.StatusBadRequest, "invalid_request", err.Error())
			return
		}
		httputil.ErrorJSON(c, http.StatusInternalServerError, "create_failed", "创建药品失败")
		return
	}

	c.JSON(http.StatusCreated, toMedicineResponse(medicine))
}

func (h *MedicineHandler) List(c *gin.Context) {
	user, ok := CurrentUser(c)
	if !ok {
		httputil.Unauthorized(c, "请先登录")
		return
	}

	page, perPage, ok := parsePagination(c)
	if !ok {
		return
	}

	medicines, total, err := h.svc.List(c.Request.Context(), user.ID, page, perPage)
	if err != nil {
		httputil.ErrorJSON(c, http.StatusInternalServerError, "list_failed", "获取药品列表失败")
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
		httputil.Unauthorized(c, "请先登录")
		return
	}

	id, ok := parsePositiveUintParam(c, "id", "药品 ID")
	if !ok {
		return
	}

	medicine, err := h.svc.GetByID(c.Request.Context(), id, user.ID)
	if err != nil {
		if err == services.ErrMedicineNotFound {
			httputil.ErrorJSON(c, http.StatusNotFound, "not_found", "药品不存在")
			return
		}
		httputil.ErrorJSON(c, http.StatusInternalServerError, "get_failed", "获取药品失败")
		return
	}

	c.JSON(http.StatusOK, toMedicineResponse(medicine))
}

func (h *MedicineHandler) Update(c *gin.Context) {
	user, ok := CurrentUser(c)
	if !ok {
		httputil.Unauthorized(c, "请先登录")
		return
	}

	id, ok := parsePositiveUintParam(c, "id", "药品 ID")
	if !ok {
		return
	}

	var req medicineRequest
	if !bindJSON(c, &req) {
		return
	}
	if !validateMedicineRequest(c, &req) {
		return
	}

	medicine, err := h.svc.Update(c.Request.Context(), id, user.ID, req.Name, req.ImageURL, req.Description, req.Notes)
	if err != nil {
		if isInvalidMedicineInput(err) {
			httputil.ErrorJSON(c, http.StatusBadRequest, "invalid_request", err.Error())
			return
		}
		if err == services.ErrMedicineNotFound {
			httputil.ErrorJSON(c, http.StatusNotFound, "not_found", "药品不存在")
			return
		}
		httputil.ErrorJSON(c, http.StatusInternalServerError, "update_failed", "更新药品失败")
		return
	}

	c.JSON(http.StatusOK, toMedicineResponse(medicine))
}

func (h *MedicineHandler) Delete(c *gin.Context) {
	user, ok := CurrentUser(c)
	if !ok {
		httputil.Unauthorized(c, "请先登录")
		return
	}

	id, ok := parsePositiveUintParam(c, "id", "药品 ID")
	if !ok {
		return
	}

	if err := h.svc.Delete(c.Request.Context(), id, user.ID); err != nil {
		httputil.ErrorJSON(c, http.StatusInternalServerError, "delete_failed", "删除药品失败")
		return
	}

	c.Status(http.StatusNoContent)
}

func isInvalidMedicineInput(err error) bool {
	return errors.Is(err, services.ErrInvalidMedicineName) ||
		errors.Is(err, services.ErrInvalidMedicineImageURL) ||
		errors.Is(err, services.ErrInvalidMedicineText)
}

func validateMedicineRequest(c *gin.Context, req *medicineRequest) bool {
	return requireString(c, &req.Name, "药品名称", 100) &&
		optionalHTTPURL(c, &req.ImageURL, "图片地址", 500) &&
		optionalString(c, &req.Description, "药品说明", 2000) &&
		optionalString(c, &req.Notes, "服用备注", 1000)
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
