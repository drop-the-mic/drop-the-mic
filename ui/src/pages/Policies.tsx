import { useState } from 'react';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { api } from '../api/client';
import type { ChecklistPolicy } from '../api/client';

function Policies() {
  const queryClient = useQueryClient();
  const [selectedPolicy, setSelectedPolicy] = useState<ChecklistPolicy | null>(null);

  const { data: policies = [], isLoading } = useQuery<ChecklistPolicy[]>({
    queryKey: ['policies'],
    queryFn: () => api.listPolicies(),
  });

  const runNowMutation = useMutation({
    mutationFn: ({ ns, name }: { ns: string; name: string }) => api.runNow(ns, name),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: ['policies'] }),
  });

  const deleteMutation = useMutation({
    mutationFn: ({ ns, name }: { ns: string; name: string }) => api.deletePolicy(ns, name),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: ['policies'] }),
  });

  if (isLoading) return <div className="empty-state">Loading...</div>;

  return (
    <div>
      <div className="page-header">
        <h2>Policies</h2>
      </div>

      {selectedPolicy ? (
        <PolicyDetail
          policy={selectedPolicy}
          onBack={() => setSelectedPolicy(null)}
        />
      ) : (
        <div className="card">
          {policies.length === 0 ? (
            <div className="empty-state">No policies found. Create a ChecklistPolicy CR to get started.</div>
          ) : (
            <table>
              <thead>
                <tr>
                  <th>Name</th>
                  <th>Namespace</th>
                  <th>Provider</th>
                  <th>Checks</th>
                  <th>Schedule</th>
                  <th>Last Scan</th>
                  <th>Status</th>
                  <th>Actions</th>
                </tr>
              </thead>
              <tbody>
                {policies.map((p) => (
                  <tr key={`${p.metadata.namespace}/${p.metadata.name}`}>
                    <td>
                      <a
                        href="#"
                        style={{ color: 'var(--accent)', textDecoration: 'none' }}
                        onClick={(e) => { e.preventDefault(); setSelectedPolicy(p); }}
                      >
                        {p.metadata.name}
                      </a>
                    </td>
                    <td>{p.metadata.namespace}</td>
                    <td>{p.spec.llm.provider}</td>
                    <td>{p.spec.checks.length}</td>
                    <td style={{ fontFamily: 'monospace', fontSize: 12 }}>{p.spec.schedule.fullScan}</td>
                    <td>{p.status?.lastFullScanTime ? new Date(p.status.lastFullScanTime).toLocaleString() : '-'}</td>
                    <td>
                      {p.status?.summary ? (
                        <>
                          <span className="badge badge-pass">{p.status.summary.pass}P</span>{' '}
                          <span className="badge badge-fail">{p.status.summary.fail}F</span>
                        </>
                      ) : '-'}
                    </td>
                    <td>
                      <button
                        className="btn btn-primary btn-sm"
                        onClick={() => runNowMutation.mutate({ ns: p.metadata.namespace, name: p.metadata.name })}
                        disabled={runNowMutation.isPending}
                        style={{ marginRight: 8 }}
                      >
                        Run Now
                      </button>
                      <button
                        className="btn btn-danger btn-sm"
                        onClick={() => {
                          if (confirm(`Delete policy ${p.metadata.name}?`)) {
                            deleteMutation.mutate({ ns: p.metadata.namespace, name: p.metadata.name });
                          }
                        }}
                      >
                        Delete
                      </button>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          )}
        </div>
      )}
    </div>
  );
}

function PolicyDetail({ policy, onBack }: { policy: ChecklistPolicy; onBack: () => void }) {
  return (
    <div>
      <button className="btn btn-primary btn-sm" onClick={onBack} style={{ marginBottom: 16 }}>
        Back
      </button>

      <div className="card">
        <h3 style={{ marginBottom: 16 }}>{policy.metadata.name}</h3>
        <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: 16 }}>
          <div>
            <div className="form-group">
              <label>Namespace</label>
              <div>{policy.metadata.namespace}</div>
            </div>
            <div className="form-group">
              <label>LLM Provider</label>
              <div>{policy.spec.llm.provider} {policy.spec.llm.model && `(${policy.spec.llm.model})`}</div>
            </div>
            <div className="form-group">
              <label>Full Scan Schedule</label>
              <div style={{ fontFamily: 'monospace' }}>{policy.spec.schedule.fullScan}</div>
            </div>
            {policy.spec.schedule.failedRescan && (
              <div className="form-group">
                <label>Failed Rescan Schedule</label>
                <div style={{ fontFamily: 'monospace' }}>{policy.spec.schedule.failedRescan}</div>
              </div>
            )}
          </div>
          <div>
            <div className="form-group">
              <label>Target Namespaces</label>
              <div>{policy.spec.targetNamespaces?.join(', ') || 'All'}</div>
            </div>
            <div className="form-group">
              <label>Escalation Threshold</label>
              <div>{policy.spec.escalationThreshold || 5}</div>
            </div>
          </div>
        </div>
      </div>

      <div className="card">
        <h3 style={{ marginBottom: 16 }}>Checks ({policy.spec.checks.length})</h3>
        <table>
          <thead>
            <tr>
              <th>ID</th>
              <th>Severity</th>
              <th>Description</th>
            </tr>
          </thead>
          <tbody>
            {policy.spec.checks.map((check) => (
              <tr key={check.id}>
                <td style={{ fontFamily: 'monospace' }}>{check.id}</td>
                <td>
                  <span className={`badge badge-${check.severity === 'critical' ? 'fail' : check.severity === 'warning' ? 'warn' : 'pass'}`}>
                    {check.severity}
                  </span>
                </td>
                <td>{check.description}</td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </div>
  );
}

export default Policies;
