package handlers

import (
	"encoding/base64"
	"errors"
	"fmt"
	"net/http"
	"net/mail"
	"net/url"
	"reflect"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"healthvision/backend/internal/httputil"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
)

const (
	maxPageSize          = 100
	defaultPageSize      = 20
	maxEmailLength       = 254
	maxPasswordLength    = 128
	minRegisterPassword  = 8
	maxChatMessageLength = 4000
	maxChatImages        = 3
	maxChatImageBytes    = 2 * 1024 * 1024
)

var fieldLabels = map[string]string{
	"accept":          "处理结果",
	"conversation_id": "会话 ID",
	"description":     "药品说明",
	"email":           "邮箱",
	"image_url":       "图片地址",
	"images":          "图片",
	"interval_days":   "间隔天数",
	"medicine_id":     "药品 ID",
	"message":         "消息",
	"name":            "用户名",
	"notes":           "服用备注",
	"password":        "密码",
	"repeat_type":     "重复规则",
	"target_user_id":  "目标用户 ID",
	"time":            "时间",
	"to_email":        "对方邮箱",
	"weekdays":        "星期配置",
}

func bindJSON(c *gin.Context, dst any) bool {
	if err := c.ShouldBindJSON(dst); err != nil {
		httputil.ErrorJSON(c, http.StatusBadRequest, "invalid_request", validationErrorMessage(err, dst))
		return false
	}
	return true
}

func validationErrorMessage(err error, dst any) string {
	var validationErrors validator.ValidationErrors
	if !errors.As(err, &validationErrors) || len(validationErrors) == 0 {
		return "请求参数无效"
	}

	messages := make([]string, 0, len(validationErrors))
	for _, fieldErr := range validationErrors {
		label := validationFieldLabel(dst, fieldErr.StructField())
		messages = append(messages, fmt.Sprintf("%s%s", label, validationTagMessage(fieldErr)))
		if len(messages) == 3 {
			break
		}
	}
	return strings.Join(messages, "；")
}

func validationFieldLabel(dst any, structField string) string {
	jsonName := jsonFieldName(dst, structField)
	if label, ok := fieldLabels[jsonName]; ok {
		return label
	}
	if jsonName != "" {
		return jsonName
	}
	return structField
}

func jsonFieldName(dst any, structField string) string {
	t := reflect.TypeOf(dst)
	for t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	if t.Kind() != reflect.Struct {
		return ""
	}
	field, ok := t.FieldByName(structField)
	if !ok {
		return ""
	}
	tag := field.Tag.Get("json")
	if tag == "" || tag == "-" {
		return field.Name
	}
	name := strings.Split(tag, ",")[0]
	if name == "" {
		return field.Name
	}
	return name
}

func validationTagMessage(err validator.FieldError) string {
	switch err.Tag() {
	case "required":
		return "不能为空"
	case "email":
		return "格式无效"
	case "min":
		return "长度不足"
	case "max":
		return "长度过长"
	case "gt":
		return "必须大于 0"
	case "oneof":
		return "取值无效"
	default:
		return "无效"
	}
}

func requireString(c *gin.Context, value *string, label string, maxRunes int) bool {
	*value = strings.TrimSpace(*value)
	if *value == "" {
		httputil.ErrorJSON(c, http.StatusBadRequest, "invalid_request", label+"不能为空")
		return false
	}
	return optionalString(c, value, label, maxRunes)
}

func optionalString(c *gin.Context, value *string, label string, maxRunes int) bool {
	*value = strings.TrimSpace(*value)
	if maxRunes > 0 && utf8.RuneCountInString(*value) > maxRunes {
		httputil.ErrorJSON(c, http.StatusBadRequest, "invalid_request", fmt.Sprintf("%s不能超过%d个字符", label, maxRunes))
		return false
	}
	return true
}

func requireEmail(c *gin.Context, value *string, label string) bool {
	*value = strings.ToLower(strings.TrimSpace(*value))
	if *value == "" {
		httputil.ErrorJSON(c, http.StatusBadRequest, "invalid_request", label+"不能为空")
		return false
	}
	if len(*value) > maxEmailLength || strings.ContainsAny(*value, "\r\n") {
		httputil.ErrorJSON(c, http.StatusBadRequest, "invalid_request", label+"格式无效")
		return false
	}
	addr, err := mail.ParseAddress(*value)
	if err != nil || addr.Address != *value {
		httputil.ErrorJSON(c, http.StatusBadRequest, "invalid_request", label+"格式无效")
		return false
	}
	return true
}

func requirePassword(c *gin.Context, value string, minRunes int) bool {
	length := utf8.RuneCountInString(value)
	if length < minRunes {
		httputil.ErrorJSON(c, http.StatusBadRequest, "invalid_request", fmt.Sprintf("密码至少%d位", minRunes))
		return false
	}
	if length > maxPasswordLength {
		httputil.ErrorJSON(c, http.StatusBadRequest, "invalid_request", fmt.Sprintf("密码不能超过%d位", maxPasswordLength))
		return false
	}
	return true
}

func optionalHTTPURL(c *gin.Context, value *string, label string, maxRunes int) bool {
	*value = strings.TrimSpace(*value)
	if *value == "" {
		return true
	}
	if !optionalString(c, value, label, maxRunes) {
		return false
	}
	parsed, err := url.ParseRequestURI(*value)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		httputil.ErrorJSON(c, http.StatusBadRequest, "invalid_request", label+"必须是有效的 http 或 https 地址")
		return false
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		httputil.ErrorJSON(c, http.StatusBadRequest, "invalid_request", label+"必须是有效的 http 或 https 地址")
		return false
	}
	return true
}

func parsePositiveUintParam(c *gin.Context, name string, label string) (uint, bool) {
	raw := strings.TrimSpace(c.Param(name))
	id, err := strconv.ParseUint(raw, 10, strconv.IntSize)
	if err != nil || id == 0 {
		httputil.ErrorJSON(c, http.StatusBadRequest, "invalid_id", "无效的"+label)
		return 0, false
	}
	return uint(id), true
}

func parseOptionalPositiveUintQuery(c *gin.Context, key string, label string) (*uint, bool) {
	raw := strings.TrimSpace(c.Query(key))
	if raw == "" {
		return nil, true
	}
	id, err := strconv.ParseUint(raw, 10, strconv.IntSize)
	if err != nil || id == 0 {
		httputil.ErrorJSON(c, http.StatusBadRequest, "invalid_query", label+"必须是正整数")
		return nil, false
	}
	v := uint(id)
	return &v, true
}

func parsePagination(c *gin.Context) (int, int, bool) {
	page, ok := parsePositiveIntQuery(c, "page", "页码", 1)
	if !ok {
		return 0, 0, false
	}
	perPage, ok := parsePositiveIntQuery(c, "per_page", "每页数量", defaultPageSize)
	if !ok {
		return 0, 0, false
	}
	if perPage > maxPageSize {
		httputil.ErrorJSON(c, http.StatusBadRequest, "invalid_query", fmt.Sprintf("每页数量不能超过%d", maxPageSize))
		return 0, 0, false
	}
	return page, perPage, true
}

func parsePositiveIntQuery(c *gin.Context, key string, label string, fallback int) (int, bool) {
	raw := strings.TrimSpace(c.Query(key))
	if raw == "" {
		return fallback, true
	}
	value, err := strconv.Atoi(raw)
	if err != nil || value < 1 {
		httputil.ErrorJSON(c, http.StatusBadRequest, "invalid_query", label+"必须是正整数")
		return 0, false
	}
	return value, true
}

func parseDateQuery(c *gin.Context, key string, fallback time.Time) (string, bool) {
	raw := strings.TrimSpace(c.DefaultQuery(key, fallback.Format("2006-01-02")))
	parsed, err := time.Parse("2006-01-02", raw)
	if err != nil || parsed.Format("2006-01-02") != raw {
		httputil.ErrorJSON(c, http.StatusBadRequest, "invalid_query", "日期格式必须为 YYYY-MM-DD")
		return "", false
	}
	return raw, true
}

func validateChatMessage(c *gin.Context, message string) bool {
	if utf8.RuneCountInString(message) > maxChatMessageLength {
		httputil.ErrorJSON(c, http.StatusBadRequest, "bad_request", fmt.Sprintf("消息不能超过%d个字符", maxChatMessageLength))
		return false
	}
	return true
}

func validateChatImages(c *gin.Context, images []string) bool {
	if len(images) > maxChatImages {
		httputil.ErrorJSON(c, http.StatusBadRequest, "bad_request", fmt.Sprintf("一次最多上传%d张图片", maxChatImages))
		return false
	}
	for i, image := range images {
		if !validateChatImage(c, image, i+1) {
			return false
		}
	}
	return true
}

func validateChatImage(c *gin.Context, image string, index int) bool {
	mime, payload, ok := strings.Cut(strings.TrimSpace(image), ";base64,")
	if !ok || !strings.HasPrefix(mime, "data:image/") {
		httputil.ErrorJSON(c, http.StatusBadRequest, "bad_request", fmt.Sprintf("第%d张图片格式无效", index))
		return false
	}
	switch strings.TrimPrefix(mime, "data:") {
	case "image/jpeg", "image/png", "image/webp", "image/gif":
	default:
		httputil.ErrorJSON(c, http.StatusBadRequest, "bad_request", fmt.Sprintf("第%d张图片类型不支持", index))
		return false
	}
	if len(payload) > base64.StdEncoding.EncodedLen(maxChatImageBytes) {
		httputil.ErrorJSON(c, http.StatusBadRequest, "bad_request", fmt.Sprintf("第%d张图片不能超过2MB", index))
		return false
	}
	raw, err := base64.StdEncoding.DecodeString(payload)
	if err != nil {
		httputil.ErrorJSON(c, http.StatusBadRequest, "bad_request", fmt.Sprintf("第%d张图片内容无效", index))
		return false
	}
	if len(raw) == 0 || len(raw) > maxChatImageBytes {
		httputil.ErrorJSON(c, http.StatusBadRequest, "bad_request", fmt.Sprintf("第%d张图片不能超过2MB", index))
		return false
	}
	return true
}
