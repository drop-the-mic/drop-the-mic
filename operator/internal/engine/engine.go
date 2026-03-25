// Package engine orchestrates LLM-based verification of checklist items by
// combining an LLM adapter with a tool registry, converting responses into
// ChecklistResult specs.
package engine

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/go-logr/logr"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"

	dtmv1alpha1 "github.com/drop-the-mic/operator/api/v1alpha1"
	"github.com/drop-the-mic/operator/internal/engine/llm"
	"github.com/drop-the-mic/operator/internal/engine/tool"
)

const batchSize = 5

// snapshotTools lists the tools invoked during snapshot collection.
var snapshotTools = []struct {
	Name   string
	Params json.RawMessage
}{
	{Name: "list_pods", Params: json.RawMessage(`{}`)},
	{Name: "list_nodes", Params: json.RawMessage(`{}`)},
	{Name: "get_events", Params: json.RawMessage(`{}`)},
}

// Engine orchestrates the verification of checks using an LLM and tools.
type Engine struct {
	adapter  llm.Adapter
	registry *tool.Registry
	log      logr.Logger

	snapshotMu     sync.Mutex
	snapshotHashes map[string]string                  // policyKey → snapshot hash
	lastResults    map[string]map[string]llm.Verdict   // policyKey → checkID → verdict
}

// New creates a new verification engine.
func New(adapter llm.Adapter, registry *tool.Registry, log logr.Logger) *Engine {
	return &Engine{
		adapter:        adapter,
		registry:       registry,
		log:            log,
		snapshotHashes: make(map[string]string),
		lastResults:    make(map[string]map[string]llm.Verdict),
	}
}

func hashSnapshot(data string) string {
	h := sha256.Sum256([]byte(data))
	return fmt.Sprintf("%x", h[:16])
}

// collectSnapshot pre-fetches basic cluster data to include in LLM requests.
func (e *Engine) collectSnapshot(ctx context.Context, cache *tool.CachingRegistry) string {
	var sb strings.Builder
	for _, st := range snapshotTools {
		result, err := cache.Call(ctx, st.Name, st.Params)
		if err != nil {
			e.log.V(1).Info("snapshot tool failed", "tool", st.Name, "error", err)
			continue
		}
		sb.WriteString(fmt.Sprintf("### %s\n%s\n\n", st.Name, result))
	}
	return sb.String()
}

// RunChecks executes a list of checks and returns the results.
func (e *Engine) RunChecks(ctx context.Context, policy *dtmv1alpha1.ChecklistPolicy, checks []dtmv1alpha1.CheckItem) (*dtmv1alpha1.ChecklistResultSpec, error) {
	startedAt := metav1.Now()
	results := make([]dtmv1alpha1.CheckResult, 0, len(checks))

	namespaces := policy.Spec.TargetNamespaces

	// Use a caching registry to avoid duplicate tool calls within a scan.
	cache := tool.NewCachingRegistry(e.registry)
	toolDefs := cache.Definitions()

	// Collect snapshot for pre-fetched data.
	snapshot := e.collectSnapshot(ctx, cache)

	policyKey := fmt.Sprintf("%s/%s", policy.Namespace, policy.Name)

	// Rescan skip: if snapshot hasn't changed, reuse previous verdicts.
	currentHash := hashSnapshot(snapshot)
	e.snapshotMu.Lock()
	prevHash := e.snapshotHashes[policyKey]
	prevResults := e.lastResults[policyKey]
	e.snapshotMu.Unlock()

	if prevHash == currentHash && prevResults != nil && len(checks) > 0 {
		allCached := true
		for _, check := range checks {
			if _, ok := prevResults[check.ID]; !ok {
				allCached = false
				break
			}
		}
		if allCached {
			e.log.Info("snapshot unchanged, reusing previous verdicts", "policy", policyKey)
			for _, check := range checks {
				results = append(results, dtmv1alpha1.CheckResult{
					ID:          check.ID,
					Description: check.Description,
					Severity:    check.Severity,
					Verdict:     dtmv1alpha1.Verdict(prevResults[check.ID]),
					Reasoning:   "Snapshot unchanged — previous verdict reused.",
				})
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
	}

	// Batch checks to reduce LLM round-trips.
	for i := 0; i < len(checks); i += batchSize {
		end := i + batchSize
		if end > len(checks) {
			end = len(checks)
		}
		batch := checks[i:end]

		if len(batch) == 1 {
			// Single check — use normal Verify.
			check := batch[0]
			e.log.Info("running check", "checkID", check.ID, "description", check.Description)

			resp, err := e.adapter.Verify(ctx, llm.VerifyRequest{
				CheckID:     check.ID,
				Description: check.Description,
				Tools:       toolDefs,
				Namespaces:  namespaces,
				Snapshot:    snapshot,
			}, func(ctx context.Context, name string, input []byte) (string, error) {
				return cache.Call(ctx, name, json.RawMessage(input))
			})

			result := dtmv1alpha1.CheckResult{
				ID:          check.ID,
				Description: check.Description,
				Severity:    check.Severity,
			}

			if err != nil {
				e.log.Error(err, "check verification failed", "checkID", check.ID)
				result.Verdict = dtmv1alpha1.VerdictFail
				result.Reasoning = fmt.Sprintf("LLM API error: %v", err)
			} else {
				result.Verdict = dtmv1alpha1.Verdict(resp.Verdict)
				result.Reasoning = resp.Reasoning
				result.Evidence = convertEvidence(resp.ToolCalls)
			}

			results = append(results, result)
		} else {
			// Multiple checks — use BatchVerify.
			e.log.Info("running batch", "size", len(batch), "firstCheckID", batch[0].ID)

			batchReqs := make([]llm.VerifyRequest, 0, len(batch))
			for _, check := range batch {
				batchReqs = append(batchReqs, llm.VerifyRequest{
					CheckID:     check.ID,
					Description: check.Description,
				})
			}

			responses, err := e.adapter.BatchVerify(ctx, llm.BatchVerifyRequest{
				Checks:     batchReqs,
				Tools:      toolDefs,
				Namespaces: namespaces,
				Snapshot:   snapshot,
			}, func(ctx context.Context, name string, input []byte) (string, error) {
				return cache.Call(ctx, name, json.RawMessage(input))
			})

			if err != nil {
				e.log.Error(err, "batch verification failed")
				for _, check := range batch {
					results = append(results, dtmv1alpha1.CheckResult{
						ID:          check.ID,
						Description: check.Description,
						Severity:    check.Severity,
						Verdict:     dtmv1alpha1.VerdictFail,
						Reasoning:   fmt.Sprintf("LLM batch API error: %v", err),
					})
				}
			} else {
				for j, check := range batch {
					result := dtmv1alpha1.CheckResult{
						ID:          check.ID,
						Description: check.Description,
						Severity:    check.Severity,
					}
					if j < len(responses) {
						result.Verdict = dtmv1alpha1.Verdict(responses[j].Verdict)
						result.Reasoning = responses[j].Reasoning
						result.Evidence = convertEvidence(responses[j].ToolCalls)
					} else {
						result.Verdict = dtmv1alpha1.VerdictFail
						result.Reasoning = "No response from batch for this check."
					}
					results = append(results, result)
				}
			}
		}
	}

	completedAt := metav1.Now()
	summary := computeSummary(results)

	// Store snapshot hash and results for rescan skip.
	e.snapshotMu.Lock()
	e.snapshotHashes[policyKey] = currentHash
	verdictMap := make(map[string]llm.Verdict, len(results))
	for _, r := range results {
		verdictMap[r.ID] = llm.Verdict(r.Verdict)
	}
	e.lastResults[policyKey] = verdictMap
	e.snapshotMu.Unlock()

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

// FullScanType returns the ScanType constant for full scans.
func FullScanType() dtmv1alpha1.ScanType { return dtmv1alpha1.ScanTypeFull }

// RescanType returns the ScanType constant for failed-only rescans.
func RescanType() dtmv1alpha1.ScanType { return dtmv1alpha1.ScanTypeRescan }

// GenerateResultName creates a unique name for a ChecklistResult.
func GenerateResultName(policyName string, scanType dtmv1alpha1.ScanType) string {
	ts := time.Now().Format("20060102-150405")
	return fmt.Sprintf("%s-%s-%s", policyName, string(scanType), ts)
}
