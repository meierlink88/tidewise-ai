import { Layout, Menu, Input, Typography } from 'antd';
import { useEffect, useState } from 'react';
import SchedulerSettings from './pages/SchedulerSettings';
import './styles/app.css';

const tokenStorageKey = 'tidewise_admin_token';

export default function App() {
  const [token, setToken] = useState(() => localStorage.getItem(tokenStorageKey) ?? '');

  useEffect(() => {
    if (token) {
      localStorage.setItem(tokenStorageKey, token);
    }
  }, [token]);

  return (
    <Layout className="app-shell">
      <Layout.Sider width={220} className="app-sidebar">
        <Typography.Title level={3}>观潮家</Typography.Title>
        <Menu
          theme="dark"
          mode="inline"
          selectedKeys={['scheduler']}
          items={[{ key: 'scheduler', label: '调度器设置' }]}
        />
      </Layout.Sider>
      <Layout>
        <Layout.Header className="app-header">
          <Input.Password
            className="token-input"
            placeholder="Admin Token"
            value={token}
            onChange={(event) => setToken(event.target.value)}
          />
        </Layout.Header>
        <Layout.Content className="app-content">
          <SchedulerSettings token={token} />
        </Layout.Content>
      </Layout>
    </Layout>
  );
}
