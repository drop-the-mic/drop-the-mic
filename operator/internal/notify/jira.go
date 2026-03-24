package notify

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/drop-the-mic/operator/internal/state"
)

// JiraNotifier creates Jira tickets for check failures.
type JiraNotifier struct {
	url       string
	email     string
	token     string
	project   string
	issueType string
	client    *http.Client
}

// NewJiraNotifier creates a new Jira notifier.
func NewJiraNotifier(url, email, token, project, issueType string) *JiraNotifier {
	if issueType == "" {
		issueType = "Bug"
	}
	return &JiraNotifier{
		url:       url,
		email:     email,
		token:     token,
		project:   project,
		issueType: issueType,
		client:    &http.Client{},
	}
}

func (j *JiraNotifier) Name() string { return "jira" }

func (j *JiraNotifier) Send(ctx context.Context, event Event) error {
	summary := fmt.Sprintf("[DTM] %s - Check %s: %s",
		event.PolicyRef, event.CheckID, event.Transition.To)

	if event.Transition.To == state.StateResolved {
		summary = fmt.Sprintf("[DTM] RESOLVED - %s Check %s", event.PolicyRef, event.CheckID)
	}

	description := fmt.Sprintf(
		"Policy: %s\nCheck ID: %s\nSeverity: %s\nVerdict: %s\nState: %s → %s\n\nDescription:\n%s\n\nReasoning:\n%s",
		event.PolicyRef,
		event.CheckID,
		event.Severity,
		event.Verdict,
		event.Transition.From,
		event.Transition.To,
		event.Description,
		event.Reasoning,
	)

	payload := map[string]interface{}{
		"fields": map[string]interface{}{
			"project": map[string]string{
				"key": j.project,
			},
			"summary":     summary,
			"description": description,
			"issuetype": map[string]string{
				"name": j.issueType,
			},
		},
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshaling jira payload: %w", err)
	}

	apiURL := fmt.Sprintf("%s/rest/api/2/issue", j.url)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, apiURL, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("creating jira request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.SetBasicAuth(j.email, j.token)

	resp, err := j.client.Do(req)
	if err != nil {
		return fmt.Errorf("sending jira notification: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		return fmt.Errorf("jira API returned status %d", resp.StatusCode)
	}

	return nil
}
