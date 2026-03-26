package tools

import (
	"bytes"
	"context"
	"customclaw/internal/llm"
	"encoding/json"
	"fmt"
	"net/http"
)

const githubAPIURL = "https://api.github.com"

type githubClient struct {
	token string
}

func (c *githubClient) do(ctx context.Context, method, path string, body any) (*http.Response, error) {
	var reqBody *bytes.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return nil, err
		}
		reqBody = bytes.NewReader(b)
	} else {
		reqBody = bytes.NewReader(nil)
	}
	req, err := http.NewRequestWithContext(ctx, method, githubAPIURL+path, reqBody)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("content-type", "application/json")
	return http.DefaultClient.Do(req)
}

// GitHubCreateBranch creates a new branch in a GitHub repository.
type GitHubCreateBranch struct {
	client *githubClient
}

func NewGitHubCreateBranch(token string) *GitHubCreateBranch {
	return &GitHubCreateBranch{client: &githubClient{token: token}}
}

func (t *GitHubCreateBranch) Name() string { return "github_create_branch" }

func (t *GitHubCreateBranch) Description() string {
	return "Create a new branch in a GitHub repository from the default branch."
}

func (t *GitHubCreateBranch) Parameters() llm.ParameterSchema {
	return llm.ParameterSchema{
		Type: "object",
		Properties: map[string]llm.PropertySchema{
			"repo":   {Type: "string", Description: "Repository in owner/repo format, e.g. acme/backend."},
			"branch": {Type: "string", Description: "Name of the branch to create, e.g. feature/JIRA-123-add-login."},
		},
		Required: []string{"repo", "branch"},
	}
}

func (t *GitHubCreateBranch) Execute(ctx context.Context, input map[string]any) (string, error) {
	repo, _ := input["repo"].(string)
	branch, _ := input["branch"].(string)
	if repo == "" || branch == "" {
		return "", fmt.Errorf("repo and branch are required")
	}

	// Get default branch SHA
	resp, err := t.client.do(ctx, http.MethodGet, "/repos/"+repo+"/git/refs/heads", nil)
	if err != nil {
		return "", fmt.Errorf("get refs: %w", err)
	}
	defer resp.Body.Close()

	var refs []struct {
		Ref    string `json:"ref"`
		Object struct {
			SHA string `json:"sha"`
		} `json:"object"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&refs); err != nil {
		return "", fmt.Errorf("decode refs: %w", err)
	}

	// Find main or master
	var sha string
	for _, r := range refs {
		if r.Ref == "refs/heads/main" || r.Ref == "refs/heads/master" {
			sha = r.Object.SHA
			break
		}
	}
	if sha == "" {
		return "", fmt.Errorf("could not find main or master branch in %s", repo)
	}

	// Create branch
	resp2, err := t.client.do(ctx, http.MethodPost, "/repos/"+repo+"/git/refs", map[string]string{
		"ref": "refs/heads/" + branch,
		"sha": sha,
	})
	if err != nil {
		return "", fmt.Errorf("create branch: %w", err)
	}
	defer resp2.Body.Close()
	if resp2.StatusCode >= 300 {
		return "", fmt.Errorf("github returned status %d when creating branch", resp2.StatusCode)
	}
	return fmt.Sprintf("Branch '%s' created in %s.", branch, repo), nil
}

// GitHubCreateIssue creates a new issue in a GitHub repository.
type GitHubCreateIssue struct {
	client *githubClient
}

func NewGitHubCreateIssue(token string) *GitHubCreateIssue {
	return &GitHubCreateIssue{client: &githubClient{token: token}}
}

func (t *GitHubCreateIssue) Name() string { return "github_create_issue" }

func (t *GitHubCreateIssue) Description() string {
	return "Create a new issue in a GitHub repository."
}

func (t *GitHubCreateIssue) Parameters() llm.ParameterSchema {
	return llm.ParameterSchema{
		Type: "object",
		Properties: map[string]llm.PropertySchema{
			"repo":  {Type: "string", Description: "Repository in owner/repo format."},
			"title": {Type: "string", Description: "Issue title."},
			"body":  {Type: "string", Description: "Issue body (markdown supported)."},
		},
		Required: []string{"repo", "title"},
	}
}

func (t *GitHubCreateIssue) Execute(ctx context.Context, input map[string]any) (string, error) {
	repo, _ := input["repo"].(string)
	title, _ := input["title"].(string)
	body, _ := input["body"].(string)
	if repo == "" || title == "" {
		return "", fmt.Errorf("repo and title are required")
	}

	resp, err := t.client.do(ctx, http.MethodPost, "/repos/"+repo+"/issues", map[string]string{
		"title": title,
		"body":  body,
	})
	if err != nil {
		return "", fmt.Errorf("create issue: %w", err)
	}
	defer resp.Body.Close()

	var result struct {
		HTMLURL string `json:"html_url"`
		Number  int    `json:"number"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("decode response: %w", err)
	}
	if resp.StatusCode >= 300 {
		return "", fmt.Errorf("github returned status %d when creating issue", resp.StatusCode)
	}
	return fmt.Sprintf("Issue #%d created: %s", result.Number, result.HTMLURL), nil
}

// GitHubCreateMR creates a pull request in a GitHub repository.
type GitHubCreateMR struct {
	client *githubClient
}

func NewGitHubCreateMR(token string) *GitHubCreateMR {
	return &GitHubCreateMR{client: &githubClient{token: token}}
}

func (t *GitHubCreateMR) Name() string { return "github_create_mr" }

func (t *GitHubCreateMR) Description() string {
	return "Create a pull request (merge request) in a GitHub repository."
}

func (t *GitHubCreateMR) Parameters() llm.ParameterSchema {
	return llm.ParameterSchema{
		Type: "object",
		Properties: map[string]llm.PropertySchema{
			"repo":   {Type: "string", Description: "Repository in owner/repo format."},
			"title":  {Type: "string", Description: "Pull request title."},
			"body":   {Type: "string", Description: "Pull request description (markdown supported)."},
			"head":   {Type: "string", Description: "The branch containing the changes."},
			"base":   {Type: "string", Description: "The branch to merge into (e.g. main)."},
			"draft":  {Type: "string", Description: "Set to 'true' to create a draft PR."},
		},
		Required: []string{"repo", "title", "head", "base"},
	}
}

func (t *GitHubCreateMR) Execute(ctx context.Context, input map[string]any) (string, error) {
	repo, _ := input["repo"].(string)
	title, _ := input["title"].(string)
	body, _ := input["body"].(string)
	head, _ := input["head"].(string)
	base, _ := input["base"].(string)
	draft := input["draft"] == "true"

	if repo == "" || title == "" || head == "" || base == "" {
		return "", fmt.Errorf("repo, title, head, and base are required")
	}

	resp, err := t.client.do(ctx, http.MethodPost, "/repos/"+repo+"/pulls", map[string]any{
		"title": title,
		"body":  body,
		"head":  head,
		"base":  base,
		"draft": draft,
	})
	if err != nil {
		return "", fmt.Errorf("create pull request: %w", err)
	}
	defer resp.Body.Close()

	var result struct {
		HTMLURL string `json:"html_url"`
		Number  int    `json:"number"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("decode response: %w", err)
	}
	if resp.StatusCode >= 300 {
		return "", fmt.Errorf("github returned status %d when creating pull request", resp.StatusCode)
	}
	return fmt.Sprintf("Pull request #%d created: %s", result.Number, result.HTMLURL), nil
}
