package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"

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
	ConversationID uint   `json:"conversation_id"`
	Message        string `json:"message"`
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
	if req.Message == "" {
		httputil.ErrorJSON(c, http.StatusBadRequest, "bad_request", "message is required")
		return
	}

	// Set SSE headers
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")

	// If new conversation, set the ID header before streaming
	if req.ConversationID == 0 {
		c.Header("X-Conversation-ID", "creating")
	}

	flusher, ok := c.Writer.(http.Flusher)
	if !ok {
		httputil.ErrorJSON(c, http.StatusInternalServerError, "internal_error", "streaming unsupported")
		return
	}
	c.Status(http.StatusOK)
	flusher.Flush()

	result, err := h.svc.Send(c.Request.Context(), services.SendInput{
		UserID:         user.ID,
		ConversationID: req.ConversationID,
		Message:        req.Message,
	}, func(token string, err error) bool {
		if err != nil {
			writeSSEError(c.Writer, flusher, err.Error())
			return false
		}
		writeSSEToken(c.Writer, flusher, token)
		return true
	})
	if err != nil {
		writeSSEError(c.Writer, flusher, err.Error())
		return
	}

	// Send final event with conversation_id
	writeSSEDone(c.Writer, flusher, result.ConversationID)
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

func writeSSEDone(w http.ResponseWriter, flusher http.Flusher, convID uint) {
	data, _ := json.Marshal(map[string]any{
		"conversation_id": convID,
		"partial":         false,
		"done":            true,
	})
	fmt.Fprintf(w, "data: %s\n\n", data)
	flusher.Flush()
}

func writeSSEError(w http.ResponseWriter, flusher http.Flusher, msg string) {
	data, _ := json.Marshal(map[string]string{"error": msg})
	fmt.Fprintf(w, "event: error\ndata: %s\n\n", data)
	flusher.Flush()
}
