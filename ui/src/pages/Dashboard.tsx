import { useQuery } from '@tanstack/react-query';
import { api } from '../api/client';
import type { ChecklistPolicy, ChecklistResult } from '../api/client';

function Dashboard() {
  const { data: policies = [] } = useQuery<ChecklistPolicy[]>({
    queryKey: ['policies'],
    queryFn: () => api.listPolicies(),
  });

  const { data: results = [] } = useQuery<ChecklistResult[]>({
    queryKey: ['results'],
    queryFn: () => api.listResults(),
  });

  const totalPolicies = policies.length;
  const totalChecks = policies.reduce((sum, p) => sum + (p.spec.checks?.length || 0), 0);

  const latestResults = [...results]
    .sort((a, b) => new Date(b.metadata.creationTimestamp).getTime() - new Date(a.metadata.creationTimestamp).getTime())
    .slice(0, 10);

  const aggregateSummary = policies.reduce(
    (acc, p) => {
      if (p.status?.summary) {
        acc.pass += p.status.summary.pass;
        acc.warn += p.status.summary.warn;
        acc.fail += p.status.summary.fail;
      }
      return acc;
    },
    { pass: 0, warn: 0, fail: 0 }
  );

  return (
    <div>
      <div className="page-header">
        <h2>Dashboard</h2>
      </div>

      <div className="card-grid">
        <div className="card stat-card">
          <div className="value">{totalPolicies}</div>
          <div className="label">Policies</div>
        </div>
        <div className="card stat-card">
          <div className="value">{totalChecks}</div>
          <div className="label">Total Checks</div>
        </div>
        <div className="card stat-card">
          <div className="value" style={{ color: 'var(--success)' }}>{aggregateSummary.pass}</div>
          <div className="label">Passing</div>
        </div>
        <div className="card stat-card">
          <div className="value" style={{ color: 'var(--warning)' }}>{aggregateSummary.warn}</div>
          <div className="label">Warnings</div>
        </div>
        <div className="card stat-card">
          <div className="value" style={{ color: 'var(--danger)' }}>{aggregateSummary.fail}</div>
          <div className="label">Failures</div>
        </div>
      </div>

      <div className="card">
        <h3 style={{ marginBottom: 16 }}>Recent Scan Results</h3>
        {latestResults.length === 0 ? (
          <div className="empty-state">No scan results yet</div>
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
                <th>Time</th>
              </tr>
            </thead>
            <tbody>
              {latestResults.map((r) => (
                <tr key={r.metadata.name}>
                  <td>{r.metadata.name}</td>
                  <td>{r.spec.policyRef}</td>
                  <td>{r.spec.scanType}</td>
                  <td><span className="badge badge-pass">{r.spec.summary?.pass || 0}</span></td>
                  <td><span className="badge badge-warn">{r.spec.summary?.warn || 0}</span></td>
                  <td><span className="badge badge-fail">{r.spec.summary?.fail || 0}</span></td>
                  <td>{new Date(r.metadata.creationTimestamp).toLocaleString()}</td>
                </tr>
              ))}
            </tbody>
          </table>
        )}
      </div>
    </div>
  );
}

export default Dashboard;
