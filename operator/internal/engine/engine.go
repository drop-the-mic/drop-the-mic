package engine

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/go-logr/logr"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"

	dtmv1alpha1 "github.com/drop-the-mic/operator/api/v1alpha1"
	"github.com/drop-the-mic/operator/internal/engine/llm"
	"github.com/drop-the-mic/operator/internal/engine/tool"
)

// Engine orchestrates the verification of checks using an LLM and tools.
type Engine struct {
	adapter  llm.Adapter
	registry *tool.Registry
	log      logr.Logger
}

// New creates a new verification engine.
func New(adapter llm.Adapter, registry *tool.Registry, log logr.Logger) *Engine {
	return &Engine{
		adapter:  adapter,
		registry: registry,
		log:      log,
	}
}

// RunChecks executes a list of checks and returns the results.
func (e *Engine) RunChecks(ctx context.Context, policy *dtmv1alpha1.ChecklistPolicy, checks []dtmv1alpha1.CheckItem) (*dtmv1alpha1.ChecklistResultSpec, error) {
	startedAt := metav1.Now()
	results := make([]dtmv1alpha1.CheckResult, 0, len(checks))

	namespace := ""
	if len(policy.Spec.TargetNamespaces) > 0 {
		namespace = policy.Spec.TargetNamespaces[0]
	}

	toolDefs := e.registry.Definitions()

	for _, check := range checks {
		e.log.Info("running check", "checkID", check.ID, "description", check.Description)

		resp, err := e.adapter.Verify(ctx, llm.VerifyRequest{
			CheckID:     check.ID,
			Description: check.Description,
			Tools:       toolDefs,
			Namespace:   namespace,
		}, func(ctx context.Context, name string, input []byte) (string, error) {
			return e.registry.Call(ctx, name, json.RawMessage(input))
		})

		result := dtmv1alpha1.CheckResult{
			ID:          check.ID,
			Description: check.Description,
			Severity:    check.Severity,
		}

		if err != nil {
			e.log.Error(err, "check verification failed", "checkID", check.ID)
			result.Verdict = dtmv1alpha1.VerdictFail
			result.Reasoning = fmt.Sprintf("Verification error: %v", err)
		} else {
			result.Verdict = dtmv1alpha1.Verdict(resp.Verdict)
			result.Reasoning = resp.Reasoning
			result.Evidence = convertEvidence(resp.ToolCalls)
		}

		results = append(results, result)
	}

	completedAt := metav1.Now()
	summary := computeSummary(results)

	return &dtmv1alpha1.ChecklistResultSpec{
		PolicyRef:   policy.Name,
		StartedAt:   startedAt,
		CompletedAt: &completedAt,
		Checks:      results,
		Summary:     &summary,
	}, nil
}

func convertEvidence(toolCalls []llm.ToolCallRecord) *dtmv1alpha1.Evidence {
	if len(toolCalls) == 0 {
		return nil
	}

	records := make([]dtmv1alpha1.ToolCallRecord, 0, len(toolCalls))
	for _, tc := range toolCalls {
		records = append(records, dtmv1alpha1.ToolCallRecord{
			ToolName: tc.ToolName,
			Input:    &runtime.RawExtension{Raw: tc.Input},
			Output:   tc.Output,
		})
	}
	return &dtmv1alpha1.Evidence{ToolCalls: records}
}

func computeSummary(results []dtmv1alpha1.CheckResult) dtmv1alpha1.ScanSummary {
	summary := dtmv1alpha1.ScanSummary{
		Total: int32(len(results)),
	}
	for _, r := range results {
		switch r.Verdict {
		case dtmv1alpha1.VerdictPass:
			summary.Pass++
		case dtmv1alpha1.VerdictWarn:
			summary.Warn++
		case dtmv1alpha1.VerdictFail:
			summary.Fail++
		}
	}
	return summary
}

// ScanType for passing to RunChecks callers.
func FullScanType() dtmv1alpha1.ScanType  { return dtmv1alpha1.ScanTypeFull }
func RescanType() dtmv1alpha1.ScanType     { return dtmv1alpha1.ScanTypeRescan }

// GenerateResultName creates a unique name for a ChecklistResult.
func GenerateResultName(policyName string, scanType dtmv1alpha1.ScanType) string {
	ts := time.Now().Format("20060102-150405")
	return fmt.Sprintf("%s-%s-%s", policyName, string(scanType), ts)
}
