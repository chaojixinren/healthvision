package agent

import (
	"net/http"

	"google.golang.org/adk/agent"
	"google.golang.org/adk/runner"
	"google.golang.org/adk/server/adkrest"
	"google.golang.org/adk/session"
)

// NewHandler creates an HTTP handler for the ADK REST API.
func NewHandler(r *runner.Runner, rootAgent agent.Agent, sessionSvc session.Service) (http.Handler, error) {
	return adkrest.NewServer(adkrest.ServerConfig{
		AgentLoader:    agent.NewSingleLoader(rootAgent),
		SessionService: sessionSvc,
	})
}
