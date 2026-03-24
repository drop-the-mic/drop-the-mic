// Package state manages the alert state machine for each check across policies.
// States follow the lifecycle: UNKNOWN → FIRING → RESOLVED (or ESCALATED).
package state

import (
	"sync"
	"time"
)

// AlertState represents a check's position in the alert lifecycle.
type AlertState string

const (
	StateUnknown   AlertState = "UNKNOWN"
	StateFiring    AlertState = "FIRING"
	StateResolved  AlertState = "RESOLVED"
	StateEscalated AlertState = "ESCALATED"
)

// CheckState holds the current alert state and metadata for a single check.
type CheckState struct {
	State       AlertState
	FailedSince time.Time
	RetryCount  int32
	LastUpdated time.Time
}

// Transition is emitted when a check moves between alert states,
// signaling that a notification should be sent.
type Transition struct {
	CheckID   string
	PolicyRef string
	From      AlertState
	To        AlertState
	State     CheckState
}

// Store manages the alert state machine for all checks across all policies.
// It is safe for concurrent use from multiple goroutines.
type Store struct {
	mu                  sync.RWMutex
	checks              map[string]map[string]*CheckState // policyRef → checkID → state
	escalationThreshold int32
}

// NewStore creates a new state store.
func NewStore(escalationThreshold int32) *Store {
	if escalationThreshold <= 0 {
		escalationThreshold = 5
	}
	return &Store{
		checks:              make(map[string]map[string]*CheckState),
		escalationThreshold: escalationThreshold,
	}
}

// RecordFail records a failure for a check and returns any state transition.
func (s *Store) RecordFail(policyRef, checkID string) *Transition {
	s.mu.Lock()
	defer s.mu.Unlock()

	cs := s.getOrCreate(policyRef, checkID)
	now := time.Now()
	prev := cs.State

	switch cs.State {
	case StateUnknown, StateResolved:
		cs.State = StateFiring
		cs.FailedSince = now
		cs.RetryCount = 0
		cs.LastUpdated = now
		return &Transition{
			CheckID:   checkID,
			PolicyRef: policyRef,
			From:      prev,
			To:        StateFiring,
			State:     *cs,
		}

	case StateFiring:
		cs.RetryCount++
		cs.LastUpdated = now
		if cs.RetryCount >= s.escalationThreshold {
			cs.State = StateEscalated
			return &Transition{
				CheckID:   checkID,
				PolicyRef: policyRef,
				From:      StateFiring,
				To:        StateEscalated,
				State:     *cs,
			}
		}
		// Suppress notification — still firing
		return nil

	case StateEscalated:
		cs.RetryCount++
		cs.LastUpdated = now
		return nil
	}

	return nil
}

// RecordPass records a pass for a check and returns any state transition.
func (s *Store) RecordPass(policyRef, checkID string) *Transition {
	s.mu.Lock()
	defer s.mu.Unlock()

	cs := s.getOrCreate(policyRef, checkID)
	prev := cs.State

	switch cs.State {
	case StateFiring, StateEscalated:
		cs.State = StateResolved
		cs.RetryCount = 0
		cs.LastUpdated = time.Now()
		return &Transition{
			CheckID:   checkID,
			PolicyRef: policyRef,
			From:      prev,
			To:        StateResolved,
			State:     *cs,
		}
	default:
		return nil
	}
}

// FailedChecks returns the IDs of all currently failing checks for a policy.
func (s *Store) FailedChecks(policyRef string) []string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	policyChecks, ok := s.checks[policyRef]
	if !ok {
		return nil
	}

	var failed []string
	for id, cs := range policyChecks {
		if cs.State == StateFiring || cs.State == StateEscalated {
			failed = append(failed, id)
		}
	}
	return failed
}

// GetState returns the current state for a specific check.
func (s *Store) GetState(policyRef, checkID string) *CheckState {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if policyChecks, ok := s.checks[policyRef]; ok {
		if cs, ok := policyChecks[checkID]; ok {
			copy := *cs
			return &copy
		}
	}
	return nil
}

// RestoreFromFailedSince restores state from a ChecklistResult's failedSince field.
func (s *Store) RestoreFromFailedSince(policyRef, checkID string, failedSince time.Time) {
	s.mu.Lock()
	defer s.mu.Unlock()

	cs := s.getOrCreate(policyRef, checkID)
	cs.State = StateFiring
	cs.FailedSince = failedSince
	cs.LastUpdated = time.Now()
}

func (s *Store) getOrCreate(policyRef, checkID string) *CheckState {
	if _, ok := s.checks[policyRef]; !ok {
		s.checks[policyRef] = make(map[string]*CheckState)
	}
	if _, ok := s.checks[policyRef][checkID]; !ok {
		s.checks[policyRef][checkID] = &CheckState{
			State: StateUnknown,
		}
	}
	return s.checks[policyRef][checkID]
}
