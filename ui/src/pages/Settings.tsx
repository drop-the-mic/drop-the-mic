import { useState, useEffect } from 'react';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { api } from '../api/client';

function Settings() {
  const queryClient = useQueryClient();
  const { data: settings } = useQuery({
    queryKey: ['settings'],
    queryFn: () => api.getSettings(),
  });

  const [formData, setFormData] = useState<string>('');

  useEffect(() => {
    if (settings) {
      setFormData(JSON.stringify(settings, null, 2));
    }
  }, [settings]);

  const mutation = useMutation({
    mutationFn: (data: Record<string, unknown>) => api.updateSettings(data),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: ['settings'] }),
  });

  const handleSave = () => {
    try {
      const parsed = JSON.parse(formData);
      mutation.mutate(parsed);
    } catch {
      alert('Invalid JSON');
    }
  };

  return (
    <div>
      <div className="page-header">
        <h2>Settings</h2>
        <button
          className="btn btn-primary"
          onClick={handleSave}
          disabled={mutation.isPending}
        >
          {mutation.isPending ? 'Saving...' : 'Save'}
        </button>
      </div>

      <div className="card">
        <div className="form-group">
          <label>DTM Configuration (JSON)</label>
          <textarea
            rows={20}
            value={formData}
            onChange={(e) => setFormData(e.target.value)}
            style={{ fontFamily: 'monospace', fontSize: 13 }}
          />
        </div>
        {mutation.isSuccess && (
          <div style={{ color: 'var(--success)', fontSize: 13 }}>Settings saved successfully.</div>
        )}
        {mutation.isError && (
          <div style={{ color: 'var(--danger)', fontSize: 13 }}>
            Error: {(mutation.error as Error).message}
          </div>
        )}
      </div>
    </div>
  );
}

export default Settings;
