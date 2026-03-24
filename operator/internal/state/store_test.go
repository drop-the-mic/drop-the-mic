package state

import (
	"testing"
	"time"
)

func TestStore_UnknownToFiring(t *testing.T) {
	s := NewStore(5)

	tr := s.RecordFail("policy-1", "check-1")
	if tr == nil {
		t.Fatal("expected transition, got nil")
	}
	if tr.From != StateUnknown {
		t.Fatalf("expected from UNKNOWN, got %s", tr.From)
	}
	if tr.To != StateFiring {
		t.Fatalf("expected to FIRING, got %s", tr.To)
	}

	cs := s.GetState("policy-1", "check-1")
	if cs == nil {
		t.Fatal("expected state, got nil")
	}
	if cs.State != StateFiring {
		t.Fatalf("expected FIRING, got %s", cs.State)
	}
	if cs.FailedSince.IsZero() {
		t.Fatal("expected failedSince to be set")
	}
}

func TestStore_FiringSuppressed(t *testing.T) {
	s := NewStore(5)

	// First fail → FIRING (transition)
	tr := s.RecordFail("p", "c")
	if tr == nil {
		t.Fatal("expected first transition")
	}

	// Second fail → still FIRING (suppressed)
	tr = s.RecordFail("p", "c")
	if tr != nil {
		t.Fatal("expected suppressed (nil transition) on second fail")
	}

	cs := s.GetState("p", "c")
	if cs.RetryCount != 1 {
		t.Fatalf("expected retryCount=1, got %d", cs.RetryCount)
	}
}

func TestStore_FiringToResolved(t *testing.T) {
	s := NewStore(5)

	s.RecordFail("p", "c")

	tr := s.RecordPass("p", "c")
	if tr == nil {
		t.Fatal("expected transition to RESOLVED")
	}
	if tr.From != StateFiring {
		t.Fatalf("expected from FIRING, got %s", tr.From)
	}
	if tr.To != StateResolved {
		t.Fatalf("expected to RESOLVED, got %s", tr.To)
	}

	cs := s.GetState("p", "c")
	if cs.State != StateResolved {
		t.Fatalf("expected RESOLVED, got %s", cs.State)
	}
	if cs.RetryCount != 0 {
		t.Fatalf("expected retryCount reset to 0, got %d", cs.RetryCount)
	}
}

func TestStore_ResolvedToFiring(t *testing.T) {
	s := NewStore(5)

	s.RecordFail("p", "c")
	s.RecordPass("p", "c") // → RESOLVED

	tr := s.RecordFail("p", "c") // → FIRING again
	if tr == nil {
		t.Fatal("expected transition from RESOLVED to FIRING")
	}
	if tr.From != StateResolved {
		t.Fatalf("expected from RESOLVED, got %s", tr.From)
	}
	if tr.To != StateFiring {
		t.Fatalf("expected to FIRING, got %s", tr.To)
	}
}

func TestStore_Escalation(t *testing.T) {
	s := NewStore(3) // low threshold for testing

	// First fail → FIRING
	s.RecordFail("p", "c")

	// 3 more fails to hit threshold
	s.RecordFail("p", "c")       // retry 1, suppressed
	s.RecordFail("p", "c")       // retry 2, suppressed
	tr := s.RecordFail("p", "c") // retry 3 → ESCALATED
	if tr == nil {
		t.Fatal("expected escalation transition")
	}
	if tr.To != StateEscalated {
		t.Fatalf("expected ESCALATED, got %s", tr.To)
	}

	cs := s.GetState("p", "c")
	if cs.State != StateEscalated {
		t.Fatalf("expected ESCALATED state, got %s", cs.State)
	}
}

func TestStore_EscalatedSuppressed(t *testing.T) {
	s := NewStore(1) // escalate after 1 retry

	s.RecordFail("p", "c") // → FIRING
	s.RecordFail("p", "c") // → ESCALATED

	// Further fails should be suppressed
	tr := s.RecordFail("p", "c")
	if tr != nil {
		t.Fatal("expected suppressed after escalation")
	}
}

func TestStore_EscalatedToResolved(t *testing.T) {
	s := NewStore(1)

	s.RecordFail("p", "c") // → FIRING
	s.RecordFail("p", "c") // → ESCALATED

	tr := s.RecordPass("p", "c")
	if tr == nil {
		t.Fatal("expected transition from ESCALATED to RESOLVED")
	}
	if tr.From != StateEscalated {
		t.Fatalf("expected from ESCALATED, got %s", tr.From)
	}
	if tr.To != StateResolved {
		t.Fatalf("expected to RESOLVED, got %s", tr.To)
	}
}

func TestStore_PassOnUnknown_NoTransition(t *testing.T) {
	s := NewStore(5)

	tr := s.RecordPass("p", "c")
	if tr != nil {
		t.Fatal("expected no transition when passing on UNKNOWN state")
	}
}

func TestStore_FailedChecks(t *testing.T) {
	s := NewStore(5)

	s.RecordFail("p", "c1")
	s.RecordFail("p", "c2")
	s.RecordFail("p", "c3")
	s.RecordPass("p", "c2") // resolve c2

	failed := s.FailedChecks("p")
	if len(failed) != 2 {
		t.Fatalf("expected 2 failed checks, got %d: %v", len(failed), failed)
	}

	failedSet := map[string]bool{}
	for _, id := range failed {
		failedSet[id] = true
	}
	if !failedSet["c1"] || !failedSet["c3"] {
		t.Fatalf("expected c1 and c3, got %v", failed)
	}
}

func TestStore_FailedChecks_EmptyPolicy(t *testing.T) {
	s := NewStore(5)

	failed := s.FailedChecks("nonexistent")
	if len(failed) != 0 {
		t.Fatalf("expected 0 failed checks, got %d", len(failed))
	}
}

func TestStore_RestoreFromFailedSince(t *testing.T) {
	s := NewStore(5)

	failedSince := time.Date(2026, 3, 20, 10, 0, 0, 0, time.UTC)
	s.RestoreFromFailedSince("p", "c", failedSince)

	cs := s.GetState("p", "c")
	if cs == nil {
		t.Fatal("expected state after restore")
	}
	if cs.State != StateFiring {
		t.Fatalf("expected FIRING after restore, got %s", cs.State)
	}
	if !cs.FailedSince.Equal(failedSince) {
		t.Fatalf("expected failedSince %v, got %v", failedSince, cs.FailedSince)
	}
}

func TestStore_GetState_NonExistent(t *testing.T) {
	s := NewStore(5)

	cs := s.GetState("nope", "nope")
	if cs != nil {
		t.Fatal("expected nil for nonexistent state")
	}
}

func TestStore_MultiplePolicies(t *testing.T) {
	s := NewStore(5)

	s.RecordFail("p1", "c1")
	s.RecordFail("p2", "c1")

	f1 := s.FailedChecks("p1")
	f2 := s.FailedChecks("p2")

	if len(f1) != 1 || f1[0] != "c1" {
		t.Fatalf("expected p1 to have 1 failed check, got %v", f1)
	}
	if len(f2) != 1 || f2[0] != "c1" {
		t.Fatalf("expected p2 to have 1 failed check, got %v", f2)
	}

	// Resolve p1, p2 should remain
	s.RecordPass("p1", "c1")
	if len(s.FailedChecks("p1")) != 0 {
		t.Fatal("expected p1 to have 0 failed checks after resolve")
	}
	if len(s.FailedChecks("p2")) != 1 {
		t.Fatal("expected p2 to still have 1 failed check")
	}
}

func TestStore_DefaultEscalationThreshold(t *testing.T) {
	s := NewStore(0) // 0 should default to 5
	if s.escalationThreshold != 5 {
		t.Fatalf("expected default threshold 5, got %d", s.escalationThreshold)
	}

	s = NewStore(-1) // negative should also default
	if s.escalationThreshold != 5 {
		t.Fatalf("expected default threshold 5, got %d", s.escalationThreshold)
	}
}
