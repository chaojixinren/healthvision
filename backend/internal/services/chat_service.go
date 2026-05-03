package services

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"

	"healthvision/backend/internal/models"

	"google.golang.org/adk/model"
	"google.golang.org/genai"
	"gorm.io/gorm"
)

const (
	maxHistory   = 20
	defaultTitle = "新对话"

	roleUser      = "user"
	roleAssistant = "assistant"
	roleModel     = "model"
)

// ChatService handles chat message processing.
type ChatService struct {
	db   *gorm.DB
	llm  model.LLM
	inst string
}

// NewChatService creates a ChatService.
func NewChatService(db *gorm.DB, llm model.LLM, instruction string) *ChatService {
	return &ChatService{db: db, llm: llm, inst: instruction}
}

// SendInput contains the data needed to process a chat message.
type SendInput struct {
	UserID         uint
	ConversationID uint
	Message        string
	Images         []string
}

// SendResult is returned after message processing completes.
type SendResult struct {
	ConversationID uint
	Title          string
	UserMessage    *models.ChatMessage
	Reply          string
}

// StreamCallback is called for each token during streaming.
type StreamCallback func(token string, err error) bool

// Send processes a user message and streams the AI reply via callback.
func (s *ChatService) Send(ctx context.Context, input SendInput, cb StreamCallback) (*SendResult, error) {
	userID := input.UserID
	convID := input.ConversationID

	if convID == 0 {
		conv := &models.Conversation{
			UserID: userID,
			Title:  defaultTitle,
		}
		if err := s.db.WithContext(ctx).Create(conv).Error; err != nil {
			return nil, fmt.Errorf("create conversation: %w", err)
		}
		convID = conv.ID
	}

	imagesJSON := marshalImages(input.Images)

	userMsg := &models.ChatMessage{
		UserID:         userID,
		ConversationID: convID,
		Role:           roleUser,
		Content:        input.Message,
		Images:         imagesJSON,
	}
	if err := s.db.WithContext(ctx).Create(userMsg).Error; err != nil {
		return nil, fmt.Errorf("save user message: %w", err)
	}

	history, err := s.getHistory(ctx, convID, userID)
	if err != nil {
		return nil, err
	}

	contents := buildContents(history, input.Message, input.Images)

	var reply strings.Builder
	for resp, err := range s.llm.GenerateContent(ctx, &model.LLMRequest{
		Contents: contents,
		Config: &genai.GenerateContentConfig{
			SystemInstruction: &genai.Content{
				Parts: []*genai.Part{{Text: s.inst}},
			},
		},
	}, true) {
		if err != nil {
			cb("", err)
			return nil, fmt.Errorf("llm error: %w", err)
		}
		if resp == nil || resp.Content == nil {
			continue
		}
		for _, part := range resp.Content.Parts {
			if part.Text != "" {
				reply.WriteString(part.Text)
				if !cb(part.Text, nil) {
					break
				}
			}
		}
	}

	replyText := reply.String()

	aiMsg := &models.ChatMessage{
		UserID:         userID,
		ConversationID: convID,
		Role:           roleAssistant,
		Content:        replyText,
	}
	if err := s.db.WithContext(ctx).Create(aiMsg).Error; err != nil {
		return nil, fmt.Errorf("save assistant message: %w", err)
	}

	// Auto-title on first exchange
	if len(history) == 0 {
		title := extractTitle(replyText)
		s.db.WithContext(ctx).Model(&models.Conversation{}).
			Where("id = ?", convID).
			Update("title", title)
	}

	return &SendResult{
		ConversationID: convID,
		Title:          defaultTitle,
		UserMessage:    userMsg,
		Reply:          replyText,
	}, nil
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
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var conv models.Conversation
		if err := tx.Where("id = ? AND user_id = ?", id, userID).First(&conv).Error; err != nil {
			return err
		}
		if err := tx.Where("conversation_id = ?", id).Delete(&models.ChatMessage{}).Error; err != nil {
			return err
		}
		return tx.Delete(&conv).Error
	})
}

func (s *ChatService) getHistory(ctx context.Context, convID uint, userID uint) ([]models.ChatMessage, error) {
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
	if len(msgs) > 0 && msgs[len(msgs)-1].Role == roleUser {
		msgs = msgs[:len(msgs)-1]
	}
	return msgs, nil
}

func buildContents(history []models.ChatMessage, userMsg string, images []string) []*genai.Content {
	contents := make([]*genai.Content, 0, len(history)+1)
	for _, m := range history {
		role := m.Role
		if role == roleAssistant {
			role = roleModel
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
		contents = append(contents, &genai.Content{Role: role, Parts: parts})
	}

	parts := []*genai.Part{{Text: userMsg}}
	for _, img := range images {
		if p := dataURLToPart(img); p != nil {
			parts = append(parts, p)
		}
	}
	contents = append(contents, &genai.Content{Role: roleUser, Parts: parts})
	return contents
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
