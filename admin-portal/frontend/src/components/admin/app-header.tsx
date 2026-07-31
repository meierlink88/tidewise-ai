import { LogOut, Menu } from 'lucide-react';
import { useState } from 'react';
import ThemeToggle from './theme-toggle';
import { AdminNavigation, type AdminPage } from './app-sidebar';
import { Button } from '../ui/Button';
import {
  Sheet,
  SheetContent,
  SheetDescription,
  SheetTitle,
  SheetTrigger
} from '../ui/sheet';
import { SidebarTrigger } from '../ui/sidebar';

interface AppHeaderProps {
  currentPage: AdminPage;
  currentTitle: string;
  onLogout: () => void;
  onNavigate: (page: AdminPage) => void;
}

export default function AppHeader({
  currentPage,
  currentTitle,
  onLogout,
  onNavigate
}: AppHeaderProps) {
  const [mobileNavigationOpen, setMobileNavigationOpen] = useState(false);

  return (
    <header className='flex h-16 shrink-0 items-center justify-between gap-3 border-b border-border bg-background/95 px-4 backdrop-blur md:px-6'>
      <div className='flex min-w-0 items-center gap-2'>
        <SidebarTrigger className='hidden md:inline-flex' />
        <Sheet open={mobileNavigationOpen} onOpenChange={setMobileNavigationOpen}>
          <SheetTrigger asChild>
            <Button aria-label='打开导航菜单' className='md:hidden' size='icon' variant='ghost'>
              <Menu aria-hidden='true' className='size-5' />
            </Button>
          </SheetTrigger>
          <SheetContent aria-label='管理后台导航' className='gap-6 pt-16'>
            <SheetTitle className='sr-only'>管理后台导航</SheetTitle>
            <SheetDescription className='sr-only'>选择管理后台页面</SheetDescription>
            <div className='flex items-center gap-3 border-b border-sidebar-border pb-5'>
              <span className='flex size-9 items-center justify-center rounded-lg bg-sidebar-primary font-serif text-lg font-semibold text-sidebar-primary-foreground'>
                潮
              </span>
              <div>
                <strong className='block text-sm'>观潮家 Admin</strong>
                <span className='block text-xs text-sidebar-foreground/55'>
                  Market intelligence ops
                </span>
              </div>
            </div>
            <AdminNavigation
              currentPage={currentPage}
              onNavigate={(page) => {
                onNavigate(page);
                setMobileNavigationOpen(false);
              }}
              showLabels
            />
          </SheetContent>
        </Sheet>
        <div className='min-w-0'>
          <span className='block text-[0.68rem] font-semibold tracking-[0.14em] text-muted-foreground uppercase'>
            Admin Console
          </span>
          <h1 className='truncate text-sm font-semibold md:text-base'>{currentTitle}</h1>
        </div>
      </div>
      <div className='flex shrink-0 items-center gap-1'>
        <ThemeToggle />
        <Button aria-label='退出登录' onClick={onLogout} size='sm' variant='outline'>
          <LogOut aria-hidden='true' className='size-4' />
          <span className='hidden sm:inline'>退出登录</span>
        </Button>
      </div>
    </header>
  );
}
