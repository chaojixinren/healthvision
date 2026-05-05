// Package agent assembles the ADK LLM agent for HealthVision and exposes a
// thin facade for ChatService. Keeping the wiring here lets us iterate on
// agent configuration (instruction, tools, sub-agents) without touching the
// HTTP layer.
package agent

import (
	"fmt"

	adkagent "google.golang.org/adk/agent"
	"google.golang.org/adk/agent/llmagent"
	"google.golang.org/adk/model"
	adktool "google.golang.org/adk/tool"
)

// Name is the canonical name of the root agent. We expose it so the chat
// service can author historical events with the same Author when replaying
// conversation history into an ADK session.
const Name = "healthvision_assistant"

// Build constructs the root LLM agent. The caller passes the underlying
// model.LLM (already configured for the OpenAI-compatible endpoint), the
// list of tools, and an optional dynamic instruction provider.
//
// When provider is nil the agent falls back to a static prompt
// (defaultInstruction). When non-nil the provider is invoked once per agent
// invocation and may inject per-user context such as a medicine snapshot.
// Per the ADK contract, InstructionProvider takes precedence over the
// Instruction field whenever both are set, so a non-nil provider must
// produce the full system prompt itself (it should embed defaultInstruction
// when in doubt — see NewDynamicInstruction).
func Build(llm model.LLM, tools []adktool.Tool, provider llmagent.InstructionProvider) (adkagent.Agent, error) {
	if llm == nil {
		return nil, fmt.Errorf("agent.Build: model.LLM is required")
	}

	cfg := llmagent.Config{
		Name:        Name,
		Description: "HealthVision 用药管理智能助手，可调用工具读取并管理用户的药品和提醒。",
		Model:       llm,
		Tools:       tools,
	}
	if provider != nil {
		cfg.InstructionProvider = provider
	} else {
		cfg.Instruction = defaultInstruction
	}

	a, err := llmagent.New(cfg)
	if err != nil {
		return nil, fmt.Errorf("agent.Build: %w", err)
	}
	return a, nil
}

const defaultInstruction = `你是 HealthVision 的健康助手，专注于帮助用户（含老年用户）管理药品与服药提醒。

可用工具说明：
- list_medicines / get_medicine：查询当前用户的药品库
- create_medicine / update_medicine / delete_medicine：管理药品（写操作）
- list_reminders / create_reminder / update_reminder / delete_reminder：管理服药提醒（写操作）

工作原则：
1. 涉及"我有什么药"、"我有哪些提醒"等问题时，优先调用对应的查询工具，不要凭推测回答。
2. 创建或修改药品/提醒前，先确认关键字段（药名、HH:MM 时间）。如果用户说"明天 8 点提醒我吃阿莫西林"，但药品库里没有阿莫西林，应先调用 list_medicines 检查；缺失时应先 create_medicine。
3. 时间一律用 24 小时制 HH:MM 格式（例如 08:00、20:30）。
4. 子女账户给绑定的老人创建提醒时，需要在 create_reminder 中带 target_user_id；普通自用情况可省略。
5. 每次工具调用结束后，用一句简短的中文向用户复述结果（不要罗列原始 JSON）。`
