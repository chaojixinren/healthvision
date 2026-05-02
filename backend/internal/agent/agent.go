package agent

import (
	"google.golang.org/adk/agent"
	"google.golang.org/adk/agent/llmagent"
	"google.golang.org/adk/model"
)

// New creates an LLM agent with the given name, model and instruction.
func New(name string, m model.LLM, instruction string) (agent.Agent, error) {
	return llmagent.New(llmagent.Config{
		Name:        name,
		Model:       m,
		Instruction: instruction,
	})
}
