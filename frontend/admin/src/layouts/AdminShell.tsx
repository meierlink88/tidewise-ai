import type { ReactNode } from 'react';
import Button from '../components/ui/Button';

interface AdminShellProps {
  children: ReactNode;
  onLogout: () => void;
}

export default function AdminShell({ children, onLogout }: AdminShellProps) {
  return (
    <div className="admin-shell">
      <aside className="admin-sidebar">
        <div className="admin-brand">观潮家</div>
        <div className="admin-section">Workspace</div>
        <nav className="admin-nav" aria-label="管理后台菜单">
          <button className="admin-nav-item active" type="button">
            <span className="nav-dot" />
            <span>调度器设置</span>
          </button>
        </nav>
      </aside>
      <div className="admin-main">
        <header className="admin-header">
          <span className="admin-header-kicker">Admin Console</span>
          <Button variant="secondary" onClick={onLogout}>退出登录</Button>
        </header>
        <main className="admin-content">{children}</main>
      </div>
    </div>
  );
}
