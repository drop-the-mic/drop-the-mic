package notify

import (
	"context"
	"fmt"
	"testing"

	"github.com/drop-the-mic/operator/internal/state"
)

type mockNotifier struct {
	name   string
	events []Event
	err    error
}

func (m *mockNotifier) Send(ctx context.Context, event Event) error {
	m.events = append(m.events, event)
	return m.err
}

func (m *mockNotifier) Name() string { return m.name }

var _ Notifier = (*mockNotifier)(nil)

func makeEvent(checkID string, to state.AlertState) Event {
	return Event{
		PolicyRef:   "test-policy",
		CheckID:     checkID,
		Description: "test check",
		Severity:    "critical",
		Verdict:     "FAIL",
		Reasoning:   "something is wrong",
		Transition: state.Transition{
			CheckID:   checkID,
			PolicyRef: "test-policy",
			From:      state.StateUnknown,
			To:        to,
		},
	}
}

func TestDispatcher_SingleNotifier(t *testing.T) {
	n := &mockNotifier{name: "mock"}
	d := NewDispatcher(n)

	event := makeEvent("c1", state.StateFiring)
	errs := d.Dispatch(context.Background(), event)
	if len(errs) != 0 {
		t.Fatalf("expected no errors, got %v", errs)
	}
	if len(n.events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(n.events))
	}
	if n.events[0].CheckID != "c1" {
		t.Fatalf("expected checkID=c1, got %s", n.events[0].CheckID)
	}
}

func TestDispatcher_MultipleNotifiers(t *testing.T) {
	n1 := &mockNotifier{name: "slack"}
	n2 := &mockNotifier{name: "github"}
	n3 := &mockNotifier{name: "jira"}
	d := NewDispatcher(n1, n2, n3)

	event := makeEvent("c1", state.StateFiring)
	errs := d.Dispatch(context.Background(), event)
	if len(errs) != 0 {
		t.Fatalf("expected no errors, got %v", errs)
	}

	for _, n := range []*mockNotifier{n1, n2, n3} {
		if len(n.events) != 1 {
			t.Fatalf("notifier %s: expected 1 event, got %d", n.name, len(n.events))
		}
	}
}

func TestDispatcher_PartialFailure(t *testing.T) {
	n1 := &mockNotifier{name: "slack"}
	n2 := &mockNotifier{name: "github", err: fmt.Errorf("github API error")}
	n3 := &mockNotifier{name: "jira"}
	d := NewDispatcher(n1, n2, n3)

	event := makeEvent("c1", state.StateFiring)
	errs := d.Dispatch(context.Background(), event)

	// Should have 1 error from github
	if len(errs) != 1 {
		t.Fatalf("expected 1 error, got %d", len(errs))
	}

	// But all notifiers should have been attempted
	if len(n1.events) != 1 {
		t.Fatal("slack should have received event")
	}
	if len(n2.events) != 1 {
		t.Fatal("github should have received event (even though it errored)")
	}
	if len(n3.events) != 1 {
		t.Fatal("jira should have received event")
	}
}

func TestDispatcher_NoNotifiers(t *testing.T) {
	d := NewDispatcher()

	event := makeEvent("c1", state.StateFiring)
	errs := d.Dispatch(context.Background(), event)
	if len(errs) != 0 {
		t.Fatalf("expected no errors with empty dispatcher, got %v", errs)
	}
}

func TestDispatcher_AllFail(t *testing.T) {
	n1 := &mockNotifier{name: "slack", err: fmt.Errorf("slack down")}
	n2 := &mockNotifier{name: "github", err: fmt.Errorf("github down")}
	d := NewDispatcher(n1, n2)

	event := makeEvent("c1", state.StateFiring)
	errs := d.Dispatch(context.Background(), event)
	if len(errs) != 2 {
		t.Fatalf("expected 2 errors, got %d", len(errs))
	}
}

func TestDispatcher_EventFields(t *testing.T) {
	n := &mockNotifier{name: "test"}
	d := NewDispatcher(n)

	event := Event{
		PolicyRef:   "my-policy",
		CheckID:     "check-42",
		Description: "verify all pods healthy",
		Severity:    "warning",
		Verdict:     "WARN",
		Reasoning:   "high memory usage",
		Transition: state.Transition{
			CheckID:   "check-42",
			PolicyRef: "my-policy",
			From:      state.StateFiring,
			To:        state.StateResolved,
		},
	}

	d.Dispatch(context.Background(), event)

	got := n.events[0]
	if got.PolicyRef != "my-policy" {
		t.Fatalf("expected policyRef=my-policy, got %s", got.PolicyRef)
	}
	if got.Transition.To != state.StateResolved {
		t.Fatalf("expected transition to RESOLVED, got %s", got.Transition.To)
	}
	if got.Verdict != "WARN" {
		t.Fatalf("expected verdict=WARN, got %s", got.Verdict)
	}
}
