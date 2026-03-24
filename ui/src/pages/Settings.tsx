import { useState, useEffect } from 'react';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { api } from '../api/client';
import { Card } from '../components/Card';
import { Button } from '../components/Button';
import { FormField, Input, Select, TextArea } from '../components/FormField';
import { LoadingState } from '../components/EmptyState';
import { Badge } from '../components/Badge';

type SettingsData = Record<string, unknown>;

interface NotifSettings {
  slack?: { enabled: boolean; channel: string; webhookSecret: string };
  github?: { enabled: boolean; owner: string; repo: string; labels: string; tokenSecret: string };
  jira?: { enabled: boolean; url: string; project: string; issueType: string; credentialSecret: string };
}

interface GeneralSettings {
  defaultProvider: string;
  defaultModel: string;
  escalationThreshold: number;
  resultRetentionDays: number;
}

function Settings() {
  const queryClient = useQueryClient();
  const [mode, setMode] = useState<'form' | 'json'>('form');
  const [jsonText, setJsonText] = useState('');
  const [toast, setToast] = useState<{ type: 'success' | 'error'; msg: string } | null>(null);

  const { data: settings, isLoading } = useQuery<SettingsData>({
    queryKey: ['settings'],
    queryFn: () => api.getSettings(),
  });

  const [general, setGeneral] = useState<GeneralSettings>({
    defaultProvider: 'claude',
    defaultModel: '',
    escalationThreshold: 5,
    resultRetentionDays: 30,
  });

  const [notif, setNotif] = useState<NotifSettings>({
    slack: { enabled: false, channel: '', webhookSecret: '' },
    github: { enabled: false, owner: '', repo: '', labels: '', tokenSecret: '' },
    jira: { enabled: false, url: '', project: '', issueType: 'Bug', credentialSecret: '' },
  });

  // Populate form fields when server settings are first fetched.
  // Intentionally omitting general/notif from deps — we only want to seed
  // the form state on initial load, not re-sync on every keystroke.
  useEffect(() => {
    if (!settings) return;
    setJsonText(JSON.stringify(settings, null, 2));
    if (settings.general) setGeneral(prev => ({ ...prev, ...(settings.general as object) }));
    if (settings.notification) setNotif(prev => ({ ...prev, ...(settings.notification as object) }));
  }, [settings]);

  const mutation = useMutation({
    mutationFn: (data: SettingsData) => api.updateSettings(data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['settings'] });
      showToast('success', 'Settings saved successfully.');
    },
    onError: (err: Error) => showToast('error', err.message),
  });

  const showToast = (type: 'success' | 'error', msg: string) => {
    setToast({ type, msg });
    setTimeout(() => setToast(null), 4000);
  };

  const handleSaveForm = () => {
    mutation.mutate({ general, notification: notif });
  };

  const handleSaveJson = () => {
    try {
      const parsed = JSON.parse(jsonText);
      mutation.mutate(parsed);
    } catch {
      showToast('error', 'Invalid JSON format.');
    }
  };

  if (isLoading) return <LoadingState />;

  return (
    <div>
      <div className="page-header">
        <div>
          <div className="page-title">Settings</div>
          <div className="page-subtitle">DTM system configuration</div>
        </div>
        <div className="flex gap-2">
          <div style={{ display: 'flex', background: 'var(--bg-input)', borderRadius: 'var(--radius-md)', border: '1px solid var(--border)', overflow: 'hidden' }}>
            <button
              onClick={() => setMode('form')}
              style={{
                padding: '6px 14px', fontSize: 12, fontWeight: 500, border: 'none', cursor: 'pointer',
                background: mode === 'form' ? 'var(--accent-muted)' : 'transparent',
                color: mode === 'form' ? 'var(--accent)' : 'var(--text-tertiary)',
              }}
            >Form</button>
            <button
              onClick={() => setMode('json')}
              style={{
                padding: '6px 14px', fontSize: 12, fontWeight: 500, border: 'none', cursor: 'pointer',
                background: mode === 'json' ? 'var(--accent-muted)' : 'transparent',
                color: mode === 'json' ? 'var(--accent)' : 'var(--text-tertiary)',
              }}
            >JSON</button>
          </div>
          <Button onClick={mode === 'form' ? handleSaveForm : handleSaveJson} disabled={mutation.isPending}>
            {mutation.isPending ? 'Saving...' : 'Save Settings'}
          </Button>
        </div>
      </div>

      {/* Toast */}
      {toast && (
        <div style={{
          padding: '10px 16px', borderRadius: 'var(--radius-md)', marginBottom: 16, fontSize: 13,
          background: toast.type === 'success' ? 'var(--success-muted)' : 'var(--danger-muted)',
          color: toast.type === 'success' ? 'var(--success)' : 'var(--danger)',
          border: `1px solid ${toast.type === 'success' ? 'rgba(34,197,94,0.2)' : 'rgba(239,68,68,0.2)'}`,
        }}>
          {toast.msg}
        </div>
      )}

      {mode === 'json' ? (
        <Card>
          <FormField label="Raw Configuration">
            <TextArea
              rows={24}
              value={jsonText}
              onChange={e => setJsonText(e.target.value)}
              style={{ fontFamily: 'var(--font-mono)', fontSize: 12, lineHeight: 1.6 }}
            />
          </FormField>
        </Card>
      ) : (
        <>
          {/* General */}
          <Card style={{ marginBottom: 16 }}>
            <div className="flex items-center gap-2 mb-4">
              <div style={{ fontSize: 14, fontWeight: 600 }}>General</div>
            </div>
            <div className="grid grid-2" style={{ gap: 12 }}>
              <FormField label="Default LLM Provider">
                <Select value={general.defaultProvider} onChange={e => setGeneral({ ...general, defaultProvider: e.target.value })}>
                  <option value="claude">Claude</option>
                  <option value="gemini">Gemini</option>
                  <option value="openai">OpenAI</option>
                </Select>
              </FormField>
              <FormField label="Default Model">
                <Input value={general.defaultModel} onChange={e => setGeneral({ ...general, defaultModel: e.target.value })} placeholder="claude-sonnet-4-20250514" />
              </FormField>
              <FormField label="Escalation Threshold">
                <Input type="number" min={1} value={general.escalationThreshold} onChange={e => setGeneral({ ...general, escalationThreshold: parseInt(e.target.value) || 5 })} />
              </FormField>
              <FormField label="Result Retention (days)">
                <Input type="number" min={1} value={general.resultRetentionDays} onChange={e => setGeneral({ ...general, resultRetentionDays: parseInt(e.target.value) || 30 })} />
              </FormField>
            </div>
          </Card>

          {/* Slack */}
          <Card style={{ marginBottom: 16 }}>
            <div className="flex items-center justify-between mb-4">
              <div className="flex items-center gap-2">
                <div style={{ fontSize: 14, fontWeight: 600 }}>Slack Notifications</div>
                <Badge variant={notif.slack?.enabled ? 'pass' : 'neutral'}>
                  {notif.slack?.enabled ? 'Enabled' : 'Disabled'}
                </Badge>
              </div>
              <label className="flex items-center gap-2" style={{ fontSize: 13, cursor: 'pointer', color: 'var(--text-secondary)' }}>
                <input type="checkbox" checked={notif.slack?.enabled || false}
                  onChange={e => setNotif({ ...notif, slack: { ...notif.slack!, enabled: e.target.checked } })} />
                Enable
              </label>
            </div>
            {notif.slack?.enabled && (
              <div className="grid grid-2" style={{ gap: 12 }}>
                <FormField label="Channel">
                  <Input value={notif.slack.channel} onChange={e => setNotif({ ...notif, slack: { ...notif.slack!, channel: e.target.value } })} placeholder="#dtm-alerts" />
                </FormField>
                <FormField label="Webhook Secret Name">
                  <Input value={notif.slack.webhookSecret} onChange={e => setNotif({ ...notif, slack: { ...notif.slack!, webhookSecret: e.target.value } })} placeholder="dtm-slack-webhook" />
                </FormField>
              </div>
            )}
          </Card>

          {/* GitHub */}
          <Card style={{ marginBottom: 16 }}>
            <div className="flex items-center justify-between mb-4">
              <div className="flex items-center gap-2">
                <div style={{ fontSize: 14, fontWeight: 600 }}>GitHub Issues</div>
                <Badge variant={notif.github?.enabled ? 'pass' : 'neutral'}>
                  {notif.github?.enabled ? 'Enabled' : 'Disabled'}
                </Badge>
              </div>
              <label className="flex items-center gap-2" style={{ fontSize: 13, cursor: 'pointer', color: 'var(--text-secondary)' }}>
                <input type="checkbox" checked={notif.github?.enabled || false}
                  onChange={e => setNotif({ ...notif, github: { ...notif.github!, enabled: e.target.checked } })} />
                Enable
              </label>
            </div>
            {notif.github?.enabled && (
              <div className="grid grid-2" style={{ gap: 12 }}>
                <FormField label="Owner">
                  <Input value={notif.github.owner} onChange={e => setNotif({ ...notif, github: { ...notif.github!, owner: e.target.value } })} placeholder="my-org" />
                </FormField>
                <FormField label="Repository">
                  <Input value={notif.github.repo} onChange={e => setNotif({ ...notif, github: { ...notif.github!, repo: e.target.value } })} placeholder="my-repo" />
                </FormField>
                <FormField label="Labels (comma-separated)">
                  <Input value={notif.github.labels} onChange={e => setNotif({ ...notif, github: { ...notif.github!, labels: e.target.value } })} placeholder="dtm, alert" />
                </FormField>
                <FormField label="Token Secret Name">
                  <Input value={notif.github.tokenSecret} onChange={e => setNotif({ ...notif, github: { ...notif.github!, tokenSecret: e.target.value } })} placeholder="dtm-github-token" />
                </FormField>
              </div>
            )}
          </Card>

          {/* Jira */}
          <Card style={{ marginBottom: 16 }}>
            <div className="flex items-center justify-between mb-4">
              <div className="flex items-center gap-2">
                <div style={{ fontSize: 14, fontWeight: 600 }}>Jira Tickets</div>
                <Badge variant={notif.jira?.enabled ? 'pass' : 'neutral'}>
                  {notif.jira?.enabled ? 'Enabled' : 'Disabled'}
                </Badge>
              </div>
              <label className="flex items-center gap-2" style={{ fontSize: 13, cursor: 'pointer', color: 'var(--text-secondary)' }}>
                <input type="checkbox" checked={notif.jira?.enabled || false}
                  onChange={e => setNotif({ ...notif, jira: { ...notif.jira!, enabled: e.target.checked } })} />
                Enable
              </label>
            </div>
            {notif.jira?.enabled && (
              <div className="grid grid-2" style={{ gap: 12 }}>
                <FormField label="Jira URL">
                  <Input value={notif.jira.url} onChange={e => setNotif({ ...notif, jira: { ...notif.jira!, url: e.target.value } })} placeholder="https://mycompany.atlassian.net" />
                </FormField>
                <FormField label="Project Key">
                  <Input value={notif.jira.project} onChange={e => setNotif({ ...notif, jira: { ...notif.jira!, project: e.target.value } })} placeholder="OPS" />
                </FormField>
                <FormField label="Issue Type">
                  <Select value={notif.jira.issueType} onChange={e => setNotif({ ...notif, jira: { ...notif.jira!, issueType: e.target.value } })}>
                    <option value="Bug">Bug</option>
                    <option value="Task">Task</option>
                    <option value="Story">Story</option>
                  </Select>
                </FormField>
                <FormField label="Credential Secret Name">
                  <Input value={notif.jira.credentialSecret} onChange={e => setNotif({ ...notif, jira: { ...notif.jira!, credentialSecret: e.target.value } })} placeholder="dtm-jira-creds" />
                </FormField>
              </div>
            )}
          </Card>
        </>
      )}
    </div>
  );
}

export default Settings;
