package services

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	hvagent "healthvision/backend/internal/agent"
	"healthvision/backend/internal/models"

	adkagent "google.golang.org/adk/agent"
	"google.golang.org/adk/model"
	"google.golang.org/adk/runner"
	"google.golang.org/adk/session"
	"google.golang.org/adk/tool/toolconfirmation"
	"google.golang.org/genai"
	"gorm.io/gorm"
)

const (
	maxHistory   = 20
	defaultTitle = "新对话"

	roleUser      = "user"
	roleAssistant = "assistant"

	appName = "healthvision"
)

// ChatService handles chat message processing on top of ADK's runner.
// It owns the conversation persistence (chat_messages / conversations) and
// drives the agent through ADK's tool-use loop.
type ChatService struct {
	db      *gorm.DB
	runner  *runner.Runner
	session session.Service
}

// NewChatService creates a ChatService backed by an ADK runner. The caller
// is expected to wire the agent's tools so that runner.Run can invoke them
// when the model requests it.
func NewChatService(db *gorm.DB, r *runner.Runner, s session.Service) *ChatService {
	return &ChatService{db: db, runner: r, session: s}
}

// ToolConfirmationInput is sent by the client to approve or reject a pending ADK write tool.
type ToolConfirmationInput struct {
	CallID    string
	Confirmed bool
	Payload   any
}

// SendInput contains the data needed to process a chat message.
type SendInput struct {
	UserID           uint
	ConversationID   uint
	Message          string
	Images           []string
	ToolConfirmation *ToolConfirmationInput
}

// SendResult is returned after message processing completes.
type SendResult struct {
	ConversationID      uint
	Title               string
	UserMessage         *models.ChatMessage
	Reply               string
	PendingConfirmation bool
}

// ToolConfirmationPayload is emitted over SSE when ADK requests human approval for a tool call.
type ToolConfirmationPayload struct {
	ConfirmationCallID   string
	Hint                 string
	OriginalFunctionCall map[string]any
}

// StreamEvent carries streamed assistant tokens, tool confirmation prompts, or errors.
type StreamEvent struct {
	Token            string
	ToolConfirmation *ToolConfirmationPayload
	Err              error
}

// StreamCallback is called for streaming tokens and tool confirmation prompts.
type StreamCallback func(ev StreamEvent) bool

// Send processes a user message (or a tool confirmation response) and streams via callback.
func (s *ChatService) Send(ctx context.Context, input SendInput, cb StreamCallback) (*SendResult, error) {
	userID := input.UserID
	convID := input.ConversationID
	isConfirmation := input.ToolConfirmation != nil

	if isConfirmation && convID == 0 {
		return nil, fmt.Errorf("conversation_id is required for tool confirmation")
	}

	if convID == 0 {
		conv := &models.Conversation{UserID: userID, Title: defaultTitle}
		if err := s.db.WithContext(ctx).Create(conv).Error; err != nil {
			return nil, fmt.Errorf("create conversation: %w", err)
		}
		convID = conv.ID
	}

	var userMsg *models.ChatMessage
	if !isConfirmation {
		if strings.TrimSpace(input.Message) == "" && len(input.Images) == 0 {
			return nil, fmt.Errorf("message or images required")
		}

		imagesJSON := marshalImages(input.Images)
		userMsg = &models.ChatMessage{
			UserID:         userID,
			ConversationID: convID,
			Role:           roleUser,
			Content:        input.Message,
			Images:         imagesJSON,
		}
		if err := s.db.WithContext(ctx).Create(userMsg).Error; err != nil {
			return nil, fmt.Errorf("save user message: %w", err)
		}
	}

	if isConfirmation && strings.TrimSpace(input.ToolConfirmation.CallID) == "" {
		return nil, fmt.Errorf("tool_confirmation.confirmation_call_id is required")
	}

	excludeLastUser := !isConfirmation
	history, err := s.getHistory(ctx, convID, userID, excludeLastUser)
	if err != nil {
		return nil, err
	}

	reply, pendingConf, err := s.runAgent(ctx, userID, convID, history, input, cb)
	if err != nil {
		return nil, err
	}

	var aiMsg *models.ChatMessage
	if strings.TrimSpace(reply) != "" {
		aiMsg = &models.ChatMessage{
			UserID:         userID,
			ConversationID: convID,
			Role:           roleAssistant,
			Content:        reply,
		}
		if err := s.db.WithContext(ctx).Create(aiMsg).Error; err != nil {
			return nil, fmt.Errorf("save assistant message: %w", err)
		}
	}

	if len(history) == 0 && !isConfirmation && aiMsg != nil {
		title := extractTitle(reply)
		s.db.WithContext(ctx).Model(&models.Conversation{}).
			Where("id = ?", convID).
			Update("title", title)
	}

	return &SendResult{
		ConversationID:      convID,
		Title:               defaultTitle,
		UserMessage:         userMsg,
		Reply:               reply,
		PendingConfirmation: pendingConf,
	}, nil
}

func adkSessionID(userID, convID uint) string {
	return fmt.Sprintf("%d-%d", userID, convID)
}

func (s *ChatService) ensureADKSession(ctx context.Context, userID, convID uint, history []models.ChatMessage) (string, error) {
	userIDStr := strconv.FormatUint(uint64(userID), 10)
	sessionID := adkSessionID(userID, convID)

	_, err := s.session.Get(ctx, &session.GetRequest{
		AppName:   appName,
		UserID:    userIDStr,
		SessionID: sessionID,
	})
	if err == nil {
		return sessionID, nil
	}

	createResp, cerr := s.session.Create(ctx, &session.CreateRequest{
		AppName:   appName,
		UserID:    userIDStr,
		SessionID: sessionID,
		State: map[string]any{
			"user_id":           userIDStr,
			"conversation_id": strconv.FormatUint(uint64(convID), 10),
		},
	})
	if cerr != nil {
		if strings.Contains(cerr.Error(), "already exists") {
			return sessionID, nil
		}
		return "", fmt.Errorf("create adk session: %w", cerr)
	}
	if err := s.replayHistory(ctx, createResp.Session, history); err != nil {
		return "", fmt.Errorf("replay history: %w", err)
	}
	return sessionID, nil
}

func (s *ChatService) runAgent(
	ctx context.Context,
	userID, convID uint,
	history []models.ChatMessage,
	input SendInput,
	cb StreamCallback,
) (string, bool, error) {
	sessionID, err := s.ensureADKSession(ctx, userID, convID, history)
	if err != nil {
		return "", false, err
	}

	userIDStr := strconv.FormatUint(uint64(userID), 10)

	var userContent *genai.Content
	if input.ToolConfirmation != nil {
		tc := input.ToolConfirmation
		resp := map[string]any{"confirmed": tc.Confirmed}
		if tc.Payload != nil {
			resp["payload"] = tc.Payload
		}
		userContent = &genai.Content{
			Role: genai.RoleUser,
			Parts: []*genai.Part{{
				FunctionResponse: &genai.FunctionResponse{
					Name:     toolconfirmation.FunctionCallName,
					ID:       tc.CallID,
					Response: resp,
				},
			}},
		}
	} else {
		userParts := []*genai.Part{{Text: input.Message}}
		for _, img := range input.Images {
			if p := dataURLToPart(img); p != nil {
				userParts = append(userParts, p)
			}
		}
		userContent = &genai.Content{Role: genai.RoleUser, Parts: userParts}
	}

	var finalReply strings.Builder
	var emittedConfirmation bool
	var seenConfirmationID string

runLoop:
	for ev, runErr := range s.runner.Run(ctx, userIDStr, sessionID, userContent, adkagent.RunConfig{
		StreamingMode: adkagent.StreamingModeSSE,
	}) {
		if runErr != nil {
			return "", emittedConfirmation, runErr
		}
		if ev == nil || ev.Content == nil {
			continue
		}

		if ev.Partial {
			if ev.Author != hvagent.Name {
				continue
			}
			for _, p := range ev.Content.Parts {
				if p.Text == "" || p.Thought {
					continue
				}
				if !cb(StreamEvent{Token: p.Text}) {
					break runLoop
				}
			}
			continue
		}

		if ev.Author != hvagent.Name {
			continue
		}

		for _, p := range ev.Content.Parts {
			if p.FunctionCall != nil && p.FunctionCall.Name == toolconfirmation.FunctionCallName {
				id := p.FunctionCall.ID
				if id != "" && id == seenConfirmationID {
					continue
				}
				if id != "" {
					seenConfirmationID = id
				}

				orig, oerr := toolconfirmation.OriginalCallFrom(p.FunctionCall)
				var origMap map[string]any
				var origName string
				if oerr == nil && orig != nil {
					origName = orig.Name
					b, _ := json.Marshal(orig)
					_ = json.Unmarshal(b, &origMap)
				}

				payload := &ToolConfirmationPayload{
					ConfirmationCallID:   p.FunctionCall.ID,
					Hint:                 friendlyToolHint(origName),
					OriginalFunctionCall: origMap,
				}
				if !cb(StreamEvent{ToolConfirmation: payload}) {
					break runLoop
				}
				emittedConfirmation = true
				continue
			}

			if p.FunctionCall != nil || p.FunctionResponse != nil || p.Thought {
				continue
			}
			if p.Text != "" {
				finalReply.WriteString(p.Text)
			}
		}
	}

		reply := finalReply.String()
		pending := emittedConfirmation && strings.TrimSpace(reply) == ""
		return reply, pending, nil
}

// friendlyToolHint returns a user-facing Chinese confirmation message for a tool,
// avoiding computer terminology (FunctionResponse, ToolConfirmation, etc.) so that
// non-technical users can understand what action needs their approval.
func friendlyToolHint(toolName string) string {
	switch toolName {
	case "create_medicine":
		return "即将为您新增药品记录，请确认药品名称和说明是否准确。"
	case "update_medicine":
		return "即将为您修改药品信息，请确认修改内容是否正确。"
	case "delete_medicine":
		return "即将为您删除该药品，删除后相关的服药提醒也会一并移除，请确认是否继续。"
	case "create_reminder":
		return "即将为您设置新的服药提醒，请确认提醒时间和对应药品是否正确。"
	case "update_reminder":
		return "即将为您修改服药提醒，请确认修改内容无误。"
	case "delete_reminder":
		return "即将为您删除该服药提醒，请确认是否继续。"
	default:
		return "模型请求执行一项操作，请确认是否继续。"
	}
}

// replayHistory writes prior chat_messages into the freshly created ADK
// session as events, so the model sees the same conversation context the
// user sees in the UI. Tool call/response events from prior turns are not
// available in our schema today, so only plain text turns are replayed.
func (s *ChatService) replayHistory(ctx context.Context, sess session.Session, history []models.ChatMessage) error {
	for i := range history {
		m := &history[i]
		role := genai.RoleUser
		author := "user"
		if m.Role == roleAssistant {
			role = genai.RoleModel
			author = hvagent.Name
		}

		parts := []*genai.Part{{Text: m.Content}}
		if m.Images != "" {
			var stored []string
			if json.Unmarshal([]byte(m.Images), &stored) == nil {
				for _, img := range stored {
					if p := dataURLToPart(img); p != nil {
						parts = append(parts, p)
					}
				}
			}
		}

		ev := session.NewEvent("history")
		ev.Author = author
		ev.LLMResponse = model.LLMResponse{
			Content: &genai.Content{Role: role, Parts: parts},
		}
		if err := s.session.AppendEvent(ctx, sess, ev); err != nil {
			return err
		}
	}
	return nil
}

// ListConversations returns conversations for a user.
func (s *ChatService) ListConversations(ctx context.Context, userID uint) ([]models.Conversation, error) {
	var convs []models.Conversation
	err := s.db.WithContext(ctx).
		Where("user_id = ?", userID).
		Order("updated_at DESC").
		Find(&convs).Error
	return convs, err
}

// GetMessages returns messages for a conversation.
func (s *ChatService) GetMessages(ctx context.Context, conversationID uint, userID uint) ([]models.ChatMessage, error) {
	var msgs []models.ChatMessage
	err := s.db.WithContext(ctx).
		Where("conversation_id = ? AND user_id = ?", conversationID, userID).
		Order("created_at ASC").
		Find(&msgs).Error
	return msgs, err
}

// DeleteConversation deletes a conversation and its messages.
func (s *ChatService) DeleteConversation(ctx context.Context, id uint, userID uint) error {
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var conv models.Conversation
		if err := tx.Where("id = ? AND user_id = ?", id, userID).First(&conv).Error; err != nil {
			return err
		}
		if err := tx.Where("conversation_id = ?", id).Delete(&models.ChatMessage{}).Error; err != nil {
			return err
		}
		return tx.Delete(&conv).Error
	})
	if err != nil {
		return err
	}

	userIDStr := strconv.FormatUint(uint64(userID), 10)
	sid := adkSessionID(userID, id)
	_ = s.session.Delete(ctx, &session.DeleteRequest{
		AppName:   appName,
		UserID:    userIDStr,
		SessionID: sid,
	})
	return nil
}

func (s *ChatService) getHistory(ctx context.Context, convID uint, userID uint, excludeLastUser bool) ([]models.ChatMessage, error) {
	var msgs []models.ChatMessage
	err := s.db.WithContext(ctx).
		Where("conversation_id = ? AND user_id = ?", convID, userID).
		Order("created_at DESC").
		Limit(maxHistory + 1).
		Find(&msgs).Error
	if err != nil {
		return nil, err
	}
	for i, j := 0, len(msgs)-1; i < j; i, j = i+1, j-1 {
		msgs[i], msgs[j] = msgs[j], msgs[i]
	}
	if excludeLastUser && len(msgs) > 0 && msgs[len(msgs)-1].Role == roleUser {
		msgs = msgs[:len(msgs)-1]
	}
	return msgs, nil
}

// marshalImages serializes image data URLs to a JSON array for DB storage.
func marshalImages(images []string) string {
	if len(images) == 0 {
		return ""
	}
	b, _ := json.Marshal(images)
	return string(b)
}

// dataURLToPart parses a data URL (data:image/jpeg;base64,...) into a genai.Part with InlineData.
func dataURLToPart(dataURL string) *genai.Part {
	mime, b64, ok := parseDataURL(dataURL)
	if !ok {
		return nil
	}
	raw, err := base64.StdEncoding.DecodeString(b64)
	if err != nil {
		return nil
	}
	return &genai.Part{
		InlineData: &genai.Blob{Data: raw, MIMEType: mime},
	}
}

// parseDataURL extracts MIME type and base64 payload from a data URL.
func parseDataURL(url string) (mime, b64 string, ok bool) {
	const prefix = "data:"
	if !strings.HasPrefix(url, prefix) {
		return "", "", false
	}
	rest := url[len(prefix):]
	idx := strings.Index(rest, ";base64,")
	if idx < 0 {
		return "", "", false
	}
	return rest[:idx], rest[idx+len(";base64,"):], true
}

func extractTitle(reply string) string {
	title := strings.TrimSpace(reply)
	if idx := strings.Index(title, "\n"); idx >= 0 {
		title = title[:idx]
	}
	title = strings.TrimSpace(title)
	runes := []rune(title)
	if len(runes) > 50 {
		return string(runes[:50]) + "..."
	}
	if title == "" {
		return defaultTitle
	}
	return title
}
