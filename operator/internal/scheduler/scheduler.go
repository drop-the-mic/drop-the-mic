package scheduler

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"github.com/go-logr/logr"
	"github.com/robfig/cron/v3"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	dtmv1alpha1 "github.com/drop-the-mic/operator/api/v1alpha1"
	"github.com/drop-the-mic/operator/internal/engine"
	"github.com/drop-the-mic/operator/internal/notify"
	"github.com/drop-the-mic/operator/internal/state"
)

// Scheduler manages the dual-loop scanning for a ChecklistPolicy.
type Scheduler struct {
	policy     *dtmv1alpha1.ChecklistPolicy
	eng        *engine.Engine
	store      *state.Store
	dispatcher *notify.Dispatcher
	k8sClient  client.Client
	log        logr.Logger

	cron          *cron.Cron
	fullScanID    cron.EntryID
	rescanID      cron.EntryID
	mu            sync.Mutex
	cancelFullScan context.CancelFunc
	cancelRescan   context.CancelFunc
}

// New creates a new scheduler for the given policy.
func New(
	policy *dtmv1alpha1.ChecklistPolicy,
	eng *engine.Engine,
	store *state.Store,
	dispatcher *notify.Dispatcher,
	k8sClient client.Client,
	log logr.Logger,
) *Scheduler {
	return &Scheduler{
		policy:     policy,
		eng:        eng,
		store:      store,
		dispatcher: dispatcher,
		k8sClient:  k8sClient,
		log:        log.WithValues("policy", policy.Name, "namespace", policy.Namespace),
		cron:       cron.New(),
	}
}

// Start starts the dual-loop scheduler.
func (s *Scheduler) Start(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	var err error

	// Full Scan loop
	s.fullScanID, err = s.cron.AddFunc(s.policy.Spec.Schedule.FullScan, func() {
		scanCtx, cancel := context.WithCancel(ctx)
		s.mu.Lock()
		s.cancelFullScan = cancel
		s.mu.Unlock()
		defer cancel()
		s.runFullScan(scanCtx)
	})
	if err != nil {
		return fmt.Errorf("adding full scan cron: %w", err)
	}

	// Failed Rescan loop (independent goroutine)
	if s.policy.Spec.Schedule.FailedRescan != "" {
		s.rescanID, err = s.cron.AddFunc(s.policy.Spec.Schedule.FailedRescan, func() {
			scanCtx, cancel := context.WithCancel(ctx)
			s.mu.Lock()
			s.cancelRescan = cancel
			s.mu.Unlock()
			defer cancel()
			s.runRescan(scanCtx)
		})
		if err != nil {
			return fmt.Errorf("adding rescan cron: %w", err)
		}
	}

	s.cron.Start()
	s.log.Info("scheduler started",
		"fullScan", s.policy.Spec.Schedule.FullScan,
		"failedRescan", s.policy.Spec.Schedule.FailedRescan)
	return nil
}

// Stop stops the scheduler.
func (s *Scheduler) Stop() {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.cancelFullScan != nil {
		s.cancelFullScan()
	}
	if s.cancelRescan != nil {
		s.cancelRescan()
	}
	s.cron.Stop()
	s.log.Info("scheduler stopped")
}

// TriggerFullScan runs a full scan immediately (used by Run Now).
func (s *Scheduler) TriggerFullScan(ctx context.Context) {
	go s.runFullScan(ctx)
}

func (s *Scheduler) runFullScan(ctx context.Context) {
	s.log.Info("starting full scan")

	resultSpec, err := s.eng.RunChecks(ctx, s.policy, s.policy.Spec.Checks)
	if err != nil {
		s.log.Error(err, "full scan failed")
		return
	}

	resultSpec.ScanType = dtmv1alpha1.ScanTypeFull

	// Process results through state machine and notify
	s.processResults(ctx, resultSpec)

	// Create ChecklistResult CR
	if err := s.createResult(ctx, resultSpec); err != nil {
		s.log.Error(err, "failed to create ChecklistResult")
		return
	}

	// Update policy status
	if err := s.updatePolicyStatus(ctx, resultSpec); err != nil {
		s.log.Error(err, "failed to update policy status")
	}

	s.log.Info("full scan completed",
		"pass", resultSpec.Summary.Pass,
		"warn", resultSpec.Summary.Warn,
		"fail", resultSpec.Summary.Fail)
}

func (s *Scheduler) runRescan(ctx context.Context) {
	failedIDs := s.store.FailedChecks(s.policy.Name)
	if len(failedIDs) == 0 {
		s.log.V(1).Info("no failed checks to rescan")
		return
	}

	s.log.Info("starting rescan", "failedChecks", len(failedIDs))

	// Filter to only failed checks
	failedSet := make(map[string]bool, len(failedIDs))
	for _, id := range failedIDs {
		failedSet[id] = true
	}

	var checksToRescan []dtmv1alpha1.CheckItem
	for _, check := range s.policy.Spec.Checks {
		if failedSet[check.ID] {
			checksToRescan = append(checksToRescan, check)
		}
	}

	resultSpec, err := s.eng.RunChecks(ctx, s.policy, checksToRescan)
	if err != nil {
		s.log.Error(err, "rescan failed")
		return
	}

	resultSpec.ScanType = dtmv1alpha1.ScanTypeRescan

	// Process results through state machine and notify
	s.processResults(ctx, resultSpec)

	// Create ChecklistResult CR
	if err := s.createResult(ctx, resultSpec); err != nil {
		s.log.Error(err, "failed to create ChecklistResult")
	}

	s.log.Info("rescan completed",
		"pass", resultSpec.Summary.Pass,
		"fail", resultSpec.Summary.Fail)
}

func (s *Scheduler) processResults(ctx context.Context, resultSpec *dtmv1alpha1.ChecklistResultSpec) {
	for i := range resultSpec.Checks {
		check := &resultSpec.Checks[i]
		var transition *state.Transition

		switch check.Verdict {
		case dtmv1alpha1.VerdictFail:
			transition = s.store.RecordFail(s.policy.Name, check.ID)
			// Set failedSince from state store
			cs := s.store.GetState(s.policy.Name, check.ID)
			if cs != nil {
				t := metav1.NewTime(cs.FailedSince)
				check.FailedSince = &t
			}
		case dtmv1alpha1.VerdictPass:
			transition = s.store.RecordPass(s.policy.Name, check.ID)
		}

		if transition != nil {
			event := notify.Event{
				PolicyRef:   s.policy.Name,
				CheckID:     check.ID,
				Description: check.Description,
				Severity:    check.Severity,
				Verdict:     string(check.Verdict),
				Reasoning:   check.Reasoning,
				Transition:  *transition,
			}
			if errs := s.dispatcher.Dispatch(ctx, event); len(errs) > 0 {
				for _, err := range errs {
					s.log.Error(err, "notification failed", "checkID", check.ID)
				}
			}
		}
	}
}

func (s *Scheduler) createResult(ctx context.Context, resultSpec *dtmv1alpha1.ChecklistResultSpec) error {
	resultName := engine.GenerateResultName(s.policy.Name, resultSpec.ScanType)
	resultName = strings.ToLower(resultName)

	result := &dtmv1alpha1.ChecklistResult{
		ObjectMeta: metav1.ObjectMeta{
			Name:      resultName,
			Namespace: s.policy.Namespace,
			Labels: map[string]string{
				"dtm.io/policy": s.policy.Name,
			},
			OwnerReferences: []metav1.OwnerReference{
				{
					APIVersion: dtmv1alpha1.GroupVersion.String(),
					Kind:       "ChecklistPolicy",
					Name:       s.policy.Name,
					UID:        s.policy.UID,
				},
			},
		},
		Spec: *resultSpec,
	}

	result.Status.Phase = "Completed"

	if err := s.k8sClient.Create(ctx, result); err != nil {
		return fmt.Errorf("creating ChecklistResult: %w", err)
	}

	return nil
}

func (s *Scheduler) updatePolicyStatus(ctx context.Context, resultSpec *dtmv1alpha1.ChecklistResultSpec) error {
	policy := s.policy.DeepCopy()
	now := metav1.Now()

	if resultSpec.ScanType == dtmv1alpha1.ScanTypeFull {
		policy.Status.LastFullScanTime = &now
	} else {
		policy.Status.LastRescanTime = &now
	}

	policy.Status.Summary = resultSpec.Summary
	policy.Status.LastResultName = engine.GenerateResultName(s.policy.Name, resultSpec.ScanType)

	return s.k8sClient.Status().Update(ctx, policy)
}
