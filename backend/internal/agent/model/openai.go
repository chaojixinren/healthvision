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
	"slices"
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

// openAIModel implements model.LLM for any OpenAI-compatible API, including
// vendors that follow the chat/completions schema (OpenAI, DeepSeek, Qwen,
// SiliconFlow, etc.). It supports text, vision (via inline base64 images)
// and function calling (tools / tool_calls).
type openAIModel struct {
	name    string
	baseURL string
	apiKey  string
	client  *http.Client
}

// NewOpenAI returns a model.LLM backed by the OpenAI-compatible chat API.
func NewOpenAI(cfg OpenAIConfig) model.LLM {
	return &openAIModel{
		name:    cfg.Name,
		baseURL: strings.TrimRight(cfg.BaseURL, "/"),
		apiKey:  cfg.APIKey,
		client:  &http.Client{},
	}
}

func (m *openAIModel) Name() string { return m.name }

func (m *openAIModel) GenerateContent(ctx context.Context, req *model.LLMRequest, stream bool) iter.Seq2[*model.LLMResponse, error] {
	if stream {
		return m.generateStream(ctx, req)
	}
	return m.generate(ctx, req)
}

// ---------- non-streaming ----------

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
	parts := make([]*genai.Part, 0, 1+len(choice.Message.ToolCalls))
	if choice.Message.Content != "" {
		parts = append(parts, &genai.Part{Text: choice.Message.Content})
	}
	for _, tc := range choice.Message.ToolCalls {
		fc, err := decodeToolCall(tc)
		if err != nil {
			return nil, err
		}
		parts = append(parts, &genai.Part{FunctionCall: fc})
	}

	return &model.LLMResponse{
		Content: &genai.Content{
			Role:  genai.RoleModel,
			Parts: parts,
		},
		FinishReason: mapFinishReason(choice.FinishReason),
		TurnComplete: true,
	}, nil
}

// ---------- streaming ----------

// generateStream emits text deltas as partial events and accumulates any
// tool_calls into a single non-partial trailing event when finish_reason is
// observed. ADK detects tool calls by inspecting the final non-partial event,
// so it is important not to surface partial tool_call fragments mid-stream.
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

		var (
			text         strings.Builder
			toolBuffers  = map[int]*toolCallBuilder{}
			finishReason string
		)

		scanner := bufio.NewScanner(resp.Body)
		// SSE chunks may exceed the default 64KB scanner buffer when tool args
		// are large; raise the cap to a safer value.
		scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)

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

			// Text deltas: forward immediately as partial events so the UI
			// can stream tokens.
			if choice.Delta.Content != "" {
				text.WriteString(choice.Delta.Content)
				partial := &model.LLMResponse{
					Partial: true,
					Content: &genai.Content{
						Role:  genai.RoleModel,
						Parts: []*genai.Part{{Text: choice.Delta.Content}},
					},
				}
				if !yield(partial, nil) {
					return
				}
			}

			// Tool call deltas: just buffer; never emit while still partial.
			for _, tc := range choice.Delta.ToolCalls {
				idx := tc.Index
				b, ok := toolBuffers[idx]
				if !ok {
					b = &toolCallBuilder{}
					toolBuffers[idx] = b
				}
				if tc.ID != "" {
					b.id = tc.ID
				}
				if tc.Function.Name != "" {
					b.name = tc.Function.Name
				}
				if tc.Function.Arguments != "" {
					b.args.WriteString(tc.Function.Arguments)
				}
			}

			if choice.FinishReason != "" {
				finishReason = choice.FinishReason
			}
		}
		if err := scanner.Err(); err != nil {
			yield(nil, fmt.Errorf("read stream: %w", err))
			return
		}

		// Build the trailing aggregated event. ADK relies on this final
		// event (Partial=false) to detect FunctionCall parts and trigger
		// the tool-execution loop.
		parts := make([]*genai.Part, 0, 1+len(toolBuffers))
		if text.Len() > 0 {
			parts = append(parts, &genai.Part{Text: text.String()})
		}
		for _, idx := range sortedKeys(toolBuffers) {
			b := toolBuffers[idx]
			fc, err := decodeToolCall(openAIToolCall{
				ID:       b.id,
				Type:     "function",
				Function: openAIToolCallFunc{Name: b.name, Arguments: b.args.String()},
			})
			if err != nil {
				yield(nil, err)
				return
			}
			parts = append(parts, &genai.Part{FunctionCall: fc})
		}

		final := &model.LLMResponse{
			Partial:      false,
			TurnComplete: true,
			FinishReason: mapFinishReason(finishReason),
			Content: &genai.Content{
				Role:  genai.RoleModel,
				Parts: parts,
			},
		}
		yield(final, nil)
	}
}

// ---------- request building ----------

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
		messages = append(messages, contentToOpenAIMessages(content)...)
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
		body.Tools = collectOpenAITools(req.Config.Tools)
	}

	return json.Marshal(body)
}

// contentToOpenAIMessages translates a single ADK Content into one or more
// OpenAI chat messages. A Content with a FunctionCall becomes an assistant
// message with tool_calls; FunctionResponse parts each become their own
// role:"tool" message. Pure text/image contents map to the natural role.
func contentToOpenAIMessages(content *genai.Content) []chatMessage {
	if content == nil || len(content.Parts) == 0 {
		return nil
	}

	role := content.Role
	switch role {
	case "model":
		role = "assistant"
	case "":
		role = "user"
	}

	var (
		texts        []string
		imageBlocks  []visionBlock
		hasImage     bool
		toolCalls    []openAIToolCall
		toolMessages []chatMessage
	)

	for _, p := range content.Parts {
		switch {
		case p.FunctionCall != nil:
			argsJSON, _ := json.Marshal(p.FunctionCall.Args)
			toolCalls = append(toolCalls, openAIToolCall{
				ID:   nonEmpty(p.FunctionCall.ID, "call_"+p.FunctionCall.Name),
				Type: "function",
				Function: openAIToolCallFunc{
					Name:      p.FunctionCall.Name,
					Arguments: string(argsJSON),
				},
			})
		case p.FunctionResponse != nil:
			respJSON, _ := json.Marshal(p.FunctionResponse.Response)
			toolMessages = append(toolMessages, chatMessage{
				Role:       "tool",
				ToolCallID: nonEmpty(p.FunctionResponse.ID, "call_"+p.FunctionResponse.Name),
				Name:       p.FunctionResponse.Name,
				Content:    string(respJSON),
			})
		case p.InlineData != nil && len(p.InlineData.Data) > 0:
			hasImage = true
			imageBlocks = append(imageBlocks, visionBlock{
				Type: "image_url",
				ImageURL: &imageURL{
					URL: fmt.Sprintf("data:%s;base64,%s",
						p.InlineData.MIMEType,
						base64.StdEncoding.EncodeToString(p.InlineData.Data)),
				},
			})
		case p.Text != "":
			texts = append(texts, p.Text)
		}
	}

	var out []chatMessage

	// Assistant tool_call message (must come before tool result messages).
	if len(toolCalls) > 0 {
		msg := chatMessage{Role: "assistant", ToolCalls: toolCalls}
		if len(texts) > 0 {
			msg.Content = strings.Join(texts, "\n")
		}
		out = append(out, msg)
		texts = nil // text was consumed by the assistant tool_call message
	}

	// Tool result messages.
	out = append(out, toolMessages...)

	// Plain text / vision message.
	if len(texts) > 0 || hasImage {
		var contentBody any
		if hasImage {
			blocks := make([]visionBlock, 0, len(texts)+len(imageBlocks))
			for _, t := range texts {
				blocks = append(blocks, visionBlock{Type: "text", Text: t})
			}
			blocks = append(blocks, imageBlocks...)
			contentBody = blocks
		} else {
			contentBody = strings.Join(texts, "\n")
		}
		out = append(out, chatMessage{Role: role, Content: contentBody})
	}

	return out
}

// collectOpenAITools converts ADK function declarations into the OpenAI
// `tools` request field. Non-function tools (e.g. retrieval) are ignored
// because OpenAI-compatible endpoints typically do not understand them.
func collectOpenAITools(tools []*genai.Tool) []openAITool {
	if len(tools) == 0 {
		return nil
	}
	var out []openAITool
	for _, t := range tools {
		if t == nil {
			continue
		}
		for _, decl := range t.FunctionDeclarations {
			if decl == nil || decl.Name == "" {
				continue
			}
			fn := openAIToolFunction{
				Name:        decl.Name,
				Description: decl.Description,
			}
			switch {
			case decl.ParametersJsonSchema != nil:
				fn.Parameters = decl.ParametersJsonSchema
			case decl.Parameters != nil:
				fn.Parameters = decl.Parameters
			}
			out = append(out, openAITool{Type: "function", Function: fn})
		}
	}
	return out
}

// ---------- response decoding ----------

func decodeToolCall(tc openAIToolCall) (*genai.FunctionCall, error) {
	if tc.Function.Name == "" {
		return nil, fmt.Errorf("tool_call missing function name")
	}
	args := map[string]any{}
	raw := strings.TrimSpace(tc.Function.Arguments)
	if raw != "" {
		if err := json.Unmarshal([]byte(raw), &args); err != nil {
			return nil, fmt.Errorf("decode tool_call args for %q: %w", tc.Function.Name, err)
		}
	}
	return &genai.FunctionCall{
		ID:   tc.ID,
		Name: tc.Function.Name,
		Args: args,
	}, nil
}

// ---------- helpers ----------

func (m *openAIModel) setHeaders(req *http.Request) {
	req.Header.Set("Content-Type", "application/json")
	if m.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+m.apiKey)
	}
}

func mapFinishReason(reason string) genai.FinishReason {
	switch reason {
	case "stop", "tool_calls", "function_call":
		return genai.FinishReasonStop
	case "length":
		return genai.FinishReasonMaxTokens
	case "content_filter":
		return genai.FinishReasonSafety
	case "":
		return ""
	default:
		return genai.FinishReasonStop
	}
}

func nonEmpty(s, fallback string) string {
	if s == "" {
		return fallback
	}
	return s
}

func sortedKeys(m map[int]*toolCallBuilder) []int {
	keys := make([]int, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	slices.Sort(keys)
	return keys
}

// ---------- wire types ----------

type chatMessage struct {
	Role       string           `json:"role"`
	Content    any              `json:"content,omitempty"`
	Name       string           `json:"name,omitempty"`
	ToolCalls  []openAIToolCall `json:"tool_calls,omitempty"`
	ToolCallID string           `json:"tool_call_id,omitempty"`
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
	Tools       []openAITool  `json:"tools,omitempty"`
}

type openAITool struct {
	Type     string             `json:"type"`
	Function openAIToolFunction `json:"function"`
}

type openAIToolFunction struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	Parameters  any    `json:"parameters,omitempty"`
}

type openAIToolCall struct {
	Index    int                `json:"index,omitempty"`
	ID       string             `json:"id,omitempty"`
	Type     string             `json:"type,omitempty"`
	Function openAIToolCallFunc `json:"function"`
}

type openAIToolCallFunc struct {
	Name      string `json:"name,omitempty"`
	Arguments string `json:"arguments,omitempty"`
}

type openAIChatResponse struct {
	Choices []struct {
		Message struct {
			Content   string           `json:"content"`
			ToolCalls []openAIToolCall `json:"tool_calls,omitempty"`
		} `json:"message"`
		FinishReason string `json:"finish_reason"`
	} `json:"choices"`
}

type openAIStreamChunk struct {
	Choices []struct {
		Delta struct {
			Content   string           `json:"content"`
			ToolCalls []openAIToolCall `json:"tool_calls,omitempty"`
		} `json:"delta"`
		FinishReason string `json:"finish_reason"`
	} `json:"choices"`
}

type toolCallBuilder struct {
	id   string
	name string
	args strings.Builder
}
