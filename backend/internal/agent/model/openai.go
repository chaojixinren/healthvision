package model

import (
	"bufio"
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"iter"
	"net/http"
	"strings"

	"google.golang.org/adk/model"
	"google.golang.org/genai"
)

// OpenAIConfig holds the configuration for an OpenAI-compatible model.
type OpenAIConfig struct {
	Name    string
	BaseURL string
	APIKey  string
}

// openAIModel implements model.LLM for any OpenAI-compatible API.
type openAIModel struct {
	name    string
	baseURL string
	apiKey  string
	client  *http.Client
}

func NewOpenAI(cfg OpenAIConfig) model.LLM {
	return &openAIModel{
		name:    cfg.Name,
		baseURL: strings.TrimRight(cfg.BaseURL, "/"),
		apiKey:  cfg.APIKey,
		client:  &http.Client{},
	}
}

func (m *openAIModel) Name() string {
	return m.name
}

func (m *openAIModel) GenerateContent(ctx context.Context, req *model.LLMRequest, stream bool) iter.Seq2[*model.LLMResponse, error] {
	if stream {
		return m.generateStream(ctx, req)
	}
	return m.generate(ctx, req)
}

func (m *openAIModel) generate(ctx context.Context, req *model.LLMRequest) iter.Seq2[*model.LLMResponse, error] {
	return func(yield func(*model.LLMResponse, error) bool) {
		resp, err := m.chat(ctx, req)
		if err != nil {
			yield(nil, err)
			return
		}
		yield(resp, nil)
	}
}

func (m *openAIModel) generateStream(ctx context.Context, req *model.LLMRequest) iter.Seq2[*model.LLMResponse, error] {
	return func(yield func(*model.LLMResponse, error) bool) {
		body, err := m.buildChatRequest(req, true)
		if err != nil {
			yield(nil, err)
			return
		}

		httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, m.baseURL+"/chat/completions", bytes.NewReader(body))
		if err != nil {
			yield(nil, err)
			return
		}
		m.setHeaders(httpReq)

		resp, err := m.client.Do(httpReq)
		if err != nil {
			yield(nil, err)
			return
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			b, _ := io.ReadAll(resp.Body)
			yield(nil, fmt.Errorf("openai stream: %s - %s", resp.Status, string(b)))
			return
		}

		scanner := bufio.NewScanner(resp.Body)
		for scanner.Scan() {
			line := strings.TrimSpace(scanner.Text())
			if line == "" || line == "data: [DONE]" {
				continue
			}
			if !strings.HasPrefix(line, "data: ") {
				continue
			}
			payload := strings.TrimPrefix(line, "data: ")

			var chunk openAIStreamChunk
			if err := json.Unmarshal([]byte(payload), &chunk); err != nil {
				continue
			}
			if len(chunk.Choices) == 0 {
				continue
			}
			choice := chunk.Choices[0]

			adkResp := &model.LLMResponse{
				Partial:      true,
				TurnComplete: choice.FinishReason != "",
			}
			if choice.Delta.Content != "" {
				adkResp.Content = &genai.Content{
					Role: "model",
					Parts: []*genai.Part{
						{Text: choice.Delta.Content},
					},
				}
			}
			if choice.FinishReason != "" {
				adkResp.FinishReason = mapFinishReason(choice.FinishReason)
				adkResp.TurnComplete = true
				adkResp.Partial = false
			}
			if !yield(adkResp, nil) {
				return
			}
		}
	}
}

func (m *openAIModel) chat(ctx context.Context, req *model.LLMRequest) (*model.LLMResponse, error) {
	body, err := m.buildChatRequest(req, false)
	if err != nil {
		return nil, err
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, m.baseURL+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	m.setHeaders(httpReq)

	resp, err := m.client.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("openai: %s - %s", resp.Status, string(b))
	}

	var chatResp openAIChatResponse
	if err := json.NewDecoder(resp.Body).Decode(&chatResp); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	if len(chatResp.Choices) == 0 {
		return nil, fmt.Errorf("openai: no choices in response")
	}

	choice := chatResp.Choices[0]
	adkResp := &model.LLMResponse{
		Content: &genai.Content{
			Role: "model",
			Parts: []*genai.Part{
				{Text: choice.Message.Content},
			},
		},
		FinishReason: mapFinishReason(choice.FinishReason),
		TurnComplete: true,
	}

	return adkResp, nil
}

type chatMessage struct {
	Role    string `json:"role"`
	Content any    `json:"content"`
}

type visionBlock struct {
	Type     string    `json:"type"`
	Text     string    `json:"text,omitempty"`
	ImageURL *imageURL `json:"image_url,omitempty"`
}

type imageURL struct {
	URL string `json:"url"`
}

type openAIChatRequest struct {
	Model       string        `json:"model"`
	Messages    []chatMessage `json:"messages"`
	Temperature *float32      `json:"temperature,omitempty"`
	MaxTokens   int32         `json:"max_tokens,omitempty"`
	TopP        *float32      `json:"top_p,omitempty"`
	Stream      bool          `json:"stream,omitempty"`
}

type openAIChatResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
		FinishReason string `json:"finish_reason"`
	} `json:"choices"`
}

type openAIStreamChunk struct {
	Choices []struct {
		Delta struct {
			Content string `json:"content"`
		} `json:"delta"`
		FinishReason string `json:"finish_reason"`
	} `json:"choices"`
}

func (m *openAIModel) buildChatRequest(req *model.LLMRequest, stream bool) ([]byte, error) {
	var messages []chatMessage

	if req.Config != nil && req.Config.SystemInstruction != nil {
		for _, part := range req.Config.SystemInstruction.Parts {
			if part.Text != "" {
				messages = append(messages, chatMessage{Role: "system", Content: part.Text})
			}
		}
	}

	for _, content := range req.Contents {
		role := content.Role
		if role == "model" {
			role = "assistant"
		}
		messages = append(messages, chatMessage{Role: role, Content: buildContent(content.Parts)})
	}

	modelName := req.Model
	if modelName == "" {
		modelName = m.name
	}

	body := openAIChatRequest{
		Model:    modelName,
		Messages: messages,
		Stream:   stream,
	}

	if req.Config != nil {
		body.Temperature = req.Config.Temperature
		body.TopP = req.Config.TopP
		if req.Config.MaxOutputTokens > 0 {
			body.MaxTokens = req.Config.MaxOutputTokens
		}
	}

	return json.Marshal(body)
}

func buildContent(parts []*genai.Part) any {
	// Check if any image parts exist
	hasImage := false
	var texts []string
	for _, p := range parts {
		if p.InlineData != nil && len(p.InlineData.Data) > 0 {
			hasImage = true
		}
		if p.Text != "" {
			texts = append(texts, p.Text)
		}
	}
	if !hasImage {
		return strings.Join(texts, "\n")
	}

	// Vision format: array of {type, text/image_url} blocks
	blocks := make([]visionBlock, 0, len(parts))
	for _, p := range parts {
		if p.Text != "" {
			blocks = append(blocks, visionBlock{Type: "text", Text: p.Text})
		}
		if p.InlineData != nil && len(p.InlineData.Data) > 0 {
			blocks = append(blocks, visionBlock{
				Type: "image_url",
				ImageURL: &imageURL{
					URL: fmt.Sprintf("data:%s;base64,%s", p.InlineData.MIMEType, base64.StdEncoding.EncodeToString(p.InlineData.Data)),
				},
			})
		}
	}
	return blocks
}

func (m *openAIModel) setHeaders(req *http.Request) {
	req.Header.Set("Content-Type", "application/json")
	if m.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+m.apiKey)
	}
}

func mapFinishReason(reason string) genai.FinishReason {
	switch reason {
	case "stop":
		return genai.FinishReasonStop
	case "length":
		return genai.FinishReasonMaxTokens
	case "content_filter":
		return genai.FinishReasonSafety
	default:
		return genai.FinishReasonStop
	}
}
