package setup

import (
	"customclaw/internal/config"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
)

// allTools is the canonical list of tools the registry supports.
var allTools = []string{
	"notify_discord",
	"notify_google_chat",
	"github_create_branch",
	"github_create_issue",
	"github_create_mr",
	"jira_get_ticket",
	"llm_check_description",
}

// configureActions runs the actions.json setup wizard.
// It loads an existing actions.json (if any) and uses its values as defaults.
func (w *Wizard) configureActions(actionsPath string) error {
	existing, err := config.LoadActions(actionsPath)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("load existing actions: %w", err)
	}
	if existing == nil {
		existing = &config.Actions{}
	}

	actions := *existing

	if err := w.configureTools(&actions); err != nil {
		return w.interruptedOrErr(err)
	}
	if err := w.configureWorkflows(&actions); err != nil {
		return w.interruptedOrErr(err)
	}

	data, err := json.MarshalIndent(actions, "", "  ")
	if err != nil {
		return fmt.Errorf("encode actions: %w", err)
	}
	if err := os.WriteFile(actionsPath, data, 0644); err != nil {
		return fmt.Errorf("write %s: %w", actionsPath, err)
	}

	fmt.Fprintf(w.out, "Actions saved to %s\n", actionsPath)
	return nil
}

// configureTools shows a checkbox list of all available tools.
func (w *Wizard) configureTools(actions *config.Actions) error {
	if w.err != nil {
		return w.err
	}

	w.section("Tools")
	fmt.Fprintln(w.out, "Select which tools the agent is allowed to use.")
	fmt.Fprintln(w.out)

	preSelected := actions.Tools
	if len(preSelected) == 0 {
		preSelected = allTools // default: all enabled
	}

	selected, err := interactiveMultiSelect(allTools, preSelected)
	if err != nil {
		w.err = err
		return err
	}

	actions.Tools = selected
	fmt.Fprintf(w.out, "  %d tool(s) enabled\n", len(selected))
	return nil
}

// configureWorkflows lets the user add / review workflows one by one.
func (w *Wizard) configureWorkflows(actions *config.Actions) error {
	if w.err != nil {
		return w.err
	}

	w.section("Workflows")

	if len(actions.Workflows) > 0 {
		fmt.Fprintf(w.out, "Existing workflows (%d):\n", len(actions.Workflows))
		for _, wf := range actions.Workflows {
			fmt.Fprintf(w.out, "  - %s (%s %s)\n", wf.Name, wf.Trigger.Service, wf.Trigger.Event)
		}
		fmt.Fprintln(w.out)
	}

	for {
		label := "Add a workflow"
		if len(actions.Workflows) > 0 {
			label = "Add another workflow"
		}
		if !w.confirm(label, len(actions.Workflows) == 0) {
			break
		}
		if w.err != nil {
			return w.err
		}

		wf, err := w.buildWorkflow()
		if err != nil {
			return w.interruptedOrErr(err)
		}
		actions.Workflows = append(actions.Workflows, *wf)
		fmt.Fprintf(w.out, "  Workflow '%s' added.\n\n", wf.Name)
	}

	return w.err
}

// buildWorkflow prompts for a single workflow definition.
func (w *Wizard) buildWorkflow() (*config.Workflow, error) {
	fmt.Fprintln(w.out)

	name := w.prompt("  Workflow name", "", "e.g. jira-ticket-created")
	if w.err != nil {
		return nil, w.err
	}

	triggerTypes := []string{"webhook", "cli"}
	fmt.Fprintf(w.out, "  Trigger type:\n")
	triggerIdx, err := interactiveSelect(triggerTypes, 0)
	if err != nil {
		w.err = err
		return nil, err
	}
	triggerType := triggerIdx

	wf := &config.Workflow{
		Name: name,
		Trigger: config.WorkflowTrigger{
			Type: triggerType,
		},
	}

	if triggerType == "webhook" {
		services := []string{"jira", "github", "gitlab", "clickup"}
		fmt.Fprintf(w.out, "  Service:\n")
		service, err := interactiveSelect(services, 0)
		if err != nil {
			w.err = err
			return nil, err
		}
		wf.Trigger.Service = service

		event := w.prompt("  Event", defaultEventFor(service), "e.g. issue_created")
		if w.err != nil {
			return nil, w.err
		}
		wf.Trigger.Event = event

		defaultPath := fmt.Sprintf("/webhook/%s", service)
		wf.Trigger.Path = w.prompt("  Webhook path", defaultPath, "")
		if w.err != nil {
			return nil, w.err
		}
	}

	goal := w.prompt("  Goal (what should the agent do?)", "", "describe the automation")
	if w.err != nil {
		return nil, w.err
	}
	wf.Goal = goal

	return wf, nil
}

func defaultEventFor(service string) string {
	switch service {
	case "jira":
		return "issue_created"
	case "github":
		return "pull_request.opened"
	case "gitlab":
		return "merge_request.opened"
	case "clickup":
		return "task_created"
	default:
		return ""
	}
}

// humanizeTrigger returns a short description of the trigger for display.
func humanizeTrigger(t config.WorkflowTrigger) string {
	if t.Type == "cli" {
		return "cli"
	}
	parts := []string{t.Service, t.Event}
	return strings.Join(parts, "/")
}
