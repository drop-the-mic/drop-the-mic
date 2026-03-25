<p align="center">
  <img src="docs/images/logo.png" alt="Drop The Mic Logo" width="180" />
</p>

<h1 align="center">Drop The Mic (DTM)</h1>

<p align="center">
  <strong>Kubernetes-native AI Verification Operator</strong><br/>
  Write checklist policies in plain language. LLM verifies your cluster. Get notified when things go wrong.
</p>

<p align="center">
  <a href="#quick-start">Quick Start</a> &bull;
  <a href="#how-it-works">How It Works</a> &bull;
  <a href="#installation">Installation</a> &bull;
  <a href="#configuration">Configuration</a> &bull;
  <a href="#contributing">Contributing</a>
</p>

---

## What is DTM?

DTM is a Kubernetes Operator that lets you define **cluster verification policies in natural language**. An LLM reads your policies, inspects the cluster using tool calls (pods, nodes, events, HPA, PDB, etc.), and reports results to Slack, GitHub Issues, or Jira.

No more brittle shell scripts or forgotten runbook items. Just describe what "healthy" means, and DTM continuously verifies it.

```yaml
apiVersion: dtm.dtm.io/v1alpha1
kind: ChecklistPolicy
metadata:
  name: production-health
spec:
  schedule:
    fullScan: "0 */6 * * *"        # Every 6 hours
    failedRescan: "*/30 * * * *"    # Retry failures every 30 min
  llm:
    provider: claude
    # model: "claude-sonnet-4-20250514"  # Optional — defaults to Haiku (fast & cheap)
    secretRef:
      name: dtm-llm-secret
  checks:
    - id: pod-restarts
      description: "Check if any pods in the production namespace have restarted more than 5 times in the last hour"
      severity: critical
    - id: hpa-saturation
      description: "Verify that no HPA is running at max replicas for more than 10 minutes"
      severity: warning
    - id: pdb-coverage
      description: "Ensure all Deployments with more than 1 replica have a PodDisruptionBudget"
      severity: warning
  notification:
    slack:
      channel: "#k8s-alerts"
      secretRef:
        name: dtm-slack-secret
```

## How It Works

```
┌─────────────────┐     ┌──────────────┐     ┌─────────────┐
│ ChecklistPolicy │────▶│   Operator   │────▶│     LLM     │
│   (Your rules)  │     │  Controller  │     │  (Claude)   │
└─────────────────┘     └──────┬───────┘     │             │
                               │             └──────┬──────┘
                               │                    │
                        ┌──────▼───────┐     ┌──────▼──────┐
                        │  Dual-Loop   │     │  Tool Calls │
                        │  Scheduler   │     │ (read-only) │
                        └──────┬───────┘     └──────┬──────┘
                               │                    │
                        ┌──────▼───────┐     ┌──────▼──────┐
                        │   State      │     │ K8s API     │
                        │   Machine    │     │ (client-go) │
                        └──────┬───────┘     └─────────────┘
                               │
              ┌────────────────┼────────────────┐
              ▼                ▼                ▼
        ┌──────────┐   ┌────────────┐   ┌──────────┐
        │  Slack   │   │   GitHub   │   │   Jira   │
        └──────────┘   └────────────┘   └──────────┘
```

### Dual-Loop Scheduling

- **Full Scan** — runs all checks on a cron schedule
- **Failed Rescan** — retries only failed checks at a faster interval

When a rescan detects recovery, DTM sends a **resolved** notification automatically.

### Alert State Machine

```
UNKNOWN ──▶ FIRING ──▶ RESOLVED
               │
               └──▶ ESCALATED (after N consecutive failures)
```

Duplicate alerts are suppressed. Escalation happens after a configurable threshold of consecutive failures.

### Read-Only by Design

The LLM **never writes** to your cluster. All tool calls are strictly read-only — listing pods, reading events, checking HPA status, etc. DTM observes; it does not mutate.

## Quick Start

### Prerequisites

- Kubernetes cluster (v1.28+)
- Helm v3
- A Claude API key ([get one here](https://console.anthropic.com/))

### Installation

```bash
# 1. Add the Helm repo
helm repo add dtm https://drop-the-mic.github.io/charts
helm repo update

# 2. Install DTM
helm install dtm dtm/drop-the-mic \
  --namespace dtm-system --create-namespace \
  --set operator.image.tag=1.1.0 \
  --set ui.image.tag=1.1.0 \
  --set 'ui.auth.password=<YOUR_PASSWORD>'

# 3. Create the LLM API key secret (can be done after install)
kubectl create secret generic dtm-llm-secret \
  -n dtm-system \
  --from-literal=api-key=<YOUR_CLAUDE_API_KEY>

# 4. Access the Web UI
kubectl port-forward -n dtm-system svc/dtm-drop-the-mic-ui 8090:8090
# Open http://localhost:8090 — login with admin / <YOUR_PASSWORD>
```

> **Tip:** The LLM Secret can be created before or after installation. Without it, the operator logs `secret not found` and retries automatically until the secret is available.
>
> The UI password is optional — if omitted, a random 22-character password is generated. Retrieve it with:
> ```bash
> kubectl get secret dtm-drop-the-mic-auth -n dtm-system -o jsonpath="{.data.password}" | base64 -d
> ```

### Create Your First Policy

Once installed, open the Web UI and click **New Policy**, or apply a YAML directly:

```bash
kubectl apply -f - <<EOF
apiVersion: dtm.dtm.io/v1alpha1
kind: ChecklistPolicy
metadata:
  name: my-first-check
  namespace: dtm-system
spec:
  schedule:
    fullScan: "0 */6 * * *"
  llm:
    provider: claude
    secretRef:
      name: dtm-llm-secret
  checks:
    - id: pod-health
      description: "Check if any pods are in CrashLoopBackOff or Error state"
      severity: critical
EOF
```

The operator will run this check every 6 hours. Click **Run Now** in the UI to trigger it immediately.

### Ingress / Gateway

<details>
<summary><strong>Nginx Ingress</strong></summary>

```bash
helm install dtm dtm/drop-the-mic \
  --namespace dtm-system --create-namespace \
  --set ui.ingress.enabled=true \
  --set ui.ingress.className=nginx \
  --set ui.ingress.host=dtm.example.com
```
</details>

<details>
<summary><strong>Gateway API (Istio, Envoy, etc.)</strong></summary>

```bash
helm install dtm dtm/drop-the-mic \
  --namespace dtm-system --create-namespace \
  --set ui.gateway.enabled=true \
  --set ui.gateway.gatewayRef.name=my-gateway \
  --set ui.gateway.gatewayRef.namespace=istio-system
```
</details>

<details>
<summary><strong>NodePort (no ingress)</strong></summary>

```bash
helm install dtm dtm/drop-the-mic \
  --namespace dtm-system --create-namespace \
  --set ui.service.type=NodePort
```

Access via `http://<node-ip>:<node-port>`.
</details>

### Custom Values File

For production deployments, create a `values-production.yaml`:

```yaml
operator:
  image:
    tag: "1.1.0"
  llm:
    provider: claude
    secretRef: dtm-llm-secret
  resources:
    limits:
      cpu: "1"
      memory: 512Mi

ui:
  image:
    tag: "1.1.0"
  auth:
    enabled: true
    password: ""          # Random generated — retrieve with kubectl
  ingress:
    enabled: true
    className: nginx
    host: dtm.example.com
    annotations:
      cert-manager.io/cluster-issuer: letsencrypt-prod
    tls:
      - secretName: dtm-tls
        hosts:
          - dtm.example.com
```

Then install with:

```bash
helm install dtm dtm/drop-the-mic \
  --namespace dtm-system --create-namespace \
  -f values-production.yaml
```

### From Source

```bash
git clone https://github.com/drop-the-mic/drop-the-mic.git
cd drop-the-mic

make generate      # Generate CRD types and deepcopy
make manifests     # Generate CRD YAML
make build         # Build operator + server binaries
make docker-build  # Build container image
make helm-package  # Package Helm chart
```

## Configuration

### Helm Values

```yaml
operator:
  image: ghcr.io/drop-the-mic/operator:latest
  llm:
    provider: claude          # Currently only claude is supported
    secretRef: dtm-llm-secret

ui:
  enabled: true
  auth:
    enabled: true
    username: admin
    password: ""            # Empty = random 22-char generated
    existingSecret: ""      # Use a pre-existing secret
  service:
    type: ClusterIP
  ingress:
    enabled: false
    className: nginx
    host: dtm.example.com
```

### LLM Providers

| Provider | Status | Default Model | Tool Call Method |
|----------|--------|---------------|-----------------|
| **Claude** | Supported | `claude-haiku-4-5-20251001` | `tool_use` blocks |
| Gemini | Planned | — | `function_calling` |
| OpenAI | Planned | — | `function_calling` |

> **Note:** Currently only the Claude adapter is implemented. Gemini and OpenAI adapters are planned.

#### Model Selection

Each provider has a sensible default model, but you can override it per policy via `spec.llm.model`:

```yaml
spec:
  llm:
    provider: claude
    model: "claude-sonnet-4-20250514"   # Optional — overrides the default
    secretRef:
      name: dtm-llm-secret
```

If `model` is omitted, the provider's default is used:

| Provider | Default Model | Notes |
|----------|--------------|-------|
| Claude | `claude-haiku-4-5-20251001` | Fast and cost-effective for most verification tasks |

You can use any Claude model:

| Model | Use Case | Cost |
|-------|----------|------|
| `claude-haiku-4-5-20251001` | Default — fast, cheap, good for most checks | $ |
| `claude-sonnet-4-20250514` | Better reasoning for complex checks | $$ |
| `claude-opus-4-20250514` | Maximum accuracy for critical checks | $$$ |

### Available Tools

The LLM can call these read-only tools to inspect your cluster:

| Tool | Description |
|------|-------------|
| `list_pods` | List pods with status, restarts, resource usage |
| `list_nodes` | List nodes with conditions, capacity, allocatable |
| `get_events` | Retrieve Kubernetes events (warnings, errors) |
| `check_pdb` | Inspect PodDisruptionBudgets |
| `check_hpa` | Inspect HorizontalPodAutoscalers |
| `check_images` | Verify container image details |
| `get_logs` | Read container logs (tail) |

### Notification Channels

<details>
<summary><strong>Slack</strong></summary>

```bash
# Create secret with Slack webhook URL
kubectl create secret generic dtm-slack-secret -n dtm-system \
  --from-literal=api-key=https://hooks.slack.com/services/T.../B.../xxx
```

```yaml
notification:
  slack:
    channel: "#k8s-alerts"
    secretRef:
      name: dtm-slack-secret
```
</details>

<details>
<summary><strong>GitHub Issues</strong></summary>

```bash
# Create secret with GitHub personal access token (needs repo scope)
kubectl create secret generic dtm-github-secret -n dtm-system \
  --from-literal=api-key=ghp_xxxxxxxxxxxxx
```

```yaml
notification:
  github:
    owner: my-org
    repo: my-repo
    labels: ["dtm", "k8s-health"]
    secretRef:
      name: dtm-github-secret
```
</details>

<details>
<summary><strong>Jira</strong></summary>

```bash
# Create secret with Jira email + API token (two keys required)
kubectl create secret generic dtm-jira-secret -n dtm-system \
  --from-literal=email=user@company.com \
  --from-literal=token=your-jira-api-token
```

```yaml
notification:
  jira:
    url: https://mycompany.atlassian.net
    project: OPS
    issueType: Bug
    secretRef:
      name: dtm-jira-secret
```
</details>

## Web UI

DTM ships with an optional web dashboard for managing policies and viewing results.

- **Login** — JWT-based authentication with custom login page (default: `admin` / random password)
- **Dashboard** — overview of all policies with pass/fail/warn counts and health status
- **Policies** — create, edit, delete policies with natural language checks; real-time status (Healthy/Error/Pending)
- **Results** — browse scan history with verdict/severity tooltips, LLM reasoning, and tool call evidence
- **Settings** — configure notification channels and LLM settings
- **Run Now** — trigger an immediate scan with toast feedback

The UI is embedded in the server binary via `go:embed` — no separate deployment needed.

### Retrieve Initial Password

```bash
kubectl get secret <release>-drop-the-mic-auth -n <namespace> -o jsonpath="{.data.password}" | base64 -d
```

## CRDs

### ChecklistPolicy

User-authored resource defining what to check, when, and where to notify.

```bash
kubectl get checklistpolicies
NAME                PROVIDER   CHECKS   PASS   FAIL   LAST SCAN              AGE
production-health   claude     3        2      1      2026-03-24T12:00:00Z   7d
```

### ChecklistResult

Auto-generated by the operator after each scan. Contains verdicts, LLM reasoning, and raw tool call evidence.

```bash
kubectl get checklistresults -l dtm.dtm.io/policy=production-health --sort-by=.metadata.creationTimestamp
```

## Security

- **Read-only cluster access** — the operator never mutates workloads based on LLM output
- **Secret references** — API keys and tokens are stored in Kubernetes Secrets, never inline in CRDs
- **Scoped Secret access** — the operator can only read Secrets in the release namespace by default, not cluster-wide
- **Separate RBAC** — the operator and UI server use distinct ServiceAccounts with minimal permissions
- **No kubectl exec** — all cluster interaction goes through `client-go`

### Secret Access Scope

By default, the operator can only read Secrets in its own namespace (e.g. `dtm-system`). If your ChecklistPolicies reference Secrets in other namespaces, grant access explicitly:

```yaml
# values.yaml
operator:
  secretAccess:
    namespaces:
      - production
      - staging
```

This creates a namespaced `Role` + `RoleBinding` in each listed namespace — no cluster-wide Secret access is granted.

## Development

```bash
make generate     # Generate deepcopy and CRD manifests
make manifests    # Generate CRD YAML
make lint         # Run golangci-lint
make test         # Run unit + integration tests
make ui-build     # Build React frontend (ui/dist)
make dev          # Deploy to local kind cluster
```

## Project Structure

```
drop-the-mic/
├── operator/          # Go Operator (core)
│   ├── api/           # CRD type definitions
│   ├── internal/
│   │   ├── controller/  # Reconcile loop
│   │   ├── scheduler/   # Dual-loop scheduler
│   │   ├── engine/      # Verification engine + LLM adapters + tools
│   │   ├── state/       # Alert state machine
│   │   └── notify/      # Slack, GitHub, Jira notifiers
│   └── config/          # CRD, RBAC, manager manifests
├── server/            # Go API server (UI backend)
├── ui/                # React + Vite + TypeScript frontend
└── charts/            # Helm chart
```

## Commit Convention

This project follows [Conventional Commits](https://www.conventionalcommits.org/).

```
<type>(<scope>): <description>

[optional body]

[optional footer(s)]
```

### Types

| Type | Description |
|------|-------------|
| `feat` | New feature |
| `fix` | Bug fix |
| `docs` | Documentation only |
| `style` | Code style (formatting, no logic change) |
| `refactor` | Code change that neither fixes a bug nor adds a feature |
| `perf` | Performance improvement |
| `test` | Adding or updating tests |
| `build` | Build system or external dependencies |
| `ci` | CI/CD configuration |
| `chore` | Maintenance tasks (deps, tooling, etc.) |

### Scopes

| Scope | Area |
|-------|------|
| `operator` | Go Operator core |
| `controller` | Controller reconcile loop |
| `scheduler` | Dual-loop scheduler |
| `engine` | Verification engine |
| `llm` | LLM adapters |
| `tool` | Cluster inspection tools |
| `notify` | Notification channels |
| `state` | Alert state machine |
| `server` | Go API server |
| `ui` | React frontend |
| `chart` | Helm chart |
| `crd` | CRD type definitions |

Scope is optional but recommended. Breaking changes must include `!` after the scope or a `BREAKING CHANGE:` footer.

### Examples

```bash
feat(llm): add Gemini adapter
fix(scheduler): prevent duplicate rescan triggers
docs: update README with commit conventions
refactor(tool)!: rename tool package to singular form
chore(deps): bump controller-runtime to v0.19.4
```

## Inspired By

- [k8sgpt](https://github.com/k8sgpt-ai/k8sgpt) — AI-powered Kubernetes troubleshooting
- [kubectl-ai](https://github.com/GoogleCloudPlatform/kubectl-ai) — Natural language Kubernetes commands

## License

Apache License 2.0 — see [LICENSE](LICENSE) for details.
