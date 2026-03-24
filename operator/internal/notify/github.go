package notify

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/drop-the-mic/operator/internal/state"
)

// GitHubNotifier creates GitHub issues for check failures.
type GitHubNotifier struct {
	token  string
	owner  string
	repo   string
	labels []string
	client *http.Client
}

// NewGitHubNotifier creates a new GitHub notifier.
func NewGitHubNotifier(token, owner, repo string, labels []string) *GitHubNotifier {
	return &GitHubNotifier{
		token:  token,
		owner:  owner,
		repo:   repo,
		labels: labels,
		client: &http.Client{},
	}
}

func (g *GitHubNotifier) Name() string { return "github" }

func (g *GitHubNotifier) Send(ctx context.Context, event Event) error {
	title := fmt.Sprintf("[DTM] %s - Check %s: %s",
		event.PolicyRef, event.CheckID, event.Transition.To)

	body := fmt.Sprintf(`## DTM Check Alert

**Policy:** %s
**Check ID:** %s
**Severity:** %s
**Verdict:** %s
**State:** %s → %s

### Description
%s

### Reasoning
%s
`,
		event.PolicyRef,
		event.CheckID,
		event.Severity,
		event.Verdict,
		event.Transition.From,
		event.Transition.To,
		event.Description,
		event.Reasoning,
	)

	if event.Transition.To == state.StateResolved {
		title = fmt.Sprintf("[DTM] RESOLVED - %s Check %s", event.PolicyRef, event.CheckID)
	}

	payload := map[string]interface{}{
		"title":  title,
		"body":   body,
		"labels": g.labels,
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshaling github payload: %w", err)
	}

	url := fmt.Sprintf("https://api.github.com/repos/%s/%s/issues", g.owner, g.repo)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(payloadBytes))
	if err != nil {
		return fmt.Errorf("creating github request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "token "+g.token)
	req.Header.Set("Accept", "application/vnd.github.v3+json")

	resp, err := g.client.Do(req)
	if err != nil {
		return fmt.Errorf("sending github notification: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		return fmt.Errorf("github API returned status %d", resp.StatusCode)
	}

	return nil
}
