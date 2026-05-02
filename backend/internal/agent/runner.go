package agent

import (
	"context"
	"iter"

	"google.golang.org/adk/agent"
	"google.golang.org/adk/memory"
	"google.golang.org/adk/runner"
	"google.golang.org/adk/session"
	"google.golang.org/genai"
)

// Setup creates the agent, runner, session service, and memory service.
func Setup(appName string, rootAgent agent.Agent) (*runner.Runner, session.Service) {
	sessionSvc := session.InMemoryService()
	memorySvc := memory.InMemoryService()

	r, err := runner.New(runner.Config{
		AppName:           appName,
		Agent:             rootAgent,
		SessionService:    sessionSvc,
		MemoryService:     memorySvc,
		AutoCreateSession: true,
	})
	if err != nil {
		panic("setup agent runner: " + err.Error())
	}

	return r, sessionSvc
}

// Run is a convenience wrapper around Runner.Run.
func Run(ctx context.Context, r *runner.Runner, userID, sessionID, message string) iter.Seq2[*session.Event, error] {
	return r.Run(ctx, userID, sessionID, &genai.Content{
		Role: "user",
		Parts: []*genai.Part{
			{Text: message},
		},
	}, agent.RunConfig{})
}
