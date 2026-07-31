import { useState } from 'react';
import AdminShell from './layouts/AdminShell';
import AdminLogin from './pages/AdminLogin';
import DataIngestionCenter from './pages/DataIngestionCenter';
import AgentStatusMonitor from './pages/AgentStatusMonitor';
import AppProviders from './providers/app-providers';

const tokenStorageKey = 'tidewise_admin_token';

function AdminApp() {
  const [token, setToken] = useState(() => localStorage.getItem(tokenStorageKey) ?? '');
  const [currentPage, setCurrentPage] = useState<'data-ingestion' | 'agent-status'>(
    'data-ingestion'
  );

  const handleLogin = (nextToken: string) => {
    localStorage.setItem(tokenStorageKey, nextToken);
    setToken(nextToken);
  };

  const handleLogout = () => {
    localStorage.removeItem(tokenStorageKey);
    setToken('');
    setCurrentPage('data-ingestion');
  };

  if (!token) {
    return <AdminLogin onLogin={handleLogin} tokenHint='local-admin-token' />;
  }

  return (
    <AdminShell
      currentPage={currentPage}
      currentTitle={currentPage === 'agent-status' ? 'Agent 状态监控' : '数据采集中心'}
      onNavigate={setCurrentPage}
      onLogout={handleLogout}
    >
      {currentPage === 'agent-status' ? (
        <AgentStatusMonitor token={token} />
      ) : (
        <DataIngestionCenter token={token} />
      )}
    </AdminShell>
  );
}

export default function App() {
  return (
    <AppProviders>
      <AdminApp />
    </AppProviders>
  );
}
