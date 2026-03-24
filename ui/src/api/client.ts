const API_BASE = import.meta.env.VITE_API_BASE || '/api/v1';

const TOKEN_KEY = 'dtm_token';

/** Check whether a token exists in localStorage. */
export function isAuthenticated(): boolean {
  return localStorage.getItem(TOKEN_KEY) !== null;
}

/** Remove the stored token and redirect to the login page. */
export function logout(): void {
  localStorage.removeItem(TOKEN_KEY);
  window.location.href = '/login';
}

/** Store the JWT token after a successful login. */
export function setToken(token: string): void {
  localStorage.setItem(TOKEN_KEY, token);
}

/** Retrieve the stored JWT token. */
export function getToken(): string | null {
  return localStorage.getItem(TOKEN_KEY);
}

async function request<T>(path: string, options?: RequestInit): Promise<T> {
  const headers: Record<string, string> = { 'Content-Type': 'application/json' };

  const token = getToken();
  if (token) {
    headers['Authorization'] = `Bearer ${token}`;
  }

  const resp = await fetch(`${API_BASE}${path}`, {
    headers,
    ...options,
  });

  if (resp.status === 401 && !path.startsWith('/login') && !path.startsWith('/auth/')) {
    localStorage.removeItem(TOKEN_KEY);
    window.location.href = '/login';
    throw new Error('unauthorized');
  }

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
    conditions?: Array<{ type: string; status: string; reason: string; message: string; lastTransitionTime?: string }>;
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
  login: (username: string, password: string) =>
    request<{ token: string }>('/login', {
      method: 'POST',
      body: JSON.stringify({ username, password }),
    }),

  authCheck: () => request<{ status: string }>('/auth/check'),

  getInfo: () => request<{ namespace: string }>('/info'),

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
