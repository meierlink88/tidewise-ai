import type { ReactNode } from 'react';
import Button from '../components/ui/Button';
import Icon from '../components/ui/Icon';

interface AdminShellProps {
  children: ReactNode;
  currentTitle: string;
  onLogout: () => void;
}

export default function AdminShell({ children, currentTitle, onLogout }: AdminShellProps) {
  return (
    <div className="admin-shell">
      <aside className="admin-sidebar">
        <div className="admin-brand">
          <span className="admin-brand-mark">M</span>
          <div>
            <strong>观潮家 Admin</strong>
            <span>Market intelligence ops</span>
          </div>
        </div>
        <div className="admin-section">WORKSPACE</div>
        <nav className="admin-nav" aria-label="管理后台菜单">
          <button className="admin-nav-item active" type="button">
            <span className="admin-nav-icon-slot">
              <Icon name="database" />
            </span>
            <span>数据采集中心</span>
            <small>Today</small>
          </button>
        </nav>
        <div className="admin-sidebar-foot">
          <span>SYSTEM NOTE</span>
          <strong>Flat surfaces, precise rhythm</strong>
        </div>
      </aside>
      <div className="admin-main">
        <header className="admin-header">
          <div>
            <span className="admin-header-kicker">Admin Console</span>
            <h1 className="admin-header-title">{currentTitle}</h1>
          </div>
          <Button variant="secondary" onClick={onLogout}>
            <Icon name="log-out" />
            退出登录
          </Button>
        </header>
        <main className="admin-content">{children}</main>
        <footer className="admin-footer">
          <span>LOCAL ADMIN</span>
          <strong>数据采集中心</strong>
        </footer>
      </div>
    </div>
  );
}
