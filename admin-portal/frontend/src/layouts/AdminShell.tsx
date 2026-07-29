import type { ReactNode } from 'react';
import Button from '../components/ui/Button';
import Icon from '../components/ui/Icon';

interface AdminShellProps {
  children: ReactNode;
  currentPage: 'data-ingestion' | 'agent-status';
  currentTitle: string;
  onNavigate: (page: 'data-ingestion' | 'agent-status') => void;
  onLogout: () => void;
}

export default function AdminShell({ children, currentPage, currentTitle, onNavigate, onLogout }: AdminShellProps) {
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
          <button
            className={`admin-nav-item ${currentPage === 'data-ingestion' ? 'active' : ''}`}
            onClick={() => onNavigate('data-ingestion')}
            type="button"
          >
            <span className="admin-nav-icon-slot">
              <Icon name="database" />
            </span>
            <span>数据采集中心</span>
            <small>Today</small>
          </button>
          <button
            className={`admin-nav-item ${currentPage === 'agent-status' ? 'active' : ''}`}
            onClick={() => onNavigate('agent-status')}
            type="button"
          >
            <span className="admin-nav-icon-slot">
              <Icon name="activity" />
            </span>
            <span>Agent 状态</span>
            <small>Live</small>
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
          <strong>{currentTitle}</strong>
        </footer>
      </div>
    </div>
  );
}
