import { PanelLeftClose, PanelLeftOpen } from 'lucide-react';
import * as React from 'react';
import { cn } from '../../lib/utils';
import { Button } from './Button';

interface SidebarContextValue {
  open: boolean;
  setOpen: React.Dispatch<React.SetStateAction<boolean>>;
  toggleSidebar: () => void;
}

const SidebarContext = React.createContext<SidebarContextValue | null>(null);

function useSidebar() {
  const context = React.useContext(SidebarContext);
  if (!context) {
    throw new Error('useSidebar must be used within a SidebarProvider.');
  }
  return context;
}

function SidebarProvider({
  children,
  defaultOpen = true
}: React.PropsWithChildren<{ defaultOpen?: boolean }>) {
  const [open, setOpen] = React.useState(defaultOpen);
  const toggleSidebar = React.useCallback(() => setOpen((value) => !value), []);

  React.useEffect(() => {
    const handleKeyDown = (event: KeyboardEvent) => {
      if (event.key.toLowerCase() === 'b' && (event.metaKey || event.ctrlKey)) {
        event.preventDefault();
        toggleSidebar();
      }
    };

    window.addEventListener('keydown', handleKeyDown);
    return () => window.removeEventListener('keydown', handleKeyDown);
  }, [toggleSidebar]);

  const value = React.useMemo(() => ({ open, setOpen, toggleSidebar }), [open, toggleSidebar]);

  return (
    <SidebarContext.Provider value={value}>
      <div
        className='group/sidebar-wrapper flex min-h-svh w-full bg-background'
        data-slot='sidebar-wrapper'
      >
        {children}
      </div>
    </SidebarContext.Provider>
  );
}

const Sidebar = React.forwardRef<HTMLElement, React.ComponentPropsWithoutRef<'aside'>>(
  ({ className, ...props }, ref) => {
    const { open } = useSidebar();

    return (
      <aside
        className={cn(
          'hidden h-svh shrink-0 flex-col border-r border-sidebar-border bg-sidebar text-sidebar-foreground transition-[width] duration-200 ease-linear md:flex',
          open ? 'w-64' : 'w-16',
          className
        )}
        data-state={open ? 'expanded' : 'collapsed'}
        data-slot='sidebar'
        ref={ref}
        {...props}
      />
    );
  }
);
Sidebar.displayName = 'Sidebar';

function SidebarHeader({ className, ...props }: React.ComponentProps<'div'>) {
  return (
    <div
      className={cn('flex min-h-16 items-center border-b border-sidebar-border p-3', className)}
      data-slot='sidebar-header'
      {...props}
    />
  );
}

function SidebarContent({ className, ...props }: React.ComponentProps<'div'>) {
  return (
    <div
      className={cn('flex min-h-0 flex-1 flex-col gap-2 overflow-auto p-2', className)}
      data-slot='sidebar-content'
      {...props}
    />
  );
}

function SidebarFooter({ className, ...props }: React.ComponentProps<'div'>) {
  return (
    <div
      className={cn('border-t border-sidebar-border p-3', className)}
      data-slot='sidebar-footer'
      {...props}
    />
  );
}

function SidebarGroup({ className, ...props }: React.ComponentProps<'div'>) {
  return (
    <div className={cn('flex flex-col gap-2', className)} data-slot='sidebar-group' {...props} />
  );
}

function SidebarGroupLabel({ className, ...props }: React.ComponentProps<'div'>) {
  const { open } = useSidebar();
  if (!open) return null;

  return (
    <div
      className={cn(
        'px-2 pt-2 text-[0.68rem] font-semibold tracking-[0.16em] text-sidebar-foreground/55 uppercase',
        className
      )}
      data-slot='sidebar-group-label'
      {...props}
    />
  );
}

function SidebarMenu({ className, ...props }: React.ComponentProps<'ul'>) {
  return <ul className={cn('grid gap-1', className)} data-slot='sidebar-menu' {...props} />;
}

function SidebarMenuItem({ className, ...props }: React.ComponentProps<'li'>) {
  return <li className={cn('min-w-0', className)} data-slot='sidebar-menu-item' {...props} />;
}

interface SidebarMenuButtonProps extends React.ComponentProps<'button'> {
  active?: boolean;
  expanded?: boolean;
}

const SidebarMenuButton = React.forwardRef<HTMLButtonElement, SidebarMenuButtonProps>(
  ({ active = false, className, expanded, ...props }, ref) => {
    const { open } = useSidebar();
    const labelsVisible = expanded ?? open;

    return (
      <button
        className={cn(
          'flex h-10 w-full items-center gap-3 overflow-hidden rounded-md px-3 text-left text-sm outline-none transition-colors hover:bg-sidebar-accent focus-visible:ring-2 focus-visible:ring-ring',
          active &&
            'bg-sidebar-primary text-sidebar-primary-foreground hover:bg-sidebar-primary/90',
          !labelsVisible && 'justify-center px-0',
          className
        )}
        data-active={active || undefined}
        data-slot='sidebar-menu-button'
        ref={ref}
        type='button'
        {...props}
      />
    );
  }
);
SidebarMenuButton.displayName = 'SidebarMenuButton';

function SidebarInset({ className, ...props }: React.ComponentProps<'div'>) {
  return (
    <div
      className={cn('flex h-svh min-w-0 flex-1 flex-col bg-background', className)}
      data-slot='sidebar-inset'
      {...props}
    />
  );
}

function SidebarTrigger({ className, onClick, ...props }: React.ComponentProps<typeof Button>) {
  const { open, toggleSidebar } = useSidebar();

  return (
    <Button
      aria-expanded={open}
      aria-label={open ? '收起侧边栏' : '展开侧边栏'}
      className={className}
      onClick={(event) => {
        onClick?.(event);
        toggleSidebar();
      }}
      size='icon'
      title={`${open ? '收起' : '展开'}侧边栏 (Ctrl/⌘+B)`}
      variant='ghost'
      {...props}
    >
      {open ? <PanelLeftClose aria-hidden='true' /> : <PanelLeftOpen aria-hidden='true' />}
    </Button>
  );
}

export {
  Sidebar,
  SidebarContent,
  SidebarFooter,
  SidebarGroup,
  SidebarGroupLabel,
  SidebarHeader,
  SidebarInset,
  SidebarMenu,
  SidebarMenuButton,
  SidebarMenuItem,
  SidebarProvider,
  SidebarTrigger,
  useSidebar
};
