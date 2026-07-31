import { type ReactNode, useState } from 'react';
import ThemeToggle from '../components/admin/theme-toggle';
import Icon from '../components/ui/Icon';
import { Button } from '../components/ui/Button';
import {
  Sheet,
  SheetContent,
  SheetDescription,
  SheetTitle,
  SheetTrigger
} from '../components/ui/sheet';

interface AdminShellProps {
  children: ReactNode;
  currentPage: 'data-ingestion' | 'agent-status';
  currentTitle: string;
  onNavigate: (page: 'data-ingestion' | 'agent-status') => void;
  onLogout: () => void;
}

export default function AdminShell({
  children,
  currentPage,
  currentTitle,
  onNavigate,
  onLogout
}: AdminShellProps) {
  const [mobileNavigationOpen, setMobileNavigationOpen] = useState(false);

  return (
    <div className='admin-shell'>
      <aside className='admin-sidebar admin-sidebar-desktop'>
        <SidebarContent currentPage={currentPage} onNavigate={onNavigate} />
      </aside>
      <div className='admin-main'>
        <header className='admin-header'>
          <div className='admin-header-leading'>
            <Sheet open={mobileNavigationOpen} onOpenChange={setMobileNavigationOpen}>
              <SheetTrigger asChild>
                <Button
                  aria-label='打开导航菜单'
                  className='admin-mobile-menu-trigger'
                  size='icon'
                  variant='outline'
                >
                  <Icon name='menu' />
                </Button>
              </SheetTrigger>
              <SheetContent aria-label='管理后台导航' className='admin-sidebar-mobile'>
                <SheetTitle className='sr-only'>管理后台导航</SheetTitle>
                <SheetDescription className='sr-only'>选择管理后台页面</SheetDescription>
                <SidebarContent
                  currentPage={currentPage}
                  onNavigate={(page) => {
                    onNavigate(page);
                    setMobileNavigationOpen(false);
                  }}
                />
              </SheetContent>
            </Sheet>
            <div>
              <span className='admin-header-kicker'>Admin Console</span>
              <h1 className='admin-header-title'>{currentTitle}</h1>
            </div>
          </div>
          <div className='admin-header-actions'>
            <ThemeToggle />
            <Button aria-label='退出登录' variant='outline' onClick={onLogout}>
              <Icon name='log-out' />
              <span className='admin-logout-label'>退出登录</span>
            </Button>
          </div>
        </header>
        <main className='admin-content'>{children}</main>
        <footer className='admin-footer'>
          <span>LOCAL ADMIN</span>
          <strong>{currentTitle}</strong>
        </footer>
      </div>
    </div>
  );
}

function SidebarContent({
  currentPage,
  onNavigate
}: Pick<AdminShellProps, 'currentPage' | 'onNavigate'>) {
  return (
    <>
      <div className='admin-brand'>
        <span className='admin-brand-mark'>潮</span>
        <div>
          <strong>观潮家 Admin</strong>
          <span>Market intelligence ops</span>
        </div>
      </div>
      <div className='admin-section'>WORKSPACE</div>
      <nav className='admin-nav' aria-label='管理后台菜单'>
        <button
          aria-current={currentPage === 'data-ingestion' ? 'page' : undefined}
          className={`admin-nav-item ${currentPage === 'data-ingestion' ? 'active' : ''}`}
          onClick={() => onNavigate('data-ingestion')}
          type='button'
        >
          <span className='admin-nav-icon-slot'>
            <Icon name='database' />
          </span>
          <span>数据采集中心</span>
          <small>Today</small>
        </button>
        <button
          aria-current={currentPage === 'agent-status' ? 'page' : undefined}
          className={`admin-nav-item ${currentPage === 'agent-status' ? 'active' : ''}`}
          onClick={() => onNavigate('agent-status')}
          type='button'
        >
          <span className='admin-nav-icon-slot'>
            <Icon name='activity' />
          </span>
          <span>Agent 状态</span>
          <small>Live</small>
        </button>
      </nav>
      <div className='admin-sidebar-foot'>
        <span>SYSTEM</span>
        <strong>Admin Backend connected</strong>
      </div>
    </>
  );
}
