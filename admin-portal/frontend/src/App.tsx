import { useState } from 'react';
import AdminShell from './layouts/AdminShell';
import AdminLogin from './pages/AdminLogin';
import DataIngestionCenter from './pages/DataIngestionCenter';
import AgentStatusMonitor from './pages/AgentStatusMonitor';
import MonitoringCenter from './pages/MonitoringCenter';
import AppProviders from './providers/app-providers';
import type { AdminPage } from './components/admin/app-sidebar';

const tokenStorageKey = 'tidewise_admin_token';

function AdminApp() {
  const [token, setToken] = useState(() => localStorage.getItem(tokenStorageKey) ?? '');
  const [currentPage, setCurrentPage] = useState<AdminPage>('data-ingestion');

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
      currentTitle={pageTitle(currentPage)}
      onNavigate={setCurrentPage}
      onLogout={handleLogout}
    >
      {currentPage === 'monitoring' ? (
        <MonitoringCenter token={token} />
      ) : currentPage === 'agent-status' ? (
        <AgentStatusMonitor token={token} />
      ) : (
        <DataIngestionCenter token={token} />
      )}
    </AdminShell>
  );
}

function pageTitle(page: AdminPage): string {
  if (page === 'monitoring') return '监控中心';
  if (page === 'agent-status') return 'Agent 状态';
  return '数据采集中心';
}

export default function App() {
  return (
    <AppProviders>
      <AdminApp />
    </AppProviders>
  );
}
