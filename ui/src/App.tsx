import { useEffect, useState } from 'react';
import { BrowserRouter, Routes, Route, NavLink, useNavigate, useLocation } from 'react-router-dom';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { ErrorBoundary } from './components/ErrorBoundary';
import { IconDashboard, IconPolicy, IconResult, IconSettings } from './components/Icons';
import Dashboard from './pages/Dashboard';
import Policies from './pages/Policies';
import Results from './pages/Results';
import Settings from './pages/Settings';
import Login from './pages/Login';
import { isAuthenticated, logout, getToken } from './api/client';
import './App.css';

const queryClient = new QueryClient({
  defaultOptions: {
    queries: { refetchInterval: 30000, retry: 1, staleTime: 10000 },
  },
});

const navItems = [
  { to: '/', icon: <IconDashboard />, label: 'Dashboard' },
  { to: '/policies', icon: <IconPolicy />, label: 'Policies' },
  { to: '/results', icon: <IconResult />, label: 'Results' },
  { to: '/settings', icon: <IconSettings />, label: 'Settings' },
];

function AuthGuard({ children }: { children: React.ReactNode }) {
  const navigate = useNavigate();
  const location = useLocation();
  const [checking, setChecking] = useState(true);
  const [authenticated, setAuthenticated] = useState(false);

  useEffect(() => {
    const checkAuth = async () => {
      if (!isAuthenticated()) {
        navigate('/login', { replace: true });
        return;
      }

      try {
        const resp = await fetch('/api/v1/auth/check', {
          headers: { Authorization: `Bearer ${getToken()}` },
        });
        if (resp.status === 401) {
          localStorage.removeItem('dtm_token');
          navigate('/login', { replace: true });
          return;
        }
        setAuthenticated(true);
      } catch {
        navigate('/login', { replace: true });
      } finally {
        setChecking(false);
      }
    };

    checkAuth();
  }, [navigate, location.pathname]);

  if (checking) {
    return null;
  }

  if (!authenticated) {
    return null;
  }

  return <>{children}</>;
}

function AppLayout() {
  return (
    <AuthGuard>
      <div className="app-layout">
        <nav className="sidebar">
          <div className="sidebar-header">
            <div className="sidebar-logo">
              <img src="/logo.png" alt="DTM" width={28} height={28} style={{ borderRadius: 6 }} />
              <div>
                <div className="sidebar-title">DTM</div>
                <div className="sidebar-subtitle">Drop The Mic</div>
              </div>
            </div>
          </div>

          <div className="sidebar-nav">
            {navItems.map(item => (
              <NavLink
                key={item.to}
                to={item.to}
                end={item.to === '/'}
                className={({ isActive }) => `nav-item ${isActive ? 'nav-item--active' : ''}`}
              >
                <span className="nav-icon">{item.icon}</span>
                <span className="nav-label">{item.label}</span>
              </NavLink>
            ))}
          </div>

          <div className="sidebar-footer">
            <button
              onClick={logout}
              className="nav-item"
              style={{ width: '100%', border: 'none', background: 'none', cursor: 'pointer', textAlign: 'left' }}
            >
              <span className="nav-icon">
                <svg width="16" height="16" viewBox="0 0 16 16" fill="none" stroke="currentColor" strokeWidth="1.5" strokeLinecap="round" strokeLinejoin="round">
                  <path d="M6 14H3.33A1.33 1.33 0 0 1 2 12.67V3.33A1.33 1.33 0 0 1 3.33 2H6" />
                  <polyline points="10.67 11.33 14 8 10.67 4.67" />
                  <line x1="14" y1="8" x2="6" y2="8" />
                </svg>
              </span>
              <span className="nav-label">Logout</span>
            </button>
            <div className="sidebar-version">v0.1.0</div>
          </div>
        </nav>

        <main className="main-content">
          <ErrorBoundary>
            <Routes>
              <Route path="/" element={<Dashboard />} />
              <Route path="/policies" element={<Policies />} />
              <Route path="/results" element={<Results />} />
              <Route path="/settings" element={<Settings />} />
            </Routes>
          </ErrorBoundary>
        </main>
      </div>
    </AuthGuard>
  );
}

function App() {
  return (
    <QueryClientProvider client={queryClient}>
      <BrowserRouter>
        <Routes>
          <Route path="/login" element={<Login />} />
          <Route path="/*" element={<AppLayout />} />
        </Routes>
      </BrowserRouter>
    </QueryClientProvider>
  );
}

export default App;
