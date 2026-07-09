import { render, screen, within } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import App from './App';
import { loadEvents, loadRawDocuments, loadSourceCatalogs } from './api/dataIngestion';
import { loadSchedulerConfig, loadSchedulerRuns } from './api/scheduler';

vi.mock('./api/scheduler', async () => {
  const actual = await vi.importActual<typeof import('./api/scheduler')>('./api/scheduler');
  return {
    ...actual,
    loadSchedulerConfig: vi.fn(),
    loadSchedulerRuns: vi.fn(),
    saveSchedulerConfig: vi.fn()
  };
});

vi.mock('./api/dataIngestion', async () => {
  const actual = await vi.importActual<typeof import('./api/dataIngestion')>('./api/dataIngestion');
  return {
    ...actual,
    loadEvents: vi.fn(),
    loadRawDocuments: vi.fn(),
    loadSourceCatalogs: vi.fn()
  };
});

describe('App admin login', () => {
  const storage = new Map<string, string>();

  beforeEach(() => {
    storage.clear();
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
    vi.mocked(loadSchedulerConfig).mockResolvedValue({
      enabled: false,
      mode: 'interval',
      interval_minutes: 60,
      fixed_times: [],
      concurrency: 1,
      batch_size: 10,
      timeout_seconds: 180,
      source_filter: {},
      timezone: 'Asia/Shanghai'
    });
    vi.mocked(loadSchedulerRuns).mockResolvedValue([]);
    vi.mocked(loadRawDocuments).mockResolvedValue({ items: [], total: 0, page: 1, page_size: 50 });
    vi.mocked(loadEvents).mockResolvedValue({ items: [], total: 0, page: 1, page_size: 50 });
    vi.mocked(loadSourceCatalogs).mockResolvedValue({ items: [] });
  });

  it('shows a login page with the local admin token hint before entering the admin shell', () => {
    render(<App />);

    expect(screen.getByRole('heading', { name: '观潮家管理后台' })).toBeInTheDocument();
    expect(screen.getByText('测试 token：local-admin-token')).toBeInTheDocument();
    expect(screen.queryByText('数据采集中心')).not.toBeInTheDocument();
  });

  it('logs in with an admin token and logs out back to the login page', async () => {
    const user = userEvent.setup();

    render(<App />);

    await user.type(screen.getByLabelText('Admin Token'), 'local-admin-token');
    await user.click(screen.getByRole('button', { name: '登录' }));

    expect(await within(screen.getByRole('banner')).findByRole('heading', { name: '数据采集中心' })).toBeInTheDocument();
    expect(within(screen.getByRole('main')).queryByRole('heading', { name: '数据采集中心' })).not.toBeInTheDocument();
    expect(within(screen.getByRole('main')).queryByText('Data Ingestion')).not.toBeInTheDocument();
    expect(within(screen.getByRole('main')).queryByText('查看采集原始数据、事件结果、搜索通道和调度器运行记录。')).not.toBeInTheDocument();
    expect(loadRawDocuments).toHaveBeenCalledWith('local-admin-token', { page: 1, title: '' });
    expect(storage.get('tidewise_admin_token')).toBe('local-admin-token');

    await user.click(screen.getByRole('button', { name: '退出登录' }));

    expect(screen.getByRole('heading', { name: '观潮家管理后台' })).toBeInTheDocument();
    expect(storage.has('tidewise_admin_token')).toBe(false);
  });
});
