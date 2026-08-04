import { Database, Radio, ScanLine } from 'lucide-react';
import {
  Sidebar,
  SidebarContent,
  SidebarFooter,
  SidebarGroup,
  SidebarGroupLabel,
  SidebarHeader,
  SidebarMenu,
  SidebarMenuButton,
  SidebarMenuItem,
  useSidebar
} from '../ui/sidebar';

export type AdminPage = 'data-ingestion' | 'monitoring';

interface AdminNavigationProps {
  currentPage: AdminPage;
  onNavigate: (page: AdminPage) => void;
  showLabels?: boolean;
}

const navigationItems = [
  {
    key: 'data-ingestion',
    label: '采集中心',
    meta: 'Today',
    icon: Database
  },
  {
    key: 'monitoring',
    label: '监控中心',
    meta: 'Live',
    icon: ScanLine
  }
] satisfies Array<{
  key: AdminPage;
  label: string;
  meta: string;
  icon: typeof Database;
}>;

export function AdminNavigation({ currentPage, onNavigate, showLabels }: AdminNavigationProps) {
  const { open } = useSidebar();
  const labelsVisible = showLabels ?? open;

  return (
    <nav aria-label='管理后台菜单'>
      <SidebarMenu>
        {navigationItems.map((item) => {
          const ItemIcon = item.icon;
          return (
            <SidebarMenuItem key={item.key}>
              <SidebarMenuButton
                active={currentPage === item.key}
                aria-current={currentPage === item.key ? 'page' : undefined}
                aria-label={`${item.label} ${item.meta}`}
                expanded={labelsVisible}
                onClick={() => onNavigate(item.key)}
                title={labelsVisible ? undefined : item.label}
              >
                <ItemIcon aria-hidden='true' className='size-4 shrink-0' />
                {labelsVisible ? (
                  <>
                    <span className='min-w-0 flex-1 truncate'>{item.label}</span>
                    <span className='text-[0.68rem] opacity-60'>{item.meta}</span>
                  </>
                ) : null}
              </SidebarMenuButton>
            </SidebarMenuItem>
          );
        })}
      </SidebarMenu>
    </nav>
  );
}

type AppSidebarProps = Omit<AdminNavigationProps, 'showLabels'>;

export default function AppSidebar({ currentPage, onNavigate }: AppSidebarProps) {
  const { open } = useSidebar();

  return (
    <Sidebar aria-label='管理后台侧边栏'>
      <SidebarHeader>
        <div className='flex min-w-0 items-center gap-3'>
          <span className='flex size-9 shrink-0 items-center justify-center rounded-lg bg-sidebar-primary font-serif text-lg font-semibold text-sidebar-primary-foreground shadow-sm'>
            潮
          </span>
          {open ? (
            <div className='min-w-0'>
              <strong className='block truncate text-sm'>观潮家 Admin</strong>
              <span className='block truncate text-xs text-sidebar-foreground/55'>
                Market intelligence ops
              </span>
            </div>
          ) : null}
        </div>
      </SidebarHeader>
      <SidebarContent>
        <SidebarGroup>
          <SidebarGroupLabel>Workspace</SidebarGroupLabel>
          <AdminNavigation currentPage={currentPage} onNavigate={onNavigate} />
        </SidebarGroup>
      </SidebarContent>
      <SidebarFooter>
        <div className='flex items-center gap-3 text-xs text-sidebar-foreground/65'>
          <span className='relative flex size-8 shrink-0 items-center justify-center rounded-md bg-sidebar-accent'>
            <Radio aria-hidden='true' className='size-4' />
            <span
              aria-hidden='true'
              className='absolute right-1 top-1 size-1.5 rounded-full bg-sidebar-status'
            />
          </span>
          <span className={open ? 'min-w-0' : 'sr-only'}>
            <span className='block font-medium text-sidebar-foreground'>Admin Backend</span>
            <span className='block truncate'>Connected</span>
          </span>
        </div>
      </SidebarFooter>
    </Sidebar>
  );
}
