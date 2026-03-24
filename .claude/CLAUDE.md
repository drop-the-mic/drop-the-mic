# CLAUDE.md — Drop The Mic (DTM)

이 파일은 Claude Code가 DTM 프로젝트를 이해하고 작업하기 위한 컨텍스트 문서다.
코드 작성, 리뷰, 디버깅 시 이 문서를 최우선 참조 기준으로 사용해라.

---

## 프로젝트 한 줄 정의

> **자연어로 작성된 점검 정책(ChecklistPolicy CRD)을 Kubernetes Operator가 주기적으로 실행하고,
> LLM Tool Call로 클러스터를 직접 검증한 뒤 Slack/GitHub/Jira로 리포트하는
> Kubernetes-native AI Verification Operator**

---

## 레포 구조

```
drop-the-mic/
├── CLAUDE.md                  ← 이 파일
├── README.md
├── Makefile
├── docs/
│   └── images/
│       └── logo.png               ← 프로젝트 로고
│
├── operator/                  ← Go Operator (핵심)
│   ├── main.go
│   ├── api/
│   │   └── v1alpha1/
│   │       ├── checklistpolicy_types.go
│   │       ├── checklistresult_types.go
│   │       └── groupversion_info.go
│   ├── internal/
│   │   ├── controller/
│   │   │   └── checklistpolicy_controller.go
│   │   ├── scheduler/
│   │   │   └── scheduler.go        # Full Scan / Failed Rescan 루프
│   │   ├── engine/
│   │   │   ├── engine.go           # Verification Engine 진입점
│   │   │   ├── llm/
│   │   │   │   ├── adapter.go      # LLM 공통 인터페이스
│   │   │   │   ├── claude.go
│   │   │   │   ├── gemini.go
│   │   │   │   └── openai.go
│   │   │   └── tools/
│   │   │       ├── registry.go     # Tool 등록 및 디스패치
│   │   │       ├── pods.go
│   │   │       ├── nodes.go
│   │   │       ├── events.go
│   │   │       ├── pdb.go
│   │   │       ├── hpa.go
│   │   │       ├── images.go
│   │   │       └── logs.go
│   │   ├── state/
│   │   │   └── store.go            # 실패 항목 상태 머신 관리
│   │   └── notify/
│   │       ├── notifier.go         # 공통 인터페이스
│   │       ├── slack.go
│   │       ├── github.go
│   │       └── jira.go
│   └── config/
│       ├── crd/
│       ├── rbac/
│       └── manager/
│
├── ui/                        ← React (Vite) 프론트엔드
│   ├── src/
│   │   ├── pages/
│   │   │   ├── Dashboard.tsx
│   │   │   ├── Policies.tsx       # 자연어 에디터 포함
│   │   │   ├── Results.tsx
│   │   │   └── Settings.tsx
│   │   ├── components/
│   │   └── api/                   # Go API 클라이언트
│   └── dist/                      # 빌드 결과 (Go embed 대상)
│
├── server/                    ← Go API Server (UI 백엔드)
│   ├── main.go
│   ├── handler/
│   │   ├── policies.go
│   │   ├── results.go
│   │   ├── run.go                 # Run Now 핸들러
│   │   └── settings.go
│   └── embed.go                   # ui/dist embed
│
└── charts/
    └── drop-the-mic/
        ├── Chart.yaml
        ├── values.yaml
        ├── crds/
        └── templates/
```

---

## 핵심 개념 — 반드시 숙지

### 1. CRD 두 가지

**ChecklistPolicy** (사용자 작성)
- 점검 정책 정의: 스케줄, LLM 설정, 점검 항목, 알림 채널
- `checks[].description`은 자연어 자유 텍스트 — 절대 파싱하거나 구조화하려 하지 말 것
- 이 리소스가 변경되면 Controller가 reconcile을 트리거한다

**ChecklistResult** (Operator 자동 생성)
- 매 스캔마다 새 오브젝트 생성 (update 아님)
- `checks[].evidence.toolCalls`에 LLM이 호출한 Tool 원본 응답을 반드시 저장
- `checks[].failedSince`: 최초 실패 시각 (알림 중복 제거에 사용)

### 2. Dual-Loop 스케줄러

```
Full Scan  (cron: fullScan)    → 모든 checks 실행
Rescan     (cron: failedRescan) → state.FailedChecks 목록만 실행
```

- Rescan은 Full Scan과 **독립적인 goroutine**으로 실행
- Full Scan 완료 후 `state.Store`의 FailedChecks를 업데이트
- Rescan이 PASS를 감지하면 → RESOLVED 알림 발송 → FailedChecks에서 제거

### 3. LLM Adapter 인터페이스

모든 LLM 구현체는 이 인터페이스를 따른다:

```go
type LLMAdapter interface {
    Verify(ctx context.Context, req VerifyRequest) (VerifyResponse, error)
}

type VerifyRequest struct {
    CheckID     string
    Description string      // 자연어 점검 내용
    Tools       []Tool       // 사용 가능한 Tool 목록
    Namespace   string
}

type VerifyResponse struct {
    Verdict    Verdict       // PASS | WARN | FAIL
    Reasoning  string
    ToolCalls  []ToolCallRecord  // 호출된 Tool과 원본 응답 (Evidence)
}
```

Claude는 `tool_use` 블록, Gemini/OpenAI는 `function_calling`을 사용한다.
각 어댑터는 각 API의 Tool Call 형식을 `VerifyRequest.Tools`로 변환하는 책임을 가진다.

### 4. 알림 상태 머신

```
UNKNOWN → (FAIL 감지)    → FIRING    : 알림 발송
FIRING  → (재스캔 FAIL)  → FIRING    : 알림 억제, retryCount++
FIRING  → (재스캔 PASS)  → RESOLVED  : 회복 알림 발송
FIRING  → (retryCount > threshold) → ESCALATED
```

`state.Store`가 check별로 이 상태를 메모리에 유지한다.
Operator 재시작 시 ChecklistResult CR에서 `failedSince`를 복원한다.

### 5. Run Now 동작 방식

UI에서 "Run Now" → API Server가 ChecklistPolicy에 annotation patch:
```
dtm.io/run-now: "2025-03-24T15:00:00Z"
```
Controller Watch가 annotation 변경을 감지 → 즉시 reconcile 트리거.
실행 완료 후 annotation 제거.

---

## 기술 스택

| 영역 | 기술 | 비고 |
|------|------|------|
| Operator | Go 1.23+, controller-runtime v0.19 | kubebuilder v4 스캐폴딩 |
| CRD 생성 | controller-gen | `make generate` |
| K8s 연동 | client-go | in-cluster config |
| LLM | 자체 adapter | Claude / Gemini / OpenAI |
| 스케줄러 | robfig/cron v3 | cron expression |
| 알림 | slack-go, go-github, go-jira | |
| UI | React 18 + Vite + TypeScript | |
| UI 서빙 | Go embed.FS | 단일 바이너리 |
| Helm | helm/helm SDK v3 | |
| 테스트 | testcontainers-go, envtest | |

---

## 개발 규칙

### 코드 스타일
- Go: `gofmt` + `golangci-lint` 통과 필수
- 에러는 `fmt.Errorf("context: %w", err)` 래핑
- 컨텍스트는 항상 첫 번째 인자로 전달
- 로그는 `controller-runtime/log` 사용 (structured logging)

### 네이밍
- CRD 필드: camelCase (Go), camelCase (YAML spec)
- Tool 이름: `snake_case` (LLM에게 노출되는 이름)
- 패키지명: 단수형 (`tool` not `tools`, `notify` not `notifier`)

### 금지 사항
- `checks[].description` 필드를 파싱하거나 구조화하지 말 것 — 자연어 그대로 LLM에 전달
- LLM 응답을 신뢰하여 클러스터에 write 작업 수행 금지 (read-only 원칙)
- `kubectl` 바이너리 exec 금지 — 반드시 `client-go` 사용
- UI에서 K8s API 직접 호출 금지 — 반드시 Go API Server를 통할 것
- Secret 값을 CRD spec에 인라인으로 저장 금지 — `secretRef` 패턴 사용

### 테스트 원칙
- Tool 함수는 단위 테스트 필수 (mock client-go 사용)
- LLM Adapter는 인터페이스 기반 mock으로 테스트
- Controller는 `envtest`로 통합 테스트
- 알림 발송은 실제 외부 호출 없이 mock

---

## RBAC 설계 원칙

Operator와 UI는 **별도 ServiceAccount**를 사용한다.

**Operator**: 클러스터 전체 read-only + CRD write
**UI API Server**: ChecklistPolicy CRUD + ChecklistResult read + ConfigMap(알림설정) read/write

추가 권한이 필요할 경우 반드시 이 파일에 명시하고 PR에서 리뷰를 받을 것.

---

## Helm values 핵심 구조

```yaml
operator:
  image: ghcr.io/drop-the-mic/operator:latest
  llm:
    provider: claude          # claude | gemini | openai
    secretRef: dtm-llm-secret

ui:
  enabled: true
  service:
    type: ClusterIP           # ClusterIP | NodePort | LoadBalancer
  ingress:
    enabled: false
    className: nginx
    host: dtm.example.com
  gateway:
    enabled: false
    gatewayRef:
      name: ""
      namespace: ""
  auth:
    enabled: false
    type: basic               # basic | oidc
```

---

## 현재 구현 상태 (작업 시작 기준)

- [ ] kubebuilder 프로젝트 초기화
- [ ] ChecklistPolicy / ChecklistResult CRD 타입 정의
- [ ] Controller Reconcile 루프
- [ ] Dual-loop Scheduler
- [ ] Tool Registry + 기본 Tool 구현 (pods, nodes, events, pdb, hpa)
- [ ] LLM Adapter (Claude 우선)
- [ ] 알림 상태 머신 (state.Store)
- [ ] Slack 알림
- [ ] GitHub Issues 알림
- [ ] Jira 알림
- [ ] Go API Server
- [~] React UI (아래 "UI 완성 로드맵" 참조)
- [ ] Go embed 통합
- [x] Helm Chart (charts 레포에 v0.1.0 퍼블리시 완료)
- [ ] Gemini / OpenAI Adapter
- [ ] Cross-model consensus
- [ ] envtest 통합 테스트

---

## UI 완성 로드맵

### 공통

- [ ] 사이드바/헤더에 프로젝트 로고 표시 (`docs/images/logo.png` → `ui/public/logo.png`로 복사하여 사용)
- [ ] 글로벌 에러 핸들링 (API 실패 시 토스트/배너)
- [ ] 반응형 레이아웃 (모바일 대응)

### Dashboard (현재: 기본 완성)

- [x] 정책 수, 체크 수, Pass/Warn/Fail 집계 카드
- [x] 최근 스캔 결과 테이블 (10건)
- [ ] 시계열 트렌드 차트 (Pass/Fail 추이)
- [ ] 정책별 상태 요약 위젯 (한눈에 어떤 정책이 문제인지)

### Policies (현재: 읽기/삭제만)

- [x] 정책 목록 조회
- [x] 정책 상세 보기 (스케줄, LLM 설정, 체크 목록)
- [x] Run Now
- [x] Delete
- [ ] **정책 생성 폼** — 자연어 에디터 포함
  - [ ] 기본 정보 입력 (이름, namespace, 스케줄)
  - [ ] LLM 프로바이더/모델 선택
  - [ ] 체크 항목 추가/삭제/순서 변경 (description은 자연어 자유 텍스트)
  - [ ] severity 선택 (critical/warning/info)
  - [ ] 알림 채널 설정 (Slack/GitHub/Jira)
  - [ ] targetNamespaces 선택
- [ ] **정책 편집 폼** — 생성 폼 재활용, 기존 값 프리필
- [ ] 정책 YAML 미리보기 (생성/편집 시 최종 CR 확인용)
- [ ] 정책 복제 (기존 정책 기반으로 새 정책 생성)

### Results (현재: 기본 완성)

- [x] 결과 목록 (시간순 정렬)
- [x] 상세 보기 (verdict, reasoning, tool call evidence)
- [ ] 정책별/namespace별 필터
- [ ] verdict별 필터 (PASS/WARN/FAIL)
- [ ] 결과 간 비교 (이전 스캔과 diff)

### Settings (현재: raw JSON 에디터)

- [x] JSON 텍스트 에디터로 설정 조회/수정
- [ ] **구조화된 설정 폼**
  - [ ] 알림 채널 관리 (Slack webhook, GitHub token, Jira 연동)
  - [ ] 기본 LLM 프로바이더 설정
  - [ ] escalation threshold 설정
- [ ] 설정 변경 이력

### UI 구현 규칙

- 로고 파일은 `docs/images/logo.png`가 원본, UI에서 사용할 때는 `ui/public/logo.png`로 복사
- `checks[].description`은 자연어 자유 텍스트 — 에디터에서 구조화/파싱하지 말 것 (textarea 사용)
- K8s API 직접 호출 금지 — 반드시 Go API Server(`/api/v1/...`)를 통할 것
- 모든 mutation(생성/수정/삭제) 후 관련 query를 invalidate하여 UI 동기화

---

## 자주 쓰는 Make 커맨드

```bash
make generate        # CRD 타입에서 deepcopy, manifest 생성
make manifests       # CRD YAML 생성
make lint            # golangci-lint 실행
make test            # 단위 + 통합 테스트
make build           # operator + server 바이너리 빌드
make ui-build        # React Vite 빌드 (ui/dist 생성)
make docker-build    # 멀티스테이지 Docker 이미지 빌드
make helm-package    # Helm Chart 패키징
make dev             # 로컬 클러스터에 개발 배포 (kind 기준)
```

---

## GitHub 및 배포 인프라

### GitHub Organization

- **Org**: [drop-the-mic](https://github.com/drop-the-mic)
- **메인 레포**: [drop-the-mic/drop-the-mic](https://github.com/drop-the-mic/drop-the-mic)
- **Helm 차트 레포**: [drop-the-mic/charts](https://github.com/drop-the-mic/charts)

### Helm Chart 배포

차트 레포는 GitHub Pages(`gh-pages` 브랜치)로 호스팅된다.

```bash
# 사용자 설치 명령
helm repo add dtm https://drop-the-mic.github.io/charts
helm repo update
helm install dtm dtm/drop-the-mic
```

### 릴리즈 파이프라인

차트 소스의 원본은 **메인 레포**(`drop-the-mic/drop-the-mic`)의 `charts/` 디렉토리다.
charts 레포는 배포 전용이며 직접 수정하지 않는다.

```
태그 push (v*.*.*)
  │
  ▼  release.yaml (메인 레포 GitHub Action)
  ├─ Docker 이미지 빌드 (operator + server)
  │  └─ ghcr.io/drop-the-mic/operator:<version>
  │  └─ ghcr.io/drop-the-mic/server:<version>
  ├─ GitHub Release 생성 (자동 릴리즈 노트)
  └─ Chart.yaml version/appVersion 업데이트 → charts 레포 동기화
       │
       ▼  release.yaml (charts 레포 chart-releaser-action)
       GitHub Pages (gh-pages) → helm repo index 업데이트
```

**릴리즈 순서:**
1. `develop`에서 기능 개발 완료
2. `develop` → `main` PR 머지
3. 태그 생성: `git tag v0.1.0 && git push origin v0.1.0`
4. 이후 자동:
   - Docker 이미지 빌드 + ghcr.io push
   - GitHub Release 생성
   - Chart version 업데이트 → charts 레포 동기화 → Helm repo 업데이트

**chart 템플릿만 변경한 경우** (릴리즈 없이):
`main`에 `charts/**` 변경이 푸시되면 `sync-charts.yaml`이 charts 레포에 자동 동기화한다.

### CI Secrets

| Secret | 레포 | 용도 |
|--------|------|------|
| `GITHUB_TOKEN` | 자동 제공 | GHCR push, GitHub Release 생성 |
| `CHARTS_SYNC_TOKEN` | 메인 레포 | charts 레포에 push (PAT, Contents read/write) |

### CI 워크플로우 파일

| 파일 | 트리거 | 동작 |
|------|--------|------|
| `.github/workflows/release.yaml` | `v*` 태그 push | 이미지 빌드, Release 생성, charts 동기화 |
| `.github/workflows/sync-charts.yaml` | `main` push (`charts/**`) | charts 레포에 템플릿 동기화 |

### 차트 구조 (charts 레포)

```
charts/
└── drop-the-mic/
    ├── Chart.yaml
    ├── values.yaml
    └── templates/
        ├── _helpers.tpl
        ├── operator-deployment.yaml
        ├── operator-serviceaccount.yaml
        ├── operator-rbac.yaml
        ├── server-deployment.yaml
        ├── server-service.yaml
        ├── server-serviceaccount.yaml
        ├── server-rbac.yaml
        └── ingress.yaml
```

---

## 참고 자료

- [controller-runtime 공식 문서](https://pkg.go.dev/sigs.k8s.io/controller-runtime)
- [kubebuilder book](https://book.kubebuilder.io)
- [Kubernetes API Conventions](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/api-conventions.md)
- 유사 프로젝트 (참고용): [k8sgpt](https://github.com/k8sgpt-ai/k8sgpt), [kubectl-ai](https://github.com/GoogleCloudPlatform/kubectl-ai)
