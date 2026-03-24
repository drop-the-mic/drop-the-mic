/*
Copyright 2026.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controller

import (
	"context"
	"fmt"
	"sync"
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	dtmv1alpha1 "github.com/drop-the-mic/operator/api/v1alpha1"
	"github.com/drop-the-mic/operator/internal/engine"
	"github.com/drop-the-mic/operator/internal/engine/llm"
	"github.com/drop-the-mic/operator/internal/engine/tool"
	"github.com/drop-the-mic/operator/internal/notify"
	"github.com/drop-the-mic/operator/internal/scheduler"
	"github.com/drop-the-mic/operator/internal/state"
)

const (
	finalizerName    = "dtm.io/finalizer"
	runNowAnnotation = "dtm.io/run-now"
)

// ChecklistPolicyReconciler reconciles a ChecklistPolicy object.
type ChecklistPolicyReconciler struct {
	client.Client
	Scheme     *runtime.Scheme
	KubeClient kubernetes.Interface
	Store      *state.Store

	mu         sync.Mutex
	schedulers map[string]*scheduler.Scheduler
}

// +kubebuilder:rbac:groups=dtm.dtm.io,resources=checklistpolicies,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=dtm.dtm.io,resources=checklistpolicies/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=dtm.dtm.io,resources=checklistpolicies/finalizers,verbs=update
// +kubebuilder:rbac:groups=dtm.dtm.io,resources=checklistresults,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups="",resources=pods;pods/log;nodes;events;services;configmaps;secrets,verbs=get;list;watch
// +kubebuilder:rbac:groups=policy,resources=poddisruptionbudgets,verbs=get;list;watch
// +kubebuilder:rbac:groups=autoscaling,resources=horizontalpodautoscalers,verbs=get;list;watch
// +kubebuilder:rbac:groups=apps,resources=deployments;replicasets;statefulsets;daemonsets,verbs=get;list;watch

func (r *ChecklistPolicyReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := logf.FromContext(ctx)

	var policy dtmv1alpha1.ChecklistPolicy
	if err := r.Get(ctx, req.NamespacedName, &policy); err != nil {
		if errors.IsNotFound(err) {
			// Policy deleted — stop scheduler
			r.stopScheduler(req.NamespacedName.String())
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	// Handle deletion with finalizer
	if !policy.DeletionTimestamp.IsZero() {
		if controllerutil.ContainsFinalizer(&policy, finalizerName) {
			r.stopScheduler(req.NamespacedName.String())
			controllerutil.RemoveFinalizer(&policy, finalizerName)
			if err := r.Update(ctx, &policy); err != nil {
				return ctrl.Result{}, err
			}
		}
		return ctrl.Result{}, nil
	}

	// Add finalizer if not present
	if !controllerutil.ContainsFinalizer(&policy, finalizerName) {
		controllerutil.AddFinalizer(&policy, finalizerName)
		if err := r.Update(ctx, &policy); err != nil {
			return ctrl.Result{}, err
		}
		return ctrl.Result{Requeue: true}, nil
	}

	// Handle "Run Now" annotation
	if ts, ok := policy.Annotations[runNowAnnotation]; ok && ts != "" {
		log.Info("run-now triggered", "timestamp", ts)
		sched := r.getScheduler(req.NamespacedName.String())
		if sched != nil {
			sched.TriggerFullScan(ctx)
		}
		// Remove the annotation
		delete(policy.Annotations, runNowAnnotation)
		if err := r.Update(ctx, &policy); err != nil {
			return ctrl.Result{}, err
		}
		return ctrl.Result{}, nil
	}

	// Build or rebuild scheduler
	if err := r.reconcileScheduler(ctx, &policy); err != nil {
		log.Error(err, "failed to reconcile scheduler")
		// Set degraded condition
		meta.SetStatusCondition(&policy.Status.Conditions, metav1.Condition{
			Type:               "Degraded",
			Status:             metav1.ConditionTrue,
			Reason:             "SchedulerError",
			Message:            err.Error(),
			LastTransitionTime: metav1.Now(),
		})
		if statusErr := r.Status().Update(ctx, &policy); statusErr != nil {
			log.Error(statusErr, "failed to update status")
		}
		return ctrl.Result{RequeueAfter: 30 * time.Second}, err
	}

	// Set available condition
	meta.SetStatusCondition(&policy.Status.Conditions, metav1.Condition{
		Type:               "Available",
		Status:             metav1.ConditionTrue,
		Reason:             "SchedulerRunning",
		Message:            "Scheduler is running",
		LastTransitionTime: metav1.Now(),
	})
	if err := r.Status().Update(ctx, &policy); err != nil {
		log.Error(err, "failed to update status")
		return ctrl.Result{RequeueAfter: 10 * time.Second}, nil
	}

	return ctrl.Result{}, nil
}

func (r *ChecklistPolicyReconciler) reconcileScheduler(ctx context.Context, policy *dtmv1alpha1.ChecklistPolicy) error {
	key := fmt.Sprintf("%s/%s", policy.Namespace, policy.Name)

	// Stop existing scheduler
	r.stopScheduler(key)

	// Resolve LLM API key from secret
	apiKey, err := r.resolveSecret(ctx, policy.Namespace, policy.Spec.LLM.SecretRef)
	if err != nil {
		return fmt.Errorf("resolving LLM secret: %w", err)
	}

	// Build LLM adapter
	adapter, err := r.buildAdapter(policy.Spec.LLM, apiKey)
	if err != nil {
		return fmt.Errorf("building LLM adapter: %w", err)
	}

	// Build tool registry
	registry := tool.NewRegistry()
	tool.RegisterPods(registry, r.KubeClient)
	tool.RegisterNodes(registry, r.KubeClient)
	tool.RegisterEvents(registry, r.KubeClient)
	tool.RegisterPDB(registry, r.KubeClient)
	tool.RegisterHPA(registry, r.KubeClient)
	tool.RegisterImages(registry, r.KubeClient)
	tool.RegisterLogs(registry, r.KubeClient)

	// Build engine
	log := logf.FromContext(ctx)
	eng := engine.New(adapter, registry, log)

	// Build notifiers
	dispatcher, err := r.buildDispatcher(ctx, policy)
	if err != nil {
		return fmt.Errorf("building dispatcher: %w", err)
	}

	// Create and start scheduler
	sched := scheduler.New(policy, eng, r.Store, dispatcher, r.Client, log)
	if err := sched.Start(ctx); err != nil {
		return fmt.Errorf("starting scheduler: %w", err)
	}

	r.mu.Lock()
	if r.schedulers == nil {
		r.schedulers = make(map[string]*scheduler.Scheduler)
	}
	r.schedulers[key] = sched
	r.mu.Unlock()

	return nil
}

func (r *ChecklistPolicyReconciler) stopScheduler(key string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if sched, ok := r.schedulers[key]; ok {
		sched.Stop()
		delete(r.schedulers, key)
	}
}

func (r *ChecklistPolicyReconciler) getScheduler(key string) *scheduler.Scheduler {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.schedulers[key]
}

func (r *ChecklistPolicyReconciler) resolveSecret(ctx context.Context, namespace string, ref dtmv1alpha1.SecretReference) (string, error) {
	var secret corev1.Secret
	if err := r.Get(ctx, client.ObjectKey{Namespace: namespace, Name: ref.Name}, &secret); err != nil {
		return "", fmt.Errorf("getting secret %s: %w", ref.Name, err)
	}

	key := ref.Key
	if key == "" {
		key = "api-key"
	}

	data, ok := secret.Data[key]
	if !ok {
		return "", fmt.Errorf("key %q not found in secret %s", key, ref.Name)
	}

	return string(data), nil
}

func (r *ChecklistPolicyReconciler) buildAdapter(cfg dtmv1alpha1.LLMConfig, apiKey string) (llm.Adapter, error) {
	switch cfg.Provider {
	case "claude":
		return llm.NewClaudeAdapter(apiKey, cfg.Model), nil
	default:
		return nil, fmt.Errorf("unsupported LLM provider: %s", cfg.Provider)
	}
}

func (r *ChecklistPolicyReconciler) buildDispatcher(ctx context.Context, policy *dtmv1alpha1.ChecklistPolicy) (*notify.Dispatcher, error) {
	if policy.Spec.Notification == nil {
		return notify.NewDispatcher(), nil
	}

	var notifiers []notify.Notifier

	if slack := policy.Spec.Notification.Slack; slack != nil {
		webhookURL, err := r.resolveSecret(ctx, policy.Namespace, slack.SecretRef)
		if err != nil {
			return nil, fmt.Errorf("resolving slack secret: %w", err)
		}
		notifiers = append(notifiers, notify.NewSlackNotifier(webhookURL, slack.Channel))
	}

	if gh := policy.Spec.Notification.GitHub; gh != nil {
		token, err := r.resolveSecret(ctx, policy.Namespace, gh.SecretRef)
		if err != nil {
			return nil, fmt.Errorf("resolving github secret: %w", err)
		}
		notifiers = append(notifiers, notify.NewGitHubNotifier(token, gh.Owner, gh.Repo, gh.Labels))
	}

	if jira := policy.Spec.Notification.Jira; jira != nil {
		token, err := r.resolveSecret(ctx, policy.Namespace, jira.SecretRef)
		if err != nil {
			return nil, fmt.Errorf("resolving jira secret: %w", err)
		}
		notifiers = append(notifiers, notify.NewJiraNotifier(jira.URL, "", token, jira.Project, jira.IssueType))
	}

	return notify.NewDispatcher(notifiers...), nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *ChecklistPolicyReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&dtmv1alpha1.ChecklistPolicy{}).
		Owns(&dtmv1alpha1.ChecklistResult{}).
		Named("checklistpolicy").
		Complete(r)
}
