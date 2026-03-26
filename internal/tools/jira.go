package tools

import (
	"context"
	"customclaw/internal/llm"
	"encoding/json"
	"fmt"
	"net/http"
)

// JiraGetTicket fetches ticket details from Jira.
type JiraGetTicket struct {
	baseURL  string
	user     string
	apiToken string
}

func NewJiraGetTicket(baseURL, user, apiToken string) *JiraGetTicket {
	return &JiraGetTicket{baseURL: baseURL, user: user, apiToken: apiToken}
}

func (t *JiraGetTicket) Name() string { return "jira_get_ticket" }

func (t *JiraGetTicket) Description() string {
	return "Fetch details of a Jira ticket by its ID (e.g. PROJ-123)."
}

func (t *JiraGetTicket) Parameters() llm.ParameterSchema {
	return llm.ParameterSchema{
		Type: "object",
		Properties: map[string]llm.PropertySchema{
			"ticket_id": {Type: "string", Description: "The Jira ticket ID, e.g. PROJ-123."},
		},
		Required: []string{"ticket_id"},
	}
}

func (t *JiraGetTicket) Execute(ctx context.Context, input map[string]any) (string, error) {
	ticketID, _ := input["ticket_id"].(string)
	if ticketID == "" {
		return "", fmt.Errorf("ticket_id is required")
	}

	url := fmt.Sprintf("%s/rest/api/3/issue/%s", t.baseURL, ticketID)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", fmt.Errorf("build request: %w", err)
	}
	req.SetBasicAuth(t.user, t.apiToken)
	req.Header.Set("Accept", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == 404 {
		return "", fmt.Errorf("ticket %s not found", ticketID)
	}
	if resp.StatusCode >= 300 {
		return "", fmt.Errorf("jira returned status %d", resp.StatusCode)
	}

	var result struct {
		Key    string `json:"key"`
		Fields struct {
			Summary     string `json:"summary"`
			Description any    `json:"description"`
			Status      struct {
				Name string `json:"name"`
			} `json:"status"`
			Assignee *struct {
				DisplayName string `json:"displayName"`
			} `json:"assignee"`
		} `json:"fields"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("decode response: %w", err)
	}

	assignee := "unassigned"
	if result.Fields.Assignee != nil {
		assignee = result.Fields.Assignee.DisplayName
	}

	return fmt.Sprintf("Ticket: %s\nSummary: %s\nStatus: %s\nAssignee: %s",
		result.Key,
		result.Fields.Summary,
		result.Fields.Status.Name,
		assignee,
	), nil
}
