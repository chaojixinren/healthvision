package agent

import (
	"encoding/json"
	"net/http"
	"strings"

	"google.golang.org/adk/agent"
	"google.golang.org/adk/runner"
	"google.golang.org/adk/session"
	"google.golang.org/genai"

	"github.com/gin-gonic/gin"
)

type sseRequest struct {
	AppName    string        `json:"appName"`
	UserId     string        `json:"userId"`
	SessionId  string        `json:"sessionId"`
	NewMessage genai.Content `json:"newMessage"`
}

type sseEvent struct {
	ID           string      `json:"id,omitempty"`
	Partial      bool        `json:"partial,omitempty"`
	TurnComplete bool        `json:"turnComplete,omitempty"`
	FinishReason string      `json:"finishReason,omitempty"`
	Content      *sseContent `json:"content,omitempty"`
}

type sseContent struct {
	Role  string    `json:"role"`
	Parts []ssePart `json:"parts"`
}

type ssePart struct {
	Text string `json:"text"`
}

// SSEHandler returns a Gin handler that streams agent responses via SSE.
func SSEHandler(r *runner.Runner) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req sseRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request: " + err.Error()})
			return
		}

		c.Status(http.StatusOK)
		c.Header("Content-Type", "text/event-stream")
		c.Header("Cache-Control", "no-cache")
		c.Header("Connection", "keep-alive")

		flusher, ok := c.Writer.(http.Flusher)
		if !ok {
			return
		}
		flusher.Flush()

		events := r.Run(
			c.Request.Context(),
			req.UserId,
			req.SessionId,
			&genai.Content{
				Role:  "user",
				Parts: req.NewMessage.Parts,
			},
			agent.RunConfig{StreamingMode: agent.StreamingModeSSE},
		)

		for event, err := range events {
			if err != nil {
				if strings.Contains(err.Error(), "last event is not final") {
					return
				}
				writeSSEError(c.Writer, flusher, err.Error())
				return
			}
			if event == nil {
				continue
			}
			data, err := json.Marshal(toSSEEvent(event))
			if err != nil {
				continue
			}
			if _, err := c.Writer.Write([]byte("data: " + string(data) + "\n\n")); err != nil {
				return
			}
			flusher.Flush()
		}
	}
}

func toSSEEvent(e *session.Event) sseEvent {
	evt := sseEvent{
		ID:           e.ID,
		Partial:      e.LLMResponse.Partial,
		TurnComplete: e.LLMResponse.TurnComplete,
		FinishReason: string(e.LLMResponse.FinishReason),
	}
	if e.LLMResponse.Content != nil {
		var parts []ssePart
		for _, p := range e.LLMResponse.Content.Parts {
			if p.Text != "" {
				parts = append(parts, ssePart{Text: p.Text})
			}
		}
		evt.Content = &sseContent{
			Role:  e.LLMResponse.Content.Role,
			Parts: parts,
		}
	}
	return evt
}

func writeSSEError(w http.ResponseWriter, flusher http.Flusher, msg string) {
	errData, _ := json.Marshal(map[string]string{"error": msg})
	w.Write([]byte("event: error\ndata: " + string(errData) + "\n\n"))
	flusher.Flush()
}
