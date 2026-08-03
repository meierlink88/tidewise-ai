import { type ReactNode } from 'react';
import AppHeader from '../components/admin/app-header';
import AppSidebar, { type AdminPage } from '../components/admin/app-sidebar';
import SkipToMain from '../components/admin/skip-to-main';
import { SidebarInset, SidebarProvider } from '../components/ui/sidebar';

interface AdminShellProps {
  children: ReactNode;
  currentPage: AdminPage;
  currentTitle: string;
  onNavigate: (page: AdminPage) => void;
  onLogout: () => void;
}

export default function AdminShell({
  children,
  currentPage,
  currentTitle,
  onNavigate,
  onLogout
}: AdminShellProps) {
  return (
    <SidebarProvider>
      <SkipToMain />
      <AppSidebar currentPage={currentPage} onNavigate={onNavigate} />
      <SidebarInset>
        <AppHeader
          currentPage={currentPage}
          currentTitle={currentTitle}
          onLogout={onLogout}
          onNavigate={onNavigate}
        />
        <main
          className='min-h-0 flex-1 overflow-hidden bg-background p-4 pt-5 md:p-6 md:pt-6'
          id='admin-main-content'
          tabIndex={-1}
        >
          {children}
        </main>
      </SidebarInset>
    </SidebarProvider>
  );
}
