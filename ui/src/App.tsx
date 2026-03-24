import { BrowserRouter, Routes, Route, NavLink } from 'react-router-dom';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { ErrorBoundary } from './components/ErrorBoundary';
import { IconDashboard, IconPolicy, IconResult, IconSettings } from './components/Icons';
import Dashboard from './pages/Dashboard';
import Policies from './pages/Policies';
import Results from './pages/Results';
import Settings from './pages/Settings';
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

function App() {
  return (
    <QueryClientProvider client={queryClient}>
      <BrowserRouter>
        <div className="app-layout">
          <nav className="sidebar">
            <div className="sidebar-header">
              <div className="sidebar-logo">
                <img src="/logo.svg" alt="DTM" width={28} height={28} />
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
      </BrowserRouter>
    </QueryClientProvider>
  );
}

export default App;
