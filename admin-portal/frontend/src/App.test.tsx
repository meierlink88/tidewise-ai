import { render, screen, within } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import App from './App';
import { loadEvents } from './api/dataIngestion';

vi.mock('./api/dataIngestion', async () => {
  const actual = await vi.importActual<typeof import('./api/dataIngestion')>('./api/dataIngestion');
  return {
    ...actual,
    loadEvents: vi.fn()
  };
});

describe('App admin login', () => {
  const storage = new Map<string, string>();

  beforeEach(() => {
    storage.clear();
    document.documentElement.classList.remove('dark');
    const localStorageMock = {
      getItem: vi.fn((key: string) => storage.get(key) ?? null),
      setItem: vi.fn((key: string, value: string) => storage.set(key, value)),
      removeItem: vi.fn((key: string) => storage.delete(key)),
      clear: vi.fn(() => storage.clear()),
      key: vi.fn((index: number) => Array.from(storage.keys())[index] ?? null),
      get length() {
        return storage.size;
      }
    };
    vi.stubGlobal('localStorage', localStorageMock);
    vi.mocked(loadEvents).mockResolvedValue({ items: [], total: 0, page: 1, page_size: 50 });
  });

  it('shows a login page with the local admin token hint before entering the admin shell', () => {
    render(<App />);

    expect(screen.getByRole('heading', { name: '观潮家管理后台' })).toBeInTheDocument();
    expect(screen.getByText('测试 token：local-admin-token')).toBeInTheDocument();
    expect(screen.queryByText('数据采集中心')).not.toBeInTheDocument();
  });

  it('associates an empty-token error with the login field', async () => {
    const user = userEvent.setup();

    render(<App />);
    const tokenInput = screen.getByLabelText('Admin Token');
    await user.click(screen.getByRole('button', { name: '登录' }));

    expect(screen.getByRole('alert')).toHaveTextContent('请输入 Admin Token');
    expect(tokenInput).toHaveAttribute('aria-invalid', 'true');
    expect(tokenInput).toHaveAccessibleDescription('请输入 Admin Token');
    expect(tokenInput).toHaveFocus();
  });

  it('logs in with an admin token and logs out back to the login page', async () => {
    const user = userEvent.setup();

    render(<App />);

    await user.type(screen.getByLabelText('Admin Token'), 'local-admin-token');
    await user.click(screen.getByRole('button', { name: '登录' }));

    expect(
      await within(screen.getByRole('banner')).findByRole('heading', { name: '数据采集中心' })
    ).toBeInTheDocument();
    expect(
      within(screen.getByRole('main')).getByRole('heading', { name: '事件中心' })
    ).toBeInTheDocument();
    expect(screen.getByRole('button', { name: /^采集中心/ })).toHaveAttribute(
      'aria-current',
      'page'
    );
    expect(screen.queryByRole('button', { name: /^数据采集中心/ })).not.toBeInTheDocument();
    expect(
      within(screen.getByRole('main')).getByText(
        '查询标准化事件。证据通过 Event Evidence Link 关联。'
      )
    ).toBeInTheDocument();
    expect(within(screen.getByRole('main')).queryByText('Data Ingestion')).not.toBeInTheDocument();
    expect(
      within(screen.getByRole('main')).queryByText(
        '查看采集原始数据、事件结果、搜索通道和调度器运行记录。'
      )
    ).not.toBeInTheDocument();
    expect(loadEvents).toHaveBeenCalledWith('local-admin-token', { page: 1, title: '' });
    expect(storage.get('tidewise_admin_token')).toBe('local-admin-token');

    await user.click(screen.getByRole('button', { name: '退出登录' }));

    expect(screen.getByRole('heading', { name: '观潮家管理后台' })).toBeInTheDocument();
    expect(storage.has('tidewise_admin_token')).toBe(false);
  });

  it('opens the navigation on narrow screens and returns focus after choosing a page', async () => {
    storage.set('tidewise_admin_token', 'local-admin-token');
    const user = userEvent.setup();

    render(<App />);

    const menuTrigger = screen.getByRole('button', { name: '打开导航菜单' });
    menuTrigger.focus();
    await user.keyboard('{Enter}');

    const navigation = screen.getByRole('dialog', { name: '管理后台导航' });
    const collectionNavigation = within(navigation).getByRole('button', { name: /^采集中心/ });
    collectionNavigation.focus();
    await user.keyboard('{Enter}');

    expect(await screen.findByRole('heading', { name: '事件中心', level: 2 })).toBeInTheDocument();
    expect(screen.queryByRole('dialog', { name: '管理后台导航' })).not.toBeInTheDocument();
    expect(menuTrigger).toHaveFocus();
  });

  it('collapses the desktop navigation from its trigger and keyboard shortcut', async () => {
    storage.set('tidewise_admin_token', 'local-admin-token');
    const user = userEvent.setup();

    render(<App />);

    const sidebarTrigger = screen.getByRole('button', { name: '收起侧边栏' });
    expect(sidebarTrigger).toHaveAttribute('aria-expanded', 'true');

    await user.click(sidebarTrigger);
    expect(screen.getByRole('button', { name: '展开侧边栏' })).toHaveAttribute(
      'aria-expanded',
      'false'
    );

    await user.keyboard('{Control>}b{/Control}');
    expect(screen.getByRole('button', { name: '收起侧边栏' })).toHaveAttribute(
      'aria-expanded',
      'true'
    );
  });

  it('switches and persists the Admin color theme', async () => {
    storage.set('tidewise_admin_token', 'local-admin-token');
    const user = userEvent.setup();

    render(<App />);
    await user.click(screen.getByRole('button', { name: '切换到深色主题' }));

    expect(document.documentElement).toHaveClass('dark');
    expect(storage.get('tidewise_admin_theme')).toBe('dark');
    expect(screen.getByRole('button', { name: '切换到浅色主题' })).toBeInTheDocument();
  });

  it('uses the light shadcn-admin theme by default even when the OS prefers dark', async () => {
    storage.set('tidewise_admin_token', 'local-admin-token');
    vi.stubGlobal('matchMedia', (query: string) => ({
      matches: query === '(prefers-color-scheme: dark)',
      media: query,
      onchange: null,
      addEventListener: vi.fn(),
      removeEventListener: vi.fn(),
      addListener: vi.fn(),
      removeListener: vi.fn(),
      dispatchEvent: vi.fn()
    }));

    render(<App />);

    await screen.findByText('暂无事件');
    expect(document.documentElement).not.toHaveClass('dark');
    expect(storage.get('tidewise_admin_theme')).toBe('light');
    expect(screen.getByRole('button', { name: '切换到深色主题' })).toBeInTheDocument();
  });
});
