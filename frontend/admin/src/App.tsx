import { useState } from 'react';
import AdminShell from './layouts/AdminShell';
import AdminLogin from './pages/AdminLogin';
import SchedulerSettings from './pages/SchedulerSettings';
import './styles/app.css';

const tokenStorageKey = 'tidewise_admin_token';

export default function App() {
  const [token, setToken] = useState(() => localStorage.getItem(tokenStorageKey) ?? '');

  const handleLogin = (nextToken: string) => {
    localStorage.setItem(tokenStorageKey, nextToken);
    setToken(nextToken);
  };

  const handleLogout = () => {
    localStorage.removeItem(tokenStorageKey);
    setToken('');
  };

  if (!token) {
    return <AdminLogin onLogin={handleLogin} tokenHint="local-admin-token" />;
  }

  return (
    <AdminShell onLogout={handleLogout}>
      <SchedulerSettings token={token} />
    </AdminShell>
  );
}
