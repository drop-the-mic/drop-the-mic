import { useState } from 'react';
import { useQuery } from '@tanstack/react-query';
import { api } from '../api/client';
import type { ChecklistResult, CheckResult } from '../api/client';

function Results() {
  const [selectedResult, setSelectedResult] = useState<ChecklistResult | null>(null);

  const { data: results = [], isLoading } = useQuery<ChecklistResult[]>({
    queryKey: ['results'],
    queryFn: () => api.listResults(),
  });

  const sorted = [...results].sort(
    (a, b) => new Date(b.metadata.creationTimestamp).getTime() - new Date(a.metadata.creationTimestamp).getTime()
  );

  if (isLoading) return <div className="empty-state">Loading...</div>;

  return (
    <div>
      <div className="page-header">
        <h2>Results</h2>
      </div>

      {selectedResult ? (
        <ResultDetail result={selectedResult} onBack={() => setSelectedResult(null)} />
      ) : (
        <div className="card">
          {sorted.length === 0 ? (
            <div className="empty-state">No results yet. Run a scan to see results.</div>
          ) : (
            <table>
              <thead>
                <tr>
                  <th>Name</th>
                  <th>Policy</th>
                  <th>Type</th>
                  <th>Pass</th>
                  <th>Warn</th>
                  <th>Fail</th>
                  <th>Phase</th>
                  <th>Time</th>
                </tr>
              </thead>
              <tbody>
                {sorted.map((r) => (
                  <tr key={r.metadata.name}>
                    <td>
                      <a
                        href="#"
                        style={{ color: 'var(--accent)', textDecoration: 'none' }}
                        onClick={(e) => { e.preventDefault(); setSelectedResult(r); }}
                      >
                        {r.metadata.name}
                      </a>
                    </td>
                    <td>{r.spec.policyRef}</td>
                    <td>{r.spec.scanType}</td>
                    <td><span className="badge badge-pass">{r.spec.summary?.pass || 0}</span></td>
                    <td><span className="badge badge-warn">{r.spec.summary?.warn || 0}</span></td>
                    <td><span className="badge badge-fail">{r.spec.summary?.fail || 0}</span></td>
                    <td>{r.status?.phase || '-'}</td>
                    <td>{new Date(r.metadata.creationTimestamp).toLocaleString()}</td>
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

function ResultDetail({ result, onBack }: { result: ChecklistResult; onBack: () => void }) {
  const [expandedCheck, setExpandedCheck] = useState<string | null>(null);

  return (
    <div>
      <button className="btn btn-primary btn-sm" onClick={onBack} style={{ marginBottom: 16 }}>
        Back
      </button>

      <div className="card">
        <h3 style={{ marginBottom: 8 }}>{result.metadata.name}</h3>
        <div style={{ color: 'var(--text-secondary)', fontSize: 13, marginBottom: 16 }}>
          Policy: {result.spec.policyRef} | Type: {result.spec.scanType} | Phase: {result.status?.phase || '-'}
        </div>
        {result.spec.summary && (
          <div className="card-grid" style={{ maxWidth: 500 }}>
            <div className="stat-card"><div className="value" style={{ color: 'var(--success)' }}>{result.spec.summary.pass}</div><div className="label">Pass</div></div>
            <div className="stat-card"><div className="value" style={{ color: 'var(--warning)' }}>{result.spec.summary.warn}</div><div className="label">Warn</div></div>
            <div className="stat-card"><div className="value" style={{ color: 'var(--danger)' }}>{result.spec.summary.fail}</div><div className="label">Fail</div></div>
          </div>
        )}
      </div>

      {result.spec.checks?.map((check: CheckResult) => (
        <div key={check.id} className="card" style={{ cursor: 'pointer' }} onClick={() => setExpandedCheck(expandedCheck === check.id ? null : check.id)}>
          <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
            <div style={{ display: 'flex', alignItems: 'center', gap: 12 }}>
              <VerdictIcon verdict={check.verdict} />
              <div>
                <div style={{ fontWeight: 500 }}>{check.id}</div>
                <div style={{ fontSize: 13, color: 'var(--text-secondary)' }}>{check.description}</div>
              </div>
            </div>
            <span className={`badge badge-${check.verdict.toLowerCase()}`}>{check.verdict}</span>
          </div>

          {expandedCheck === check.id && (
            <div style={{ marginTop: 16, paddingTop: 16, borderTop: '1px solid var(--border)' }}>
              <div className="form-group">
                <label>Reasoning</label>
                <div style={{ whiteSpace: 'pre-wrap', fontSize: 13, lineHeight: 1.6 }}>{check.reasoning}</div>
              </div>
              {check.evidence?.toolCalls && check.evidence.toolCalls.length > 0 && (
                <div className="form-group">
                  <label>Tool Calls ({check.evidence.toolCalls.length})</label>
                  {check.evidence.toolCalls.map((tc, i) => (
                    <div key={i} style={{ background: 'var(--bg-dark)', borderRadius: 8, padding: 12, marginBottom: 8, fontSize: 12, fontFamily: 'monospace' }}>
                      <div style={{ color: 'var(--accent)', marginBottom: 4 }}>{tc.toolName}</div>
                      <div style={{ color: 'var(--text-secondary)', marginBottom: 4 }}>Input: {JSON.stringify(tc.input)}</div>
                      <div style={{ whiteSpace: 'pre-wrap', maxHeight: 200, overflow: 'auto' }}>{tc.output}</div>
                    </div>
                  ))}
                </div>
              )}
            </div>
          )}
        </div>
      ))}
    </div>
  );
}

function VerdictIcon({ verdict }: { verdict: string }) {
  const map: Record<string, { icon: string; color: string }> = {
    PASS: { icon: '\u2713', color: 'var(--success)' },
    WARN: { icon: '!', color: 'var(--warning)' },
    FAIL: { icon: '\u2717', color: 'var(--danger)' },
  };
  const { icon, color } = map[verdict] || { icon: '?', color: 'var(--text-secondary)' };
  return (
    <div style={{
      width: 28, height: 28, borderRadius: '50%',
      display: 'flex', alignItems: 'center', justifyContent: 'center',
      background: `${color}22`, color, fontWeight: 700, fontSize: 14,
    }}>
      {icon}
    </div>
  );
}

export default Results;
