import { useState } from 'react';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { api } from '../api/client';
import type { ChecklistPolicy, CheckItem } from '../api/client';
import { Card } from '../components/Card';
import { Badge, SeverityBadge } from '../components/Badge';
import { Button } from '../components/Button';
import { Modal } from '../components/Modal';
import { FormField, Input, TextArea, Select } from '../components/FormField';
import { LoadingState, EmptyState } from '../components/EmptyState';
import { IconPlus, IconPlay, IconTrash, IconBack, IconPolicy, IconClock } from '../components/Icons';

function Policies() {
  const queryClient = useQueryClient();
  const [selected, setSelected] = useState<ChecklistPolicy | null>(null);
  const [showCreate, setShowCreate] = useState(false);

  const { data: policies = [], isLoading } = useQuery<ChecklistPolicy[]>({
    queryKey: ['policies'],
    queryFn: () => api.listPolicies(),
  });

  const runNow = useMutation({
    mutationFn: ({ ns, name }: { ns: string; name: string }) => api.runNow(ns, name),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: ['policies'] }),
  });

  const deletePol = useMutation({
    mutationFn: ({ ns, name }: { ns: string; name: string }) => api.deletePolicy(ns, name),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: ['policies'] }),
  });

  if (isLoading) return <LoadingState />;

  if (selected) {
    return <PolicyDetail policy={selected} onBack={() => setSelected(null)} />;
  }

  return (
    <div>
      <div className="page-header">
        <div>
          <div className="page-title">Policies</div>
          <div className="page-subtitle">{policies.length} checklist {policies.length === 1 ? 'policy' : 'policies'}</div>
        </div>
        <Button icon={<IconPlus />} onClick={() => setShowCreate(true)}>New Policy</Button>
      </div>

      {policies.length === 0 ? (
        <EmptyState
          icon={<IconPolicy size={40} />}
          title="No policies yet"
          description="Create a ChecklistPolicy to define verification checks for your cluster."
          action={<Button icon={<IconPlus />} onClick={() => setShowCreate(true)}>Create Policy</Button>}
        />
      ) : (
        <Card style={{ padding: 0, overflow: 'hidden' }}>
          <div className="table-wrap">
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
                  <th style={{ width: 160 }}>Actions</th>
                </tr>
              </thead>
              <tbody>
                {policies.map(p => {
                  const s = p.status?.summary;
                  return (
                    <tr key={`${p.metadata.namespace}/${p.metadata.name}`}>
                      <td>
                        <span className="link" onClick={() => setSelected(p)}>{p.metadata.name}</span>
                      </td>
                      <td><Badge variant="neutral">{p.metadata.namespace}</Badge></td>
                      <td style={{ textTransform: 'capitalize' }}>{p.spec.llm.provider}</td>
                      <td>{p.spec.checks.length}</td>
                      <td className="mono">{p.spec.schedule.fullScan}</td>
                      <td style={{ fontSize: 12, color: 'var(--text-tertiary)' }}>
                        {p.status?.lastFullScanTime ? (
                          <span className="flex items-center gap-2"><IconClock size={12} />{timeAgo(p.status.lastFullScanTime)}</span>
                        ) : '-'}
                      </td>
                      <td>
                        {s ? (
                          <div className="flex gap-2">
                            <Badge variant="pass">{s.pass}P</Badge>
                            {s.warn > 0 && <Badge variant="warn">{s.warn}W</Badge>}
                            {s.fail > 0 && <Badge variant="fail">{s.fail}F</Badge>}
                          </div>
                        ) : <span style={{ color: 'var(--text-tertiary)' }}>-</span>}
                      </td>
                      <td>
                        <div className="flex gap-2">
                          <Button
                            variant="secondary" size="sm" icon={<IconPlay size={12} />}
                            onClick={() => runNow.mutate({ ns: p.metadata.namespace, name: p.metadata.name })}
                            disabled={runNow.isPending}
                          >
                            Run
                          </Button>
                          <Button
                            variant="danger" size="sm" icon={<IconTrash size={12} />}
                            onClick={() => { if (confirm(`Delete policy "${p.metadata.name}"?`)) deletePol.mutate({ ns: p.metadata.namespace, name: p.metadata.name }); }}
                          />
                        </div>
                      </td>
                    </tr>
                  );
                })}
              </tbody>
            </table>
          </div>
        </Card>
      )}

      <CreatePolicyModal open={showCreate} onClose={() => setShowCreate(false)} />
    </div>
  );
}

/* Detail view */
function PolicyDetail({ policy, onBack }: { policy: ChecklistPolicy; onBack: () => void }) {
  const queryClient = useQueryClient();
  const runNow = useMutation({
    mutationFn: () => api.runNow(policy.metadata.namespace, policy.metadata.name),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: ['policies'] }),
  });

  const s = policy.status?.summary;

  return (
    <div>
      <div className="page-header">
        <div className="flex items-center gap-3">
          <Button variant="ghost" size="sm" icon={<IconBack />} onClick={onBack} />
          <div>
            <div className="page-title">{policy.metadata.name}</div>
            <div className="page-subtitle">{policy.metadata.namespace}</div>
          </div>
        </div>
        <Button icon={<IconPlay size={14} />} onClick={() => runNow.mutate()} disabled={runNow.isPending}>
          {runNow.isPending ? 'Running...' : 'Run Now'}
        </Button>
      </div>

      {/* Info cards */}
      <div className="grid grid-3 mb-6">
        <Card>
          <div style={{ fontSize: 11, color: 'var(--text-tertiary)', textTransform: 'uppercase', letterSpacing: '0.06em', marginBottom: 8, fontWeight: 600 }}>Configuration</div>
          <InfoRow label="Provider" value={policy.spec.llm.provider + (policy.spec.llm.model ? ` (${policy.spec.llm.model})` : '')} />
          <InfoRow label="Secret" value={policy.spec.llm.secretRef.name} mono />
          <InfoRow label="Namespaces" value={policy.spec.targetNamespaces?.join(', ') || 'All'} />
          <InfoRow label="Escalation" value={`${policy.spec.escalationThreshold || 5} failures`} />
        </Card>
        <Card>
          <div style={{ fontSize: 11, color: 'var(--text-tertiary)', textTransform: 'uppercase', letterSpacing: '0.06em', marginBottom: 8, fontWeight: 600 }}>Schedule</div>
          <InfoRow label="Full Scan" value={policy.spec.schedule.fullScan} mono />
          <InfoRow label="Rescan" value={policy.spec.schedule.failedRescan || 'Disabled'} mono />
          <InfoRow label="Last Full Scan" value={policy.status?.lastFullScanTime ? timeAgo(policy.status.lastFullScanTime) : 'Never'} />
          <InfoRow label="Last Rescan" value={policy.status?.lastRescanTime ? timeAgo(policy.status.lastRescanTime) : 'Never'} />
        </Card>
        <Card>
          <div style={{ fontSize: 11, color: 'var(--text-tertiary)', textTransform: 'uppercase', letterSpacing: '0.06em', marginBottom: 8, fontWeight: 600 }}>Summary</div>
          {s ? (
            <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: 12, marginTop: 4 }}>
              <MiniStat label="Total" value={s.total} />
              <MiniStat label="Pass" value={s.pass} color="var(--success)" />
              <MiniStat label="Warn" value={s.warn} color="var(--warning)" />
              <MiniStat label="Fail" value={s.fail} color="var(--danger)" />
            </div>
          ) : (
            <div style={{ color: 'var(--text-tertiary)', fontSize: 13 }}>No scan data yet</div>
          )}
        </Card>
      </div>

      {/* Checks list */}
      <div className="section-title">Checks ({policy.spec.checks.length})</div>
      <Card style={{ padding: 0, overflow: 'hidden' }}>
        <table>
          <thead>
            <tr>
              <th style={{ width: 140 }}>ID</th>
              <th style={{ width: 100 }}>Severity</th>
              <th>Description</th>
            </tr>
          </thead>
          <tbody>
            {policy.spec.checks.map(check => (
              <tr key={check.id}>
                <td className="mono">{check.id}</td>
                <td><SeverityBadge severity={check.severity} /></td>
                <td style={{ lineHeight: 1.6 }}>{check.description}</td>
              </tr>
            ))}
          </tbody>
        </table>
      </Card>

      {/* Conditions */}
      {policy.status?.conditions && policy.status.conditions.length > 0 && (
        <div className="mt-4">
          <div className="section-title">Conditions</div>
          <Card style={{ padding: 0, overflow: 'hidden' }}>
            <table>
              <thead>
                <tr><th>Type</th><th>Status</th><th>Message</th></tr>
              </thead>
              <tbody>
                {policy.status.conditions.map((c, i) => (
                  <tr key={i}>
                    <td className="mono">{c.type}</td>
                    <td><Badge variant={c.status === 'True' ? 'pass' : 'fail'}>{c.status}</Badge></td>
                    <td style={{ fontSize: 12, color: 'var(--text-secondary)' }}>{c.message}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </Card>
        </div>
      )}
    </div>
  );
}

function InfoRow({ label, value, mono }: { label: string; value: string; mono?: boolean }) {
  return (
    <div className="flex justify-between" style={{ padding: '6px 0', borderBottom: '1px solid var(--border-subtle)' }}>
      <span style={{ fontSize: 12, color: 'var(--text-tertiary)' }}>{label}</span>
      <span style={{ fontSize: 13, fontFamily: mono ? 'var(--font-mono)' : undefined, color: 'var(--text-primary)' }}>{value}</span>
    </div>
  );
}

function MiniStat({ label, value, color }: { label: string; value: number; color?: string }) {
  return (
    <div>
      <div style={{ fontSize: 22, fontWeight: 700, color: color || 'var(--text-primary)' }}>{value}</div>
      <div style={{ fontSize: 11, color: 'var(--text-tertiary)' }}>{label}</div>
    </div>
  );
}

/* Create Policy Modal */
function CreatePolicyModal({ open, onClose }: { open: boolean; onClose: () => void }) {
  const queryClient = useQueryClient();
  const [name, setName] = useState('');
  const [namespace, setNamespace] = useState('default');
  const [provider, setProvider] = useState('claude');
  const [model, setModel] = useState('');
  const [secretName, setSecretName] = useState('dtm-llm-secret');
  const [fullScan, setFullScan] = useState('0 */6 * * *');
  const [failedRescan, setFailedRescan] = useState('*/30 * * * *');
  const [targetNs, setTargetNs] = useState('');
  const [checks, setChecks] = useState<CheckItem[]>([
    { id: 'check-1', description: '', severity: 'warning' },
  ]);

  const createMut = useMutation({
    mutationFn: (policy: ChecklistPolicy) => api.createPolicy(policy),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['policies'] });
      onClose();
      resetForm();
    },
  });

  const resetForm = () => {
    setName(''); setNamespace('default'); setProvider('claude'); setModel('');
    setSecretName('dtm-llm-secret'); setFullScan('0 */6 * * *'); setFailedRescan('*/30 * * * *');
    setTargetNs(''); setChecks([{ id: 'check-1', description: '', severity: 'warning' }]);
  };

  const addCheck = () => {
    setChecks([...checks, { id: `check-${checks.length + 1}`, description: '', severity: 'warning' }]);
  };

  const removeCheck = (idx: number) => {
    if (checks.length <= 1) return;
    setChecks(checks.filter((_, i) => i !== idx));
  };

  const updateCheck = (idx: number, field: keyof CheckItem, value: string) => {
    setChecks(checks.map((c, i) => i === idx ? { ...c, [field]: value } : c));
  };

  const handleSubmit = () => {
    if (!name.trim()) return;
    const validChecks = checks.filter(c => c.id.trim() && c.description.trim());
    if (validChecks.length === 0) return;

    const policy: ChecklistPolicy = {
      metadata: { name, namespace, creationTimestamp: '' },
      spec: {
        schedule: { fullScan, failedRescan: failedRescan || undefined },
        llm: { provider, model: model || undefined, secretRef: { name: secretName } },
        checks: validChecks,
        targetNamespaces: targetNs.trim() ? targetNs.split(',').map(s => s.trim()) : undefined,
      },
    };
    createMut.mutate(policy);
  };

  return (
    <Modal open={open} onClose={onClose} title="Create Policy" width={640}>
      <div className="grid grid-2" style={{ gap: 12 }}>
        <FormField label="Name">
          <Input value={name} onChange={e => setName(e.target.value)} placeholder="my-policy" />
        </FormField>
        <FormField label="Namespace">
          <Input value={namespace} onChange={e => setNamespace(e.target.value)} placeholder="default" />
        </FormField>
      </div>

      <div className="grid grid-3" style={{ gap: 12 }}>
        <FormField label="LLM Provider">
          <Select value={provider} onChange={e => setProvider(e.target.value)}>
            <option value="claude">Claude</option>
            <option value="gemini">Gemini</option>
            <option value="openai">OpenAI</option>
          </Select>
        </FormField>
        <FormField label="Model (optional)">
          <Input value={model} onChange={e => setModel(e.target.value)} placeholder="claude-sonnet-4-20250514" />
        </FormField>
        <FormField label="Secret Name">
          <Input value={secretName} onChange={e => setSecretName(e.target.value)} />
        </FormField>
      </div>

      <div className="grid grid-2" style={{ gap: 12 }}>
        <FormField label="Full Scan Schedule">
          <Input value={fullScan} onChange={e => setFullScan(e.target.value)} placeholder="0 */6 * * *" style={{ fontFamily: 'var(--font-mono)' }} />
        </FormField>
        <FormField label="Failed Rescan Schedule">
          <Input value={failedRescan} onChange={e => setFailedRescan(e.target.value)} placeholder="*/30 * * * *" style={{ fontFamily: 'var(--font-mono)' }} />
        </FormField>
      </div>

      <FormField label="Target Namespaces (comma-separated, empty = all)">
        <Input value={targetNs} onChange={e => setTargetNs(e.target.value)} placeholder="default, kube-system" />
      </FormField>

      {/* Checks Editor */}
      <div style={{ marginBottom: 16 }}>
        <div className="flex items-center justify-between mb-4">
          <label style={{ fontSize: 12, fontWeight: 600, color: 'var(--text-tertiary)', textTransform: 'uppercase', letterSpacing: '0.04em' }}>
            Checks ({checks.length})
          </label>
          <Button variant="secondary" size="sm" icon={<IconPlus size={12} />} onClick={addCheck}>Add Check</Button>
        </div>

        <div className="flex flex-col gap-3">
          {checks.map((check, idx) => (
            <Card key={idx} style={{ padding: 14, background: 'var(--bg-input)' }}>
              <div className="grid grid-2" style={{ gap: 10, marginBottom: 10 }}>
                <FormField label="Check ID" style={{ marginBottom: 0 }}>
                  <Input value={check.id} onChange={e => updateCheck(idx, 'id', e.target.value)} placeholder="pod-health" />
                </FormField>
                <div className="flex gap-2 items-end">
                  <FormField label="Severity" style={{ marginBottom: 0, flex: 1 }}>
                    <Select value={check.severity} onChange={e => updateCheck(idx, 'severity', e.target.value)}>
                      <option value="info">Info</option>
                      <option value="warning">Warning</option>
                      <option value="critical">Critical</option>
                    </Select>
                  </FormField>
                  <Button variant="ghost" size="sm" onClick={() => removeCheck(idx)} disabled={checks.length <= 1}
                    style={{ color: 'var(--danger)', marginBottom: 0 }}>
                    <IconTrash size={14} />
                  </Button>
                </div>
              </div>
              <FormField label="Description (natural language)" style={{ marginBottom: 0 }}>
                <TextArea
                  rows={2}
                  value={check.description}
                  onChange={e => updateCheck(idx, 'description', e.target.value)}
                  placeholder="Verify that all pods in the default namespace are running and not in CrashLoopBackOff..."
                />
              </FormField>
            </Card>
          ))}
        </div>
      </div>

      {createMut.isError && (
        <div style={{ color: 'var(--danger)', fontSize: 13, marginBottom: 12 }}>
          Error: {(createMut.error as Error).message}
        </div>
      )}

      <div className="flex justify-between" style={{ paddingTop: 8, borderTop: '1px solid var(--border)' }}>
        <Button variant="secondary" onClick={onClose}>Cancel</Button>
        <Button onClick={handleSubmit} disabled={createMut.isPending || !name.trim()}>
          {createMut.isPending ? 'Creating...' : 'Create Policy'}
        </Button>
      </div>
    </Modal>
  );
}

function timeAgo(dateStr: string): string {
  const diff = Date.now() - new Date(dateStr).getTime();
  const mins = Math.floor(diff / 60000);
  if (mins < 1) return 'just now';
  if (mins < 60) return `${mins}m ago`;
  const hrs = Math.floor(mins / 60);
  if (hrs < 24) return `${hrs}h ago`;
  return `${Math.floor(hrs / 24)}d ago`;
}

export default Policies;
