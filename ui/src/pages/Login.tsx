import { useState, type FormEvent } from 'react';
import { useNavigate } from 'react-router-dom';
import { api, setToken } from '../api/client';

function Login() {
  const navigate = useNavigate();
  const [username, setUsername] = useState('');
  const [password, setPassword] = useState('');
  const [error, setError] = useState('');
  const [loading, setLoading] = useState(false);

  const handleSubmit = async (e: FormEvent) => {
    e.preventDefault();
    setError('');
    setLoading(true);

    try {
      const resp = await api.login(username, password);
      setToken(resp.token);
      navigate('/', { replace: true });
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Login failed');
    } finally {
      setLoading(false);
    }
  };

  return (
    <div style={{
      display: 'flex',
      alignItems: 'center',
      justifyContent: 'center',
      minHeight: '100vh',
      background: 'var(--bg-dark)',
    }}>
      <form onSubmit={handleSubmit} style={{
        width: 360,
        padding: 32,
        background: 'var(--bg-card)',
        borderRadius: 'var(--radius-lg)',
        border: '1px solid var(--border)',
        boxShadow: 'var(--shadow-lg)',
      }}>
        <div style={{ textAlign: 'center', marginBottom: 28 }}>
          <img
            src="/logo.png"
            alt="DTM"
            width={48}
            height={48}
            style={{ borderRadius: 12, marginBottom: 12 }}
          />
          <div style={{
            fontSize: 20,
            fontWeight: 700,
            color: 'var(--text-primary)',
          }}>DTM</div>
          <div style={{
            fontSize: 12,
            color: 'var(--text-tertiary)',
            marginTop: 2,
          }}>Drop The Mic</div>
        </div>

        {error && (
          <div style={{
            padding: '10px 14px',
            marginBottom: 16,
            background: 'var(--danger-muted)',
            color: 'var(--danger)',
            borderRadius: 'var(--radius-sm)',
            fontSize: 13,
            border: '1px solid rgba(239, 68, 68, 0.2)',
          }}>{error}</div>
        )}

        <div style={{ marginBottom: 14 }}>
          <label style={{
            display: 'block',
            fontSize: 12,
            fontWeight: 500,
            color: 'var(--text-secondary)',
            marginBottom: 6,
          }}>Username</label>
          <input
            className="dtm-input"
            type="text"
            value={username}
            onChange={e => setUsername(e.target.value)}
            autoComplete="username"
            autoFocus
            required
          />
        </div>

        <div style={{ marginBottom: 22 }}>
          <label style={{
            display: 'block',
            fontSize: 12,
            fontWeight: 500,
            color: 'var(--text-secondary)',
            marginBottom: 6,
          }}>Password</label>
          <input
            className="dtm-input"
            type="password"
            value={password}
            onChange={e => setPassword(e.target.value)}
            autoComplete="current-password"
            required
          />
        </div>

        <button
          type="submit"
          disabled={loading}
          style={{
            width: '100%',
            padding: '10px 16px',
            fontSize: 14,
            fontWeight: 600,
            color: '#fff',
            background: 'var(--accent)',
            border: 'none',
            borderRadius: 'var(--radius-md)',
            cursor: loading ? 'not-allowed' : 'pointer',
            opacity: loading ? 0.6 : 1,
            transition: 'var(--transition-fast)',
          }}
        >
          {loading ? 'Signing in...' : 'Sign In'}
        </button>
      </form>
    </div>
  );
}

export default Login;
