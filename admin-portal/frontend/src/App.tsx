import { useState } from 'react';
import { useQueryClient } from '@tanstack/react-query';
import AdminShell from './layouts/AdminShell';
import AdminLogin from './pages/AdminLogin';
import DataIngestionCenter from './pages/DataIngestionCenter';
import AppProviders from './providers/app-providers';
import type { AdminPage } from './components/admin/app-sidebar';

const tokenStorageKey = 'tidewise_admin_token';

function AdminApp() {
  const queryClient = useQueryClient();
  const [token, setToken] = useState(() => localStorage.getItem(tokenStorageKey) ?? '');
  const currentPage: AdminPage = 'data-ingestion';

  const handleLogin = (nextToken: string) => {
    queryClient.clear();
    localStorage.setItem(tokenStorageKey, nextToken);
    setToken(nextToken);
  };

  const handleLogout = () => {
    queryClient.clear();
    localStorage.removeItem(tokenStorageKey);
    setToken('');
  };

  if (!token) {
    return <AdminLogin onLogin={handleLogin} tokenHint='local-admin-token' />;
  }

  return (
    <AdminShell
      currentPage={currentPage}
      currentTitle={pageTitle(currentPage)}
      onNavigate={() => undefined}
      onLogout={handleLogout}
    >
      <DataIngestionCenter token={token} />
    </AdminShell>
  );
}

function pageTitle(_page: AdminPage): string {
  return '数据采集中心';
}

export default function App() {
  return (
    <AppProviders>
      <AdminApp />
    </AppProviders>
  );
}
