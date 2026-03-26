package triggers

import (
	"context"
	"customclaw/internal/agent"
	"customclaw/internal/config"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
)

// WebhookServer listens for incoming webhook events and dispatches them to the agent.
type WebhookServer struct {
	cfg     *config.Config
	actions *config.Actions
	agent   *agent.Agent
	mux     *http.ServeMux
}

func NewWebhookServer(cfg *config.Config, actions *config.Actions, ag *agent.Agent) *WebhookServer {
	s := &WebhookServer{
		cfg:     cfg,
		actions: actions,
		agent:   ag,
		mux:     http.NewServeMux(),
	}
	s.registerRoutes()
	return s
}

func (s *WebhookServer) registerRoutes() {
	for _, wf := range s.actions.Workflows {
		if wf.Trigger.Type != "webhook" {
			continue
		}
		wf := wf // capture loop var
		s.mux.HandleFunc(wf.Trigger.Path, func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodPost {
				http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
				return
			}
			s.handleWebhook(w, r, wf)
		})
		log.Printf("registered webhook: POST %s → workflow '%s'", wf.Trigger.Path, wf.Name)
	}
}

func (s *WebhookServer) handleWebhook(w http.ResponseWriter, r *http.Request, wf config.Workflow) {
	body, err := io.ReadAll(io.LimitReader(r.Body, 1<<20)) // 1 MB limit
	if err != nil {
		http.Error(w, "failed to read body", http.StatusBadRequest)
		return
	}

	var payload map[string]any
	if err := json.Unmarshal(body, &payload); err != nil {
		http.Error(w, "invalid JSON body", http.StatusBadRequest)
		return
	}

	log.Printf("webhook received: workflow=%s service=%s event=%s", wf.Name, wf.Trigger.Service, wf.Trigger.Event)
	w.WriteHeader(http.StatusAccepted)
	fmt.Fprintln(w, `{"status":"accepted"}`)

	// Run the agent asynchronously so the webhook returns immediately.
	go func() {
		ctx := context.Background()
		result, err := s.agent.Run(ctx, wf.Goal, s.actions.Tools, payload)
		if err != nil {
			log.Printf("agent error (workflow=%s): %v", wf.Name, err)
			return
		}
		log.Printf("agent done (workflow=%s): %s", wf.Name, result)
	}()
}

// Start starts the HTTP server.
func (s *WebhookServer) Start() error {
	addr := fmt.Sprintf(":%d", s.cfg.Server.Port)
	log.Printf("customclaw listening on %s", addr)
	return http.ListenAndServe(addr, s.mux)
}
