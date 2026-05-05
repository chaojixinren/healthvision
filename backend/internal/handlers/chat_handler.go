package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"healthvision/backend/internal/httputil"
	"healthvision/backend/internal/services"

	"github.com/gin-gonic/gin"
)

// ChatHandler handles chat-related endpoints.
type ChatHandler struct {
	svc *services.ChatService
}

// NewChatHandler creates a ChatHandler.
func NewChatHandler(svc *services.ChatService) *ChatHandler {
	return &ChatHandler{svc: svc}
}

type sendRequest struct {
	ConversationID uint `json:"conversation_id"`
	Message        string `json:"message"`
	Images         []string `json:"images,omitempty"`
	ToolConfirmation *struct {
		ConfirmationCallID string `json:"confirmation_call_id"`
		Confirmed          bool   `json:"confirmed"`
		Payload            any    `json:"payload,omitempty"`
	} `json:"tool_confirmation,omitempty"`
}

// Send processes a chat message and streams the response via SSE.
func (h *ChatHandler) Send(c *gin.Context) {
	user, ok := CurrentUser(c)
	if !ok {
		httputil.Unauthorized(c, "authentication required")
		return
	}

	var req sendRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httputil.ErrorJSON(c, http.StatusBadRequest, "bad_request", "invalid request")
		return
	}

	hasConfirmation := req.ToolConfirmation != nil
	if hasConfirmation {
		if req.ConversationID == 0 {
			httputil.ErrorJSON(c, http.StatusBadRequest, "bad_request", "conversation_id is required when sending tool_confirmation")
			return
		}
		if strings.TrimSpace(req.ToolConfirmation.ConfirmationCallID) == "" {
			httputil.ErrorJSON(c, http.StatusBadRequest, "bad_request", "tool_confirmation.confirmation_call_id is required")
			return
		}
	} else if strings.TrimSpace(req.Message) == "" {
		httputil.ErrorJSON(c, http.StatusBadRequest, "bad_request", "message is required")
		return
	}

	input := services.SendInput{
		UserID:         user.ID,
		ConversationID: req.ConversationID,
		Message:        req.Message,
		Images:         req.Images,
	}
	if hasConfirmation {
		input.ToolConfirmation = &services.ToolConfirmationInput{
			CallID:    req.ToolConfirmation.ConfirmationCallID,
			Confirmed: req.ToolConfirmation.Confirmed,
			Payload:   req.ToolConfirmation.Payload,
		}
	}

	// Set SSE headers
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")

	if req.ConversationID == 0 && !hasConfirmation {
		c.Header("X-Conversation-ID", "creating")
	}

	flusher, ok := c.Writer.(http.Flusher)
	if !ok {
		httputil.ErrorJSON(c, http.StatusInternalServerError, "internal_error", "streaming unsupported")
		return
	}
	c.Status(http.StatusOK)
	flusher.Flush()

	result, err := h.svc.Send(c.Request.Context(), input, func(ev services.StreamEvent) bool {
		if ev.Err != nil {
			writeSSEError(c.Writer, flusher, ev.Err.Error())
			return false
		}
		if ev.ToolConfirmation != nil {
			writeSSEToolConfirmation(c.Writer, flusher, ev.ToolConfirmation)
			return true
		}
		if ev.Token != "" {
			writeSSEToken(c.Writer, flusher, ev.Token)
		}
		return true
	})
	if err != nil {
		writeSSEError(c.Writer, flusher, err.Error())
		return
	}

	writeSSEDone(c.Writer, flusher, result.ConversationID, result.PendingConfirmation)
}

// ListConversations returns the user's conversations.
func (h *ChatHandler) ListConversations(c *gin.Context) {
	user, ok := CurrentUser(c)
	if !ok {
		httputil.Unauthorized(c, "authentication required")
		return
	}

	convs, err := h.svc.ListConversations(c.Request.Context(), user.ID)
	if err != nil {
		httputil.ErrorJSON(c, http.StatusInternalServerError, "internal_error", err.Error())
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": convs})
}

// GetMessages returns messages for a conversation.
func (h *ChatHandler) GetMessages(c *gin.Context) {
	user, ok := CurrentUser(c)
	if !ok {
		httputil.Unauthorized(c, "authentication required")
		return
	}

	var req struct {
		ConversationID uint `json:"conversation_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		httputil.ErrorJSON(c, http.StatusBadRequest, "bad_request", "invalid request")
		return
	}

	msgs, err := h.svc.GetMessages(c.Request.Context(), req.ConversationID, user.ID)
	if err != nil {
		httputil.ErrorJSON(c, http.StatusInternalServerError, "internal_error", err.Error())
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": msgs})
}

// DeleteConversation deletes a conversation and its messages.
func (h *ChatHandler) DeleteConversation(c *gin.Context) {
	user, ok := CurrentUser(c)
	if !ok {
		httputil.Unauthorized(c, "authentication required")
		return
	}

	var req struct {
		ConversationID uint `json:"conversation_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		httputil.ErrorJSON(c, http.StatusBadRequest, "bad_request", "invalid request")
		return
	}

	if err := h.svc.DeleteConversation(c.Request.Context(), req.ConversationID, user.ID); err != nil {
		httputil.ErrorJSON(c, http.StatusInternalServerError, "internal_error", err.Error())
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "deleted"})
}

// ---- SSE helpers ----

func writeSSEToken(w http.ResponseWriter, flusher http.Flusher, token string) {
	data, _ := json.Marshal(map[string]any{
		"token":   token,
		"partial": true,
	})
	fmt.Fprintf(w, "data: %s\n\n", data)
	flusher.Flush()
}

func writeSSEToolConfirmation(w http.ResponseWriter, flusher http.Flusher, p *services.ToolConfirmationPayload) {
	data, _ := json.Marshal(map[string]any{
		"partial": false,
		"tool_confirmation": map[string]any{
			"confirmation_call_id":   p.ConfirmationCallID,
			"hint":                   p.Hint,
			"original_function_call": p.OriginalFunctionCall,
		},
	})
	fmt.Fprintf(w, "data: %s\n\n", data)
	flusher.Flush()
}

func writeSSEDone(w http.ResponseWriter, flusher http.Flusher, convID uint, pendingConfirmation bool) {
	data, _ := json.Marshal(map[string]any{
		"conversation_id":       convID,
		"partial":             false,
		"done":                true,
		"pending_confirmation": pendingConfirmation,
	})
	fmt.Fprintf(w, "data: %s\n\n", data)
	flusher.Flush()
}

func writeSSEError(w http.ResponseWriter, flusher http.Flusher, msg string) {
	data, _ := json.Marshal(map[string]string{"error": msg})
	fmt.Fprintf(w, "event: error\ndata: %s\n\n", data)
	flusher.Flush()
}
