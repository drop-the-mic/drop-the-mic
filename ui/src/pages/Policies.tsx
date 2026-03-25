import { useState } from 'react';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import yaml from 'js-yaml';
import { api } from '../api/client';
import type { ChecklistPolicy, CheckItem } from '../api/client';
import { timeAgo } from '../utils/format';
import { Combobox } from '../components/Combobox';
import { Card } from '../components/Card';
import { Badge, SeverityBadge } from '../components/Badge';
import { Button } from '../components/Button';
import { Modal } from '../components/Modal';
import { FormField, Input, TextArea, Select } from '../components/FormField';
import { LoadingState, EmptyState } from '../components/EmptyState';
import { useToast } from '../components/Toast';
import { IconPlus, IconPlay, IconTrash, IconEdit, IconBack, IconPolicy, IconClock } from '../components/Icons';
import { HelpTooltip } from '../components/HelpTooltip';

/** Returns the latest meaningful condition from a policy's status. */
function getLatestCondition(policy: ChecklistPolicy) {
  const conditions = policy.status?.conditions;
  if (!conditions || conditions.length === 0) return null;

  // Check for Degraded=True first (errors take priority)
  const degraded = conditions.find(c => c.type === 'Degraded' && c.status === 'True');
  if (degraded) return degraded;

  // Then Available=True
  const available = conditions.find(c => c.type === 'Available' && c.status === 'True');
  if (available) return available;

  return conditions[0];
}

function ConditionBadge({ policy }: { policy: ChecklistPolicy }) {
  const cond = getLatestCondition(policy);
  if (!cond) return <span style={{ color: 'var(--text-tertiary)' }}>-</span>;

  if (cond.type === 'Degraded' && cond.status === 'True') {
    return (
      <span title={cond.message}>
        <Badge variant="fail">Error</Badge>
      </span>
    );
  }

  if (cond.type === 'Available' && cond.status === 'True') {
    return <Badge variant="pass">Healthy</Badge>;
  }

  return <Badge variant="neutral">{cond.type}</Badge>;
}

function Policies() {
  const queryClient = useQueryClient();
  const { showToast } = useToast();
  const [selected, setSelected] = useState<ChecklistPolicy | null>(null);
  const [showCreate, setShowCreate] = useState(false);
  const [deleteTarget, setDeleteTarget] = useState<{ ns: string; name: string } | null>(null);

  const { data: policies = [], isLoading } = useQuery<ChecklistPolicy[]>({
    queryKey: ['policies'],
    queryFn: () => api.listPolicies(),
  });

  const runNow = useMutation({
    mutationFn: ({ ns, name }: { ns: string; name: string }) => api.runNow(ns, name),
    onSuccess: (_data, variables) => {
      showToast(`Scan triggered for ${variables.name}`);
      queryClient.invalidateQueries({ queryKey: ['policies'] });
    },
    onError: (err: Error) => {
      showToast(`Failed to trigger scan: ${err.message}`, 'error');
    },
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
                        <div className="flex items-center gap-2">
                          <ConditionBadge policy={p} />
                          {s && (s.warn > 0 || s.fail > 0) && (
                            <span style={{ fontSize: 11, color: 'var(--text-tertiary)' }}>
                              {s.pass}P{s.warn > 0 ? ` ${s.warn}W` : ''}{s.fail > 0 ? ` ${s.fail}F` : ''}
                            </span>
                          )}
                        </div>
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
                            onClick={() => setDeleteTarget({ ns: p.metadata.namespace, name: p.metadata.name })}
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

      <PolicyFormModal open={showCreate} onClose={() => setShowCreate(false)} />

      <DeletePolicyModal
        target={deleteTarget}
        onClose={() => setDeleteTarget(null)}
        onConfirm={() => {
          if (deleteTarget) {
            deletePol.mutate(deleteTarget);
            setDeleteTarget(null);
          }
        }}
        isPending={deletePol.isPending}
      />
    </div>
  );
}

/* Detail view */
function PolicyDetail({ policy, onBack }: { policy: ChecklistPolicy; onBack: () => void }) {
  const queryClient = useQueryClient();
  const { showToast } = useToast();
  const [showEdit, setShowEdit] = useState(false);
  const runNow = useMutation({
    mutationFn: () => api.runNow(policy.metadata.namespace, policy.metadata.name),
    onSuccess: () => {
      showToast(`Scan triggered for ${policy.metadata.name}`);
      queryClient.invalidateQueries({ queryKey: ['policies'] });
    },
    onError: (err: Error) => {
      showToast(`Failed to trigger scan: ${err.message}`, 'error');
    },
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
        <div className="flex gap-2">
          <Button variant="secondary" icon={<IconEdit size={14} />} onClick={() => setShowEdit(true)}>Edit</Button>
          <Button icon={<IconPlay size={14} />} onClick={() => runNow.mutate()} disabled={runNow.isPending}>
            {runNow.isPending ? 'Running...' : 'Run Now'}
          </Button>
        </div>
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

      {/* Health Status */}
      {(() => {
        const conditions = policy.status?.conditions;
        if (!conditions || conditions.length === 0) {
          return (
            <Card style={{ padding: '14px 16px', display: 'flex', alignItems: 'center', gap: 12 }}>
              <Badge variant="neutral">Pending</Badge>
              <span style={{ fontSize: 13, color: 'var(--text-tertiary)' }}>No scan has been executed yet</span>
            </Card>
          );
        }

        const degraded = conditions.find(c => c.type === 'Degraded' && c.status === 'True');
        const available = conditions.find(c => c.type === 'Available' && c.status === 'True');
        if (degraded) {
          return (
            <Card style={{ padding: '14px 16px', background: 'var(--danger-muted)', border: '1px solid rgba(239,68,68,0.2)' }}>
              <div style={{ display: 'flex', alignItems: 'center', gap: 12, marginBottom: 6 }}>
                <Badge variant="fail">Error</Badge>
                <span className="mono" style={{ fontSize: 12, color: 'var(--danger)' }}>{degraded.reason}</span>
                {degraded.lastTransitionTime && (
                  <span style={{ fontSize: 11, color: 'var(--text-tertiary)', marginLeft: 'auto' }}>{timeAgo(degraded.lastTransitionTime)}</span>
                )}
              </div>
              <div style={{ fontSize: 12, color: 'var(--danger)', opacity: 0.85, lineHeight: 1.5 }}>{degraded.message}</div>
            </Card>
          );
        }

        if (available) {
          return (
            <Card style={{ padding: '14px 16px', background: 'var(--success-muted)', border: '1px solid rgba(34,197,94,0.2)' }}>
              <div style={{ display: 'flex', alignItems: 'center', gap: 12 }}>
                <Badge variant="pass">Healthy</Badge>
                <span style={{ fontSize: 13, color: 'var(--success)' }}>{available.message}</span>
                {available.lastTransitionTime && (
                  <span style={{ fontSize: 11, color: 'var(--text-tertiary)', marginLeft: 'auto' }}>{timeAgo(available.lastTransitionTime)}</span>
                )}
              </div>
            </Card>
          );
        }

        return null;
      })()}

      <PolicyFormModal
        open={showEdit}
        onClose={() => setShowEdit(false)}
        editPolicy={policy}
      />
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

/* Policy Form Modal — used for both Create and Edit */
function PolicyFormModal({ open, onClose, editPolicy }: { open: boolean; onClose: () => void; editPolicy?: ChecklistPolicy }) {
  const isEdit = !!editPolicy;
  const queryClient = useQueryClient();
  const { showToast } = useToast();

  const { data: info } = useQuery({
    queryKey: ['info'],
    queryFn: () => api.getInfo(),
    staleTime: Infinity,
  });
  const defaultNs = info?.namespace || 'dtm-system';

  const { data: clusterNamespaces = [], isLoading: nsLoading } = useQuery({
    queryKey: ['cluster-namespaces'],
    queryFn: () => api.listNamespaces(),
    staleTime: 60_000,
  });

  const [name, setName] = useState('');
  const [namespace, setNamespace] = useState('');
  const [provider, setProvider] = useState('claude');
  const [model, setModel] = useState('');
  const [secretName, setSecretName] = useState('dtm-llm-secret');
  const [fullScan, setFullScan] = useState('0 */6 * * *');
  const [failedRescan, setFailedRescan] = useState('*/30 * * * *');
  const [targetNsList, setTargetNsList] = useState<string[]>([]);
  const [checks, setChecks] = useState<CheckItem[]>([
    { id: 'check-1', description: '', severity: 'warning' },
  ]);
  const [initialized, setInitialized] = useState(false);
  const [showPreview, setShowPreview] = useState(false);

  // Prefill form when editing
  if (isEdit && open && !initialized) {
    const p = editPolicy;
    setName(p.metadata.name);
    setNamespace(p.metadata.namespace);
    setProvider(p.spec.llm.provider);
    setModel(p.spec.llm.model || '');
    setSecretName(p.spec.llm.secretRef.name);
    setFullScan(p.spec.schedule.fullScan);
    setFailedRescan(p.spec.schedule.failedRescan || '');
    setTargetNsList(p.spec.targetNamespaces || []);
    setChecks(p.spec.checks.map(c => ({ ...c })));
    setInitialized(true);
  }

  // Reset initialized when modal closes
  if (!open && initialized) {
    setInitialized(false);
  }

  const createMut = useMutation({
    mutationFn: (policy: ChecklistPolicy) => api.createPolicy(policy),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['policies'] });
      showToast('Policy created');
      handleClose();
    },
  });

  const updateMut = useMutation({
    mutationFn: (spec: ChecklistPolicy['spec']) => api.updatePolicy(namespace, name, spec),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['policies'] });
      showToast('Policy updated');
      handleClose();
    },
  });

  const mutation = isEdit ? updateMut : createMut;

  const handleClose = () => {
    resetForm();
    onClose();
  };

  const resetForm = () => {
    setName(''); setNamespace(''); setProvider('claude'); setModel('');
    setSecretName('dtm-llm-secret'); setFullScan('0 */6 * * *'); setFailedRescan('*/30 * * * *');
    setTargetNsList([]); setChecks([{ id: 'check-1', description: '', severity: 'warning' }]);
    setInitialized(false);
    setShowPreview(false);
  };

  const buildCR = () => {
    const validChecks = checks.filter(c => c.id.trim() && c.description.trim());
    const spec: Record<string, unknown> = {
      schedule: { fullScan, ...(failedRescan ? { failedRescan } : {}) },
      llm: { provider, ...(model ? { model } : {}), secretRef: { name: secretName } },
      checks: validChecks.map(c => ({ id: c.id, description: c.description, severity: c.severity })),
    };
    if (targetNsList.length > 0) spec.targetNamespaces = targetNsList;

    return {
      apiVersion: 'dtm.dtm.io/v1alpha1',
      kind: 'ChecklistPolicy',
      metadata: { name: name || 'my-policy', namespace: namespace || defaultNs },
      spec,
    };
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

    const spec: ChecklistPolicy['spec'] = {
      schedule: { fullScan, failedRescan: failedRescan || undefined },
      llm: { provider, model: model || undefined, secretRef: { name: secretName } },
      checks: validChecks,
      targetNamespaces: targetNsList.length > 0 ? targetNsList : undefined,
    };

    if (isEdit) {
      updateMut.mutate(spec);
    } else {
      const policy: ChecklistPolicy = {
        metadata: { name, namespace: namespace || defaultNs } as ChecklistPolicy['metadata'],
        spec,
      };
      createMut.mutate(policy);
    }
  };

  return (
    <Modal open={open} onClose={handleClose} title={isEdit ? 'Edit Policy' : 'Create Policy'} width={640}>
      <div className="grid grid-2" style={{ gap: 12 }}>
        <FormField label="Name">
          <Input value={name} onChange={e => setName(e.target.value)} placeholder="my-policy" disabled={isEdit} />
        </FormField>
        <FormField label={<>Namespace <HelpTooltip text="The Kubernetes namespace where this Policy CRD will be created. Leave empty to use the namespace where DTM is deployed." /></>}>
          <Combobox
            options={clusterNamespaces}
            value={namespace ? [namespace] : []}
            onChange={v => setNamespace(v[0] || '')}
            placeholder={defaultNs}
            disabled={isEdit}
            loading={nsLoading}
          />
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
          <Input value={model} onChange={e => setModel(e.target.value)} placeholder="claude-haiku-4-5-20251001" />
        </FormField>
        <FormField label={<>Secret Name <HelpTooltip text="Name of the Kubernetes Secret containing the LLM API key. Must be in the same namespace as the Policy." /></>}>
          <Input value={secretName} onChange={e => setSecretName(e.target.value)} />
        </FormField>
      </div>

      <div className="grid grid-2" style={{ gap: 12 }}>
        <FormField label={<>Full Scan Schedule <HelpTooltip text="Cron schedule for running all checks.\ne.g. 0 */6 * * * (every 6 hours)" /></>}>
          <Input value={fullScan} onChange={e => setFullScan(e.target.value)} placeholder="0 */6 * * *" style={{ fontFamily: 'var(--font-mono)' }} />
        </FormField>
        <FormField label={<>Failed Rescan Schedule <HelpTooltip text="Cron schedule for re-checking only failed items. A shorter interval helps detect recovery faster." /></>}>
          <Input value={failedRescan} onChange={e => setFailedRescan(e.target.value)} placeholder="*/30 * * * *" style={{ fontFamily: 'var(--font-mono)' }} />
        </FormField>
      </div>

      <FormField label={<>Target Namespaces <HelpTooltip text="Namespaces the LLM will inspect. Leave empty to scan all accessible namespaces." /></>}>
        <Combobox
          options={clusterNamespaces}
          value={targetNsList}
          onChange={setTargetNsList}
          multi
          placeholder="Select namespaces..."
          loading={nsLoading}
        />
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
                  <FormField label={<>Severity <HelpTooltip text={"Importance level of this check. Used for alert escalation.\n• info: informational\n• warning: needs attention\n• critical: urgent"} /></>} style={{ marginBottom: 0, flex: 1 }}>
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
              <FormField label={<>Description <HelpTooltip text="Describe what to verify in plain language. The LLM reads this text and inspects the cluster accordingly. No need to structure it — just write naturally." /></>} style={{ marginBottom: 0 }}>
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

      {/* YAML Preview */}
      <div style={{ marginBottom: 12 }}>
        <Button
          variant="ghost"
          size="sm"
          onClick={() => setShowPreview(!showPreview)}
          style={{ fontSize: 12, color: 'var(--text-tertiary)', padding: '4px 0' }}
        >
          {showPreview ? '▲ Hide YAML Preview' : '▼ Preview YAML'}
        </Button>
        {showPreview && (
          <pre style={{
            marginTop: 8,
            padding: 14,
            background: 'var(--bg-input)',
            border: '1px solid var(--border)',
            borderRadius: 'var(--radius-md)',
            fontSize: 12,
            fontFamily: 'var(--font-mono)',
            color: 'var(--text-primary)',
            overflow: 'auto',
            maxHeight: 260,
            lineHeight: 1.6,
            whiteSpace: 'pre',
          }}>
            {yaml.dump(buildCR(), { lineWidth: -1, noRefs: true, sortKeys: false })}
          </pre>
        )}
      </div>

      {mutation.isError && (
        <div style={{ color: 'var(--danger)', fontSize: 13, marginBottom: 12 }}>
          Error: {(mutation.error as Error).message}
        </div>
      )}

      <div className="flex justify-between" style={{ paddingTop: 8, borderTop: '1px solid var(--border)' }}>
        <Button variant="secondary" onClick={handleClose}>Cancel</Button>
        <Button onClick={handleSubmit} disabled={mutation.isPending || !name.trim()}>
          {mutation.isPending ? (isEdit ? 'Saving...' : 'Creating...') : (isEdit ? 'Save Changes' : 'Create Policy')}
        </Button>
      </div>
    </Modal>
  );
}

/* Delete Confirmation Modal */
function DeletePolicyModal({ target, onClose, onConfirm, isPending }: {
  target: { ns: string; name: string } | null;
  onClose: () => void;
  onConfirm: () => void;
  isPending: boolean;
}) {
  return (
    <Modal open={!!target} onClose={onClose} title="Delete Policy" width={480}>
      <div style={{ marginBottom: 16 }}>
        <div style={{ fontSize: 14, color: 'var(--text-primary)', marginBottom: 12 }}>
          Are you sure you want to delete <strong>{target?.name}</strong>?
        </div>
        <div style={{
          padding: '10px 14px',
          background: 'var(--danger-muted)',
          border: '1px solid rgba(239,68,68,0.2)',
          borderRadius: 'var(--radius-sm)',
          fontSize: 13,
          color: 'var(--danger)',
          lineHeight: 1.5,
        }}>
          All scan history (ChecklistResults) for this policy will be permanently deleted. This action cannot be undone.
        </div>
      </div>
      <div className="flex justify-between" style={{ paddingTop: 8, borderTop: '1px solid var(--border)' }}>
        <Button variant="secondary" onClick={onClose}>Cancel</Button>
        <Button variant="danger" onClick={onConfirm} disabled={isPending}>
          {isPending ? 'Deleting...' : 'Delete Policy'}
        </Button>
      </div>
    </Modal>
  );
}

export default Policies;
