package notify

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/drop-the-mic/operator/internal/state"
)

// SlackNotifier sends notifications to Slack via webhook.
type SlackNotifier struct {
	webhookURL string
	channel    string
	client     *http.Client
}

// NewSlackNotifier creates a new Slack notifier.
func NewSlackNotifier(webhookURL, channel string) *SlackNotifier {
	return &SlackNotifier{
		webhookURL: webhookURL,
		channel:    channel,
		client:     &http.Client{},
	}
}

func (s *SlackNotifier) Name() string { return "slack" }

func (s *SlackNotifier) Send(ctx context.Context, event Event) error {
	emoji := emojiForTransition(event.Transition.To)
	color := colorForTransition(event.Transition.To)

	payload := map[string]interface{}{
		"channel": s.channel,
		"attachments": []map[string]interface{}{
			{
				"color": color,
				"blocks": []map[string]interface{}{
					{
						"type": "header",
						"text": map[string]string{
							"type": "plain_text",
							"text": fmt.Sprintf("%s DTM Check %s", emoji, event.Transition.To),
						},
					},
					{
						"type": "section",
						"fields": []map[string]string{
							{"type": "mrkdwn", "text": fmt.Sprintf("*Policy:*\n%s", event.PolicyRef)},
							{"type": "mrkdwn", "text": fmt.Sprintf("*Check:*\n%s", event.CheckID)},
							{"type": "mrkdwn", "text": fmt.Sprintf("*Severity:*\n%s", event.Severity)},
							{"type": "mrkdwn", "text": fmt.Sprintf("*Verdict:*\n%s", event.Verdict)},
						},
					},
					{
						"type": "section",
						"text": map[string]string{
							"type": "mrkdwn",
							"text": fmt.Sprintf("*Description:*\n%s", event.Description),
						},
					},
					{
						"type": "section",
						"text": map[string]string{
							"type": "mrkdwn",
							"text": fmt.Sprintf("*Reasoning:*\n%.500s", event.Reasoning),
						},
					},
				},
			},
		},
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshaling slack payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.webhookURL, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("creating slack request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.client.Do(req)
	if err != nil {
		return fmt.Errorf("sending slack notification: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("slack API returned status %d", resp.StatusCode)
	}

	return nil
}

func emojiForTransition(to state.AlertState) string {
	switch to {
	case state.StateFiring:
		return "🔴"
	case state.StateResolved:
		return "✅"
	case state.StateEscalated:
		return "🚨"
	default:
		return "ℹ️"
	}
}

func colorForTransition(to state.AlertState) string {
	switch to {
	case state.StateFiring:
		return "#ff0000"
	case state.StateResolved:
		return "#36a64f"
	case state.StateEscalated:
		return "#ff6600"
	default:
		return "#cccccc"
	}
}
