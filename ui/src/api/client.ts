const API_BASE = import.meta.env.VITE_API_BASE || '/api/v1';

async function request<T>(path: string, options?: RequestInit): Promise<T> {
  const resp = await fetch(`${API_BASE}${path}`, {
    headers: { 'Content-Type': 'application/json' },
    ...options,
  });
  if (!resp.ok) {
    const err = await resp.json().catch(() => ({ error: resp.statusText }));
    throw new Error(err.error || resp.statusText);
  }
  return resp.json();
}

export interface CheckItem {
  id: string;
  description: string;
  severity: string;
}

export interface ScheduleConfig {
  fullScan: string;
  failedRescan?: string;
}

export interface LLMConfig {
  provider: string;
  model?: string;
  secretRef: { name: string; key?: string };
}

export interface ChecklistPolicy {
  metadata: { name: string; namespace: string; creationTimestamp: string };
  spec: {
    schedule: ScheduleConfig;
    llm: LLMConfig;
    checks: CheckItem[];
    targetNamespaces?: string[];
    notification?: Record<string, unknown>;
    escalationThreshold?: number;
  };
  status?: {
    lastFullScanTime?: string;
    lastRescanTime?: string;
    lastResultName?: string;
    summary?: { total: number; pass: number; warn: number; fail: number };
    conditions?: Array<{ type: string; status: string; message: string }>;
  };
}

export interface CheckResult {
  id: string;
  description: string;
  verdict: 'PASS' | 'WARN' | 'FAIL';
  reasoning: string;
  severity?: string;
  failedSince?: string;
  evidence?: {
    toolCalls?: Array<{ toolName: string; input: unknown; output: string }>;
  };
}

export interface ChecklistResult {
  metadata: { name: string; namespace: string; creationTimestamp: string };
  spec: {
    policyRef: string;
    scanType: string;
    startedAt: string;
    completedAt?: string;
    checks?: CheckResult[];
    summary?: { total: number; pass: number; warn: number; fail: number };
  };
  status?: { phase: string };
}

export const api = {
  listPolicies: (namespace?: string) =>
    request<ChecklistPolicy[]>(`/policies${namespace ? `?namespace=${namespace}` : ''}`),

  getPolicy: (namespace: string, name: string) =>
    request<ChecklistPolicy>(`/policies/${namespace}/${name}`),

  createPolicy: (policy: ChecklistPolicy) =>
    request<ChecklistPolicy>('/policies', { method: 'POST', body: JSON.stringify(policy) }),

  updatePolicy: (namespace: string, name: string, spec: ChecklistPolicy['spec']) =>
    request<ChecklistPolicy>(`/policies/${namespace}/${name}`, {
      method: 'PUT',
      body: JSON.stringify(spec),
    }),

  deletePolicy: (namespace: string, name: string) =>
    request<{ status: string }>(`/policies/${namespace}/${name}`, { method: 'DELETE' }),

  listResults: (namespace?: string, policy?: string) => {
    const params = new URLSearchParams();
    if (namespace) params.set('namespace', namespace);
    if (policy) params.set('policy', policy);
    const qs = params.toString();
    return request<ChecklistResult[]>(`/results${qs ? `?${qs}` : ''}`);
  },

  getResult: (namespace: string, name: string) =>
    request<ChecklistResult>(`/results/${namespace}/${name}`),

  runNow: (namespace: string, name: string) =>
    request<{ status: string }>(`/run/${namespace}/${name}`, { method: 'POST' }),

  getSettings: () => request<Record<string, unknown>>('/settings'),

  updateSettings: (settings: Record<string, unknown>) =>
    request<Record<string, unknown>>('/settings', {
      method: 'PUT',
      body: JSON.stringify(settings),
    }),
};
