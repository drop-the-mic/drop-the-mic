import { useQuery } from '@tanstack/react-query';
import { useNavigate } from 'react-router-dom';
import { api } from '../api/client';
import type { ChecklistPolicy, ChecklistResult } from '../api/client';
import { timeAgo } from '../utils/format';
import { Card, StatCard } from '../components/Card';
import { HealthRing } from '../components/HealthRing';
import { VerdictBadge, Badge } from '../components/Badge';
import { LoadingState, EmptyState } from '../components/EmptyState';
import { IconPolicy, IconClock } from '../components/Icons';

function Dashboard() {
  const navigate = useNavigate();
  const { data: policies = [], isLoading: loadingPolicies } = useQuery<ChecklistPolicy[]>({
    queryKey: ['policies'],
    queryFn: () => api.listPolicies(),
  });

  const { data: results = [], isLoading: loadingResults } = useQuery<ChecklistResult[]>({
    queryKey: ['results'],
    queryFn: () => api.listResults(),
  });

  if (loadingPolicies || loadingResults) return <LoadingState />;

  const totalPolicies = policies.length;
  const totalChecks = policies.reduce((sum, p) => sum + (p.spec.checks?.length || 0), 0);

  const agg = policies.reduce(
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

  const latestResults = [...results]
    .sort((a, b) => new Date(b.metadata.creationTimestamp).getTime() - new Date(a.metadata.creationTimestamp).getTime())
    .slice(0, 8);

  const hasData = totalPolicies > 0;

  return (
    <div>
      <div className="page-header">
        <div>
          <div className="page-title">Dashboard</div>
          <div className="page-subtitle">Cluster verification overview</div>
        </div>
      </div>

      {!hasData ? (
        <EmptyState
          icon={<IconPolicy size={40} />}
          title="No policies configured"
          description="Create a ChecklistPolicy custom resource to start verifying your cluster."
        />
      ) : (
        <>
          {/* Stats + Health Ring */}
          <div style={{ display: 'grid', gridTemplateColumns: '1fr 180px', gap: 14, marginBottom: 24 }}>
            <div className="grid grid-4">
              <StatCard value={totalPolicies} label="Policies" color="var(--accent)" />
              <StatCard value={totalChecks} label="Total Checks" />
              <StatCard value={agg.pass} label="Passing" color="var(--success)" />
              <StatCard value={agg.fail} label="Failing" color="var(--danger)" />
            </div>
            <Card style={{ display: 'flex', alignItems: 'center', justifyContent: 'center' }}>
              <HealthRing pass={agg.pass} warn={agg.warn} fail={agg.fail} size={140} />
            </Card>
          </div>

          {/* Policy Status Cards */}
          <div className="section">
            <div className="section-title">Policy Status</div>
            <div className="grid grid-3">
              {policies.map(p => {
                const s = p.status?.summary;
                const isHealthy = s && s.fail === 0;
                return (
                  <Card key={`${p.metadata.namespace}/${p.metadata.name}`} hoverable onClick={() => navigate('/policies')}>
                    <div className="flex items-center justify-between mb-4">
                      <div style={{ fontWeight: 600, fontSize: 14 }}>{p.metadata.name}</div>
                      <Badge variant={isHealthy ? 'pass' : s ? 'fail' : 'neutral'}>
                        {isHealthy ? 'Healthy' : s ? `${s.fail} failing` : 'No data'}
                      </Badge>
                    </div>
                    <div className="flex items-center gap-3" style={{ fontSize: 12, color: 'var(--text-tertiary)' }}>
                      <span>{p.spec.llm.provider}</span>
                      <span>{p.spec.checks.length} checks</span>
                      <span className="mono">{p.spec.schedule.fullScan}</span>
                    </div>
                    {s && (
                      <div className="flex gap-2 mt-4">
                        <ProgressBar pass={s.pass} warn={s.warn} fail={s.fail} />
                      </div>
                    )}
                    {p.status?.lastFullScanTime && (
                      <div className="flex items-center gap-2 mt-4" style={{ fontSize: 11, color: 'var(--text-tertiary)' }}>
                        <IconClock size={12} />
                        <span>Last scan: {timeAgo(p.status.lastFullScanTime)}</span>
                      </div>
                    )}
                  </Card>
                );
              })}
            </div>
          </div>

          {/* Recent Results */}
          <div className="section">
            <div className="flex items-center justify-between mb-4">
              <div className="section-title" style={{ marginBottom: 0 }}>Recent Scans</div>
              {results.length > 8 && (
                <span className="link" style={{ fontSize: 12 }} onClick={() => navigate('/results')}>
                  View all
                </span>
              )}
            </div>
            <Card style={{ padding: 0, overflow: 'hidden' }}>
              {latestResults.length === 0 ? (
                <EmptyState title="No scan results yet" description="Results will appear here after the first scan runs." />
              ) : (
                <div className="table-wrap">
                  <table>
                    <thead>
                      <tr>
                        <th>Result</th>
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
                      {latestResults.map(r => (
                        <tr key={r.metadata.name} style={{ cursor: 'pointer' }} onClick={() => navigate('/results')}>
                          <td className="mono link">{r.metadata.name}</td>
                          <td>{r.spec.policyRef}</td>
                          <td><Badge variant={r.spec.scanType === 'FullScan' ? 'info' : 'neutral'}>{r.spec.scanType}</Badge></td>
                          <td><VerdictBadge verdict="PASS" /> {r.spec.summary?.pass || 0}</td>
                          <td><VerdictBadge verdict="WARN" /> {r.spec.summary?.warn || 0}</td>
                          <td><VerdictBadge verdict="FAIL" /> {r.spec.summary?.fail || 0}</td>
                          <td style={{ color: 'var(--text-secondary)' }}>{r.status?.phase || '-'}</td>
                          <td style={{ color: 'var(--text-tertiary)', fontSize: 12 }}>{timeAgo(r.metadata.creationTimestamp)}</td>
                        </tr>
                      ))}
                    </tbody>
                  </table>
                </div>
              )}
            </Card>
          </div>
        </>
      )}
    </div>
  );
}

function ProgressBar({ pass, warn, fail }: { pass: number; warn: number; fail: number }) {
  const total = pass + warn + fail;
  if (total === 0) return null;
  return (
    <div style={{ display: 'flex', width: '100%', height: 4, borderRadius: 2, overflow: 'hidden', background: 'var(--border)' }}>
      {pass > 0 && <div style={{ width: `${(pass / total) * 100}%`, background: 'var(--success)', transition: 'width 0.3s' }} />}
      {warn > 0 && <div style={{ width: `${(warn / total) * 100}%`, background: 'var(--warning)', transition: 'width 0.3s' }} />}
      {fail > 0 && <div style={{ width: `${(fail / total) * 100}%`, background: 'var(--danger)', transition: 'width 0.3s' }} />}
    </div>
  );
}

export default Dashboard;
