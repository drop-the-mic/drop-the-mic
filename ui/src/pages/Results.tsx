import { useState, useMemo } from 'react';
import { useQuery } from '@tanstack/react-query';
import { api } from '../api/client';
import type { ChecklistResult, CheckResult } from '../api/client';
import { timeAgo, formatDuration } from '../utils/format';
import { Card } from '../components/Card';
import { Badge, VerdictBadge, SeverityBadge } from '../components/Badge';
import { Button } from '../components/Button';
import { LoadingState, EmptyState } from '../components/EmptyState';
import { IconBack, IconResult, IconChevron, IconClock } from '../components/Icons';
import { HelpTooltip } from '../components/HelpTooltip';

const PAGE_SIZE = 20;

function Results() {
  const [selected, setSelected] = useState<ChecklistResult | null>(null);
  const [search, setSearch] = useState('');
  const [filterPolicy, setFilterPolicy] = useState('');
  const [filterVerdict, setFilterVerdict] = useState('');
  const [filterType, setFilterType] = useState('');
  const [page, setPage] = useState(0);

  const { data: results = [], isLoading } = useQuery<ChecklistResult[]>({
    queryKey: ['results'],
    queryFn: () => api.listResults(),
  });

  const filtered = useMemo(() => {
    let list = [...results].sort(
      (a, b) => new Date(b.metadata.creationTimestamp).getTime() - new Date(a.metadata.creationTimestamp).getTime()
    );

    if (search) {
      const q = search.toLowerCase();
      list = list.filter(r =>
        r.metadata.name.toLowerCase().includes(q) ||
        r.spec.policyRef.toLowerCase().includes(q)
      );
    }
    if (filterPolicy) list = list.filter(r => r.spec.policyRef === filterPolicy);
    if (filterType) list = list.filter(r => r.spec.scanType === filterType);
    if (filterVerdict) {
      list = list.filter(r => {
        if (filterVerdict === 'PASS') return r.spec.summary && r.spec.summary.fail === 0 && r.spec.summary.warn === 0;
        if (filterVerdict === 'FAIL') return r.spec.summary && r.spec.summary.fail > 0;
        if (filterVerdict === 'WARN') return r.spec.summary && r.spec.summary.warn > 0 && r.spec.summary.fail === 0;
        return true;
      });
    }
    return list;
  }, [results, search, filterPolicy, filterType, filterVerdict]);

  const totalPages = Math.ceil(filtered.length / PAGE_SIZE);
  const paged = filtered.slice(page * PAGE_SIZE, (page + 1) * PAGE_SIZE);

  const policyNames = [...new Set(results.map(r => r.spec.policyRef))].sort();

  if (isLoading) return <LoadingState />;

  if (selected) {
    return <ResultDetail result={selected} onBack={() => setSelected(null)} />;
  }

  return (
    <div>
      <div className="page-header">
        <div>
          <div className="page-title">Results</div>
          <div className="page-subtitle">{filtered.length} scan {filtered.length === 1 ? 'result' : 'results'}</div>
        </div>
      </div>

      {/* Toolbar */}
      <div className="toolbar">
        <input
          className="toolbar-search"
          placeholder="Search results..."
          aria-label="Search results"
          value={search}
          onChange={e => { setSearch(e.target.value); setPage(0); }}
        />
        <select className="toolbar-select" aria-label="Filter by policy" value={filterPolicy} onChange={e => { setFilterPolicy(e.target.value); setPage(0); }}>
          <option value="">All Policies</option>
          {policyNames.map(n => <option key={n} value={n}>{n}</option>)}
        </select>
        <select className="toolbar-select" value={filterType} onChange={e => { setFilterType(e.target.value); setPage(0); }}>
          <option value="">All Types</option>
          <option value="FullScan">Full Scan</option>
          <option value="Rescan">Rescan</option>
        </select>
        <select className="toolbar-select" value={filterVerdict} onChange={e => { setFilterVerdict(e.target.value); setPage(0); }}>
          <option value="">All Status</option>
          <option value="PASS">All Pass</option>
          <option value="WARN">Has Warnings</option>
          <option value="FAIL">Has Failures</option>
        </select>
      </div>

      {filtered.length === 0 ? (
        <EmptyState
          icon={<IconResult size={40} />}
          title={results.length === 0 ? 'No results yet' : 'No matching results'}
          description={results.length === 0 ? 'Results will appear here after a scan runs.' : 'Try adjusting your filters.'}
        />
      ) : (
        <>
          <Card style={{ padding: 0, overflow: 'hidden' }}>
            <div className="table-wrap">
              <table>
                <thead>
                  <tr>
                    <th>Result</th>
                    <th>Policy</th>
                    <th>Type</th>
                    <th>Pass <HelpTooltip text="Checks that found no issues" /></th>
                    <th>Warn <HelpTooltip text="Checks that found issues not requiring immediate action" /></th>
                    <th>Fail <HelpTooltip text="Checks that found issues requiring immediate action" /></th>
                    <th>Phase</th>
                    <th>Duration</th>
                    <th>Time</th>
                  </tr>
                </thead>
                <tbody>
                  {paged.map(r => {
                    const dur = r.spec.completedAt && r.spec.startedAt
                      ? formatDuration(new Date(r.spec.completedAt).getTime() - new Date(r.spec.startedAt).getTime())
                      : '-';
                    return (
                      <tr key={r.metadata.name}>
                        <td>
                          <span className="link mono" onClick={() => setSelected(r)}>{r.metadata.name}</span>
                        </td>
                        <td>{r.spec.policyRef}</td>
                        <td><Badge variant={r.spec.scanType === 'FullScan' ? 'info' : 'neutral'}>{r.spec.scanType}</Badge></td>
                        <td style={{ color: 'var(--success)' }}>{r.spec.summary?.pass || 0}</td>
                        <td style={{ color: 'var(--warning)' }}>{r.spec.summary?.warn || 0}</td>
                        <td style={{ color: 'var(--danger)' }}>{r.spec.summary?.fail || 0}</td>
                        <td><Badge variant={r.status?.phase === 'Completed' ? 'pass' : r.status?.phase === 'Failed' ? 'fail' : 'neutral'}>{r.status?.phase || '-'}</Badge></td>
                        <td className="mono" style={{ fontSize: 12, color: 'var(--text-tertiary)' }}>{dur}</td>
                        <td style={{ fontSize: 12, color: 'var(--text-tertiary)' }}>
                          <span className="flex items-center gap-2"><IconClock size={11} />{timeAgo(r.metadata.creationTimestamp)}</span>
                        </td>
                      </tr>
                    );
                  })}
                </tbody>
              </table>
            </div>
          </Card>

          {/* Pagination */}
          {totalPages > 1 && (
            <div className="flex items-center justify-between mt-4" style={{ fontSize: 13, color: 'var(--text-tertiary)' }}>
              <span>Page {page + 1} of {totalPages}</span>
              <div className="flex gap-2">
                <Button variant="secondary" size="sm" disabled={page === 0} onClick={() => setPage(p => p - 1)}>Previous</Button>
                <Button variant="secondary" size="sm" disabled={page >= totalPages - 1} onClick={() => setPage(p => p + 1)}>Next</Button>
              </div>
            </div>
          )}
        </>
      )}
    </div>
  );
}

/* Result Detail */
function ResultDetail({ result, onBack }: { result: ChecklistResult; onBack: () => void }) {
  const [expanded, setExpanded] = useState<Set<string>>(new Set());

  const toggle = (id: string) => {
    setExpanded(prev => {
      const next = new Set(prev);
      next.has(id) ? next.delete(id) : next.add(id);
      return next;
    });
  };

  const expandAll = () => {
    if (expanded.size === (result.spec.checks?.length || 0)) {
      setExpanded(new Set());
    } else {
      setExpanded(new Set(result.spec.checks?.map(c => c.id) || []));
    }
  };

  const dur = result.spec.completedAt && result.spec.startedAt
    ? formatDuration(new Date(result.spec.completedAt).getTime() - new Date(result.spec.startedAt).getTime())
    : '-';

  return (
    <div>
      <div className="page-header">
        <div className="flex items-center gap-3">
          <Button variant="ghost" size="sm" icon={<IconBack />} onClick={onBack} />
          <div>
            <div className="page-title mono" style={{ fontSize: 18 }}>{result.metadata.name}</div>
            <div className="page-subtitle flex items-center gap-3">
              <span>Policy: {result.spec.policyRef}</span>
              <Badge variant={result.spec.scanType === 'FullScan' ? 'info' : 'neutral'}>{result.spec.scanType}</Badge>
              <Badge variant={result.status?.phase === 'Completed' ? 'pass' : 'fail'}>{result.status?.phase || '-'}</Badge>
              <span>Duration: {dur}</span>
            </div>
          </div>
        </div>
      </div>

      {/* Summary */}
      {result.spec.summary && (
        <div className="grid grid-4 mb-6">
          <Card style={{ textAlign: 'center', padding: '16px 12px' }}>
            <div style={{ fontSize: 28, fontWeight: 700 }}>{result.spec.summary.total}</div>
            <div style={{ fontSize: 11, color: 'var(--text-tertiary)', marginTop: 4 }}>TOTAL</div>
          </Card>
          <Card style={{ textAlign: 'center', padding: '16px 12px' }}>
            <div style={{ fontSize: 28, fontWeight: 700, color: 'var(--success)' }}>{result.spec.summary.pass}</div>
            <div style={{ fontSize: 11, color: 'var(--text-tertiary)', marginTop: 4 }}>PASS</div>
          </Card>
          <Card style={{ textAlign: 'center', padding: '16px 12px' }}>
            <div style={{ fontSize: 28, fontWeight: 700, color: 'var(--warning)' }}>{result.spec.summary.warn}</div>
            <div style={{ fontSize: 11, color: 'var(--text-tertiary)', marginTop: 4 }}>WARN</div>
          </Card>
          <Card style={{ textAlign: 'center', padding: '16px 12px' }}>
            <div style={{ fontSize: 28, fontWeight: 700, color: 'var(--danger)' }}>{result.spec.summary.fail}</div>
            <div style={{ fontSize: 11, color: 'var(--text-tertiary)', marginTop: 4 }}>FAIL</div>
          </Card>
        </div>
      )}

      {/* Checks */}
      <div className="flex items-center justify-between mb-4">
        <div className="section-title" style={{ marginBottom: 0 }}>Check Results</div>
        <Button variant="ghost" size="sm" onClick={expandAll}>
          {expanded.size === (result.spec.checks?.length || 0) ? 'Collapse All' : 'Expand All'}
        </Button>
      </div>

      <div className="flex flex-col gap-2">
        {result.spec.checks?.map((check: CheckResult) => {
          const isOpen = expanded.has(check.id);
          return (
            <Card key={check.id} style={{ padding: 0, overflow: 'hidden' }}>
              <div
                onClick={() => toggle(check.id)}
                style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', padding: '14px 16px', cursor: 'pointer' }}
              >
                <div className="flex items-center gap-3">
                  <VerdictDot verdict={check.verdict} />
                  <div>
                    <div className="flex items-center gap-2">
                      <span className="mono" style={{ fontWeight: 600, fontSize: 13 }}>{check.id}</span>
                      {check.severity && (
                        <>
                          <SeverityBadge severity={check.severity} />
                          <HelpTooltip size={12} text="Severity is the importance level defined by the user (info / warning / critical).\nIt is separate from the verdict and used for alert escalation." />
                        </>
                      )}
                    </div>
                    <div style={{ fontSize: 12, color: 'var(--text-tertiary)', marginTop: 2, maxWidth: 600, overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>
                      {check.description}
                    </div>
                  </div>
                </div>
                <div className="flex items-center gap-3">
                  <VerdictBadge verdict={check.verdict} />
                  <HelpTooltip size={12} text={"Verdict determined by the LLM:\n• PASS: No issues found\n• WARN: Issues found, but no immediate action needed\n• FAIL: Issues found, immediate action required"} />
                  <IconChevron size={14} className={isOpen ? 'rotate-90' : ''} />
                </div>
              </div>

              {isOpen && (
                <div style={{ padding: '0 16px 16px', borderTop: '1px solid var(--border-subtle)' }}>
                  {/* Reasoning */}
                  <div style={{ marginTop: 14 }}>
                    <div style={{ fontSize: 11, fontWeight: 600, color: 'var(--text-tertiary)', textTransform: 'uppercase', letterSpacing: '0.05em', marginBottom: 8 }}>Reasoning</div>
                    <div style={{
                      whiteSpace: 'pre-wrap', fontSize: 13, lineHeight: 1.7,
                      color: 'var(--text-secondary)', background: 'var(--bg-input)',
                      padding: 14, borderRadius: 'var(--radius-md)', border: '1px solid var(--border-subtle)',
                    }}>
                      {check.reasoning}
                    </div>
                  </div>

                  {/* Tool Calls */}
                  {check.evidence?.toolCalls && check.evidence.toolCalls.length > 0 && (
                    <div style={{ marginTop: 14 }}>
                      <div style={{ fontSize: 11, fontWeight: 600, color: 'var(--text-tertiary)', textTransform: 'uppercase', letterSpacing: '0.05em', marginBottom: 8 }}>
                        Tool Calls ({check.evidence.toolCalls.length})
                      </div>
                      <div className="flex flex-col gap-2">
                        {check.evidence.toolCalls.map((tc, i) => (
                          <div key={i} style={{
                            background: 'var(--bg-input)', borderRadius: 'var(--radius-md)',
                            border: '1px solid var(--border-subtle)', overflow: 'hidden',
                          }}>
                            <div style={{
                              padding: '8px 12px', borderBottom: '1px solid var(--border-subtle)',
                              display: 'flex', alignItems: 'center', justifyContent: 'space-between',
                            }}>
                              <span className="mono" style={{ color: 'var(--accent)', fontWeight: 600, fontSize: 12 }}>{tc.toolName}</span>
                              <span className="mono" style={{ color: 'var(--text-tertiary)', fontSize: 11 }}>
                                {JSON.stringify(tc.input)}
                              </span>
                            </div>
                            <pre style={{
                              padding: 12, margin: 0, fontSize: 11, lineHeight: 1.5,
                              fontFamily: 'var(--font-mono)', color: 'var(--text-secondary)',
                              maxHeight: 200, overflow: 'auto', whiteSpace: 'pre-wrap', wordBreak: 'break-word',
                            }}>
                              {tc.output}
                            </pre>
                          </div>
                        ))}
                      </div>
                    </div>
                  )}

                  {check.failedSince && (
                    <div style={{ marginTop: 10, fontSize: 11, color: 'var(--text-tertiary)' }}>
                      Failing since: {new Date(check.failedSince).toLocaleString()}
                    </div>
                  )}
                </div>
              )}
            </Card>
          );
        })}
      </div>
    </div>
  );
}

function VerdictDot({ verdict }: { verdict: string }) {
  const colors: Record<string, string> = { PASS: 'var(--success)', WARN: 'var(--warning)', FAIL: 'var(--danger)' };
  return (
    <div style={{
      width: 10, height: 10, borderRadius: '50%', flexShrink: 0,
      background: colors[verdict] || 'var(--text-tertiary)',
      boxShadow: `0 0 6px ${colors[verdict] || 'transparent'}`,
    }} />
  );
}

export default Results;
