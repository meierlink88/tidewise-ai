// @vitest-environment jsdom
import { act, createElement } from 'react';
import { createRoot, type Root } from 'react-dom/client';
import { afterEach, beforeEach, expect, it, vi } from 'vitest';
import { useTracking } from './use-tracking';
import { TrackingError, type Page, type Company, type TrackingPort } from './contract';

const mock = vi.hoisted(() => ({
  show: () => {},
  hide: () => {},
  readSession: vi.fn(),
  clearSession: vi.fn()
}));
vi.mock('@tarojs/taro', () => ({
  useDidShow: (f: () => void) => {
    mock.show = f;
  },
  useDidHide: (f: () => void) => {
    mock.hide = f;
  }
}));
vi.mock('../../platform/identity', () => mock);
vi.mock('./api', () => ({ trackingAPI: {} }));
let root: Root;
let state: ReturnType<typeof useTracking>;
const company: Company = {
  id: 'STKf4a8eb61-c352-5980-91b1-9da6eba8f8af',
  title: '平安银行股份有限公司',
  stock_name: '平安银行',
  symbol: '000001.SZ',
  industry_label: '银行',
  industry_path: '金融 / 银行',
  concepts: ['跨境支付'],
  is_followed: false
};
const page = (items: Company[] = [], total = items.length): Page => ({
  items,
  total,
  next_cursor: '',
  has_more: false
});
let port: TrackingPort;
let mode: 'watchlist' | 'directory' = 'watchlist';
let enabled: boolean | undefined;
function deferred<T>() {
  let resolve!: (v: T) => void;
  let reject!: (e: Error) => void;
  const promise = new Promise<T>((a, b) => {
    resolve = a;
    reject = b;
  });
  return { promise, resolve, reject };
}
function Harness() {
  state = useTracking(port, mode, enabled);
  return null;
}
beforeEach(async () => {
  mode = 'watchlist';
  enabled = undefined;
  vi.useFakeTimers();
  vi.clearAllMocks();
  Object.assign(globalThis, { IS_REACT_ACT_ENVIRONMENT: true });
  mock.readSession.mockReturnValue(null);
  port = {
    search: vi.fn().mockResolvedValue(page()),
    list: vi.fn().mockResolvedValue(page()),
    change: vi.fn().mockResolvedValue(undefined)
  };
  root = createRoot(document.createElement('div'));
  await act(async () => root.render(createElement(Harness)));
  await act(async () => mock.show());
});
afterEach(async () => {
  await act(async () => root.unmount());
  vi.useRealTimers();
});
it('debounces search and rejects old responses immediately when input changes', async () => {
  const old = deferred<Page>();
  vi.mocked(port.search).mockReturnValueOnce(old.promise);
  await act(async () => state.setQuery('PAYH'));
  await act(async () => vi.advanceTimersByTime(250));
  await act(async () => state.setQuery('GZMT'));
  await act(async () => old.resolve(page([company])));
  expect(state.results).toEqual([]);
  await act(async () => vi.advanceTimersByTime(250));
  expect(port.search).toHaveBeenLastCalledWith('GZMT', '', 0);
  expect(state.searchStatus).toBe('ready');
});
it('never displays another account late list and clears rows on logout', async () => {
  const old = deferred<Page>();
  vi.mocked(port.list).mockReturnValueOnce(old.promise);
  mock.readSession.mockReturnValue({ session_token: 'a' });
  await act(async () => mock.show());
  mock.readSession.mockReturnValue({ session_token: 'b' });
  await act(async () => mock.show());
  await act(async () => old.resolve(page([company])));
  expect(state.items).toEqual([]);
  expect(port.list).toHaveBeenLastCalledWith('b', '');
  mock.readSession.mockReturnValue(null);
  await act(async () => mock.show());
  expect(state.guest).toBe(true);
  expect(state.total).toBe(0);
});
it('locks duplicate mutations and updates count only from persisted list', async () => {
  mock.readSession.mockReturnValue({ session_token: 'a' });
  await act(async () => mock.show());
  const saving = deferred<void>();
  vi.mocked(port.change).mockReturnValueOnce(saving.promise);
  let first: Promise<void>;
  await act(async () => {
    first = state.change(company, true);
    void state.change(company, true);
  });
  expect(port.change).toHaveBeenCalledTimes(1);
  expect(state.pending).toBe(company.id);
  expect(state.total).toBe(0);
  vi.mocked(port.list).mockResolvedValue(page([{ ...company, is_followed: true }]));
  await act(async () => {
    saving.resolve();
    await first;
  });
  expect(state.total).toBe(1);
  expect(state.pending).toBe('');
});
it('retains prior rows on failed write and clears identity only for auth failures', async () => {
  mock.readSession.mockReturnValue({ session_token: 'a' });
  vi.mocked(port.list).mockResolvedValue(page([{ ...company, is_followed: true }]));
  await act(async () => mock.show());
  vi.mocked(port.change).mockRejectedValueOnce(new TrackingError('网络失败'));
  await act(async () => state.change(company, false));
  expect(state.items).toHaveLength(1);
  expect(state.status).toBe('error');
  expect(mock.clearSession).not.toHaveBeenCalled();
  vi.mocked(port.change).mockRejectedValueOnce(new TrackingError('会话失效', true));
  await act(async () => state.change(company, false));
  expect(state.items).toEqual([]);
  expect(state.guest).toBe(true);
  expect(mock.clearSession).toHaveBeenCalledOnce();
});
it('deduplicates pagination and ignores a response after leaving the page', async () => {
  mock.readSession.mockReturnValue({ session_token: 'a' });
  vi.mocked(port.list).mockResolvedValueOnce({ ...page([company]), next_cursor: 'next' });
  await act(async () => mock.show());
  vi.mocked(port.list).mockResolvedValueOnce(page([company]));
  await act(async () => state.loadMore());
  expect(state.items).toHaveLength(1);
  expect(state.hasMore).toBe(false);
  const late = deferred<Page>();
  vi.mocked(port.search).mockReturnValue(late.promise);
  await act(async () => state.setQuery('PAYH'));
  await act(async () => vi.advanceTimersByTime(250));
  await act(async () => mock.hide());
  await act(async () => late.resolve(page([company])));
  expect(state.results).toEqual([]);
});

it('browses the public directory, paginates, searches and restores it without reading a private list', async () => {
  mode = 'directory';
  await act(async () => root.render(createElement(Harness)));
  vi.mocked(port.search).mockResolvedValueOnce({ ...page([company]), has_more: true });
  await act(async () => mock.show());
  expect(port.search).toHaveBeenLastCalledWith('', '', 0);
  expect(state.results).toHaveLength(1);
  await act(async () => state.loadMore());
  expect(port.search).toHaveBeenLastCalledWith('', '', 1);
  await act(async () => state.setQuery('平安银行股份有限公司'));
  await act(async () => vi.advanceTimersByTime(250));
  expect(port.search).toHaveBeenLastCalledWith('平安银行股份有限公司', '', 0);
  await act(async () => state.setQuery(''));
  await act(async () => vi.advanceTimersByTime(250));
  expect(port.search).toHaveBeenLastCalledWith('', '', 0);
  mock.readSession.mockReturnValue({ session_token: 'owner' });
  vi.mocked(port.search).mockResolvedValue(page([{ ...company, is_followed: true }]));
  await act(async () => mock.show());
  expect(state.results[0].is_followed).toBe(true);
  await act(async () => state.change(company, true));
  expect(port.change).toHaveBeenLastCalledWith('owner', company.id, true);
  expect(port.list).not.toHaveBeenCalled();
});

it('keeps loaded directory pages after a persisted follow and rejects older membership reads', async () => {
  mode = 'directory';
  await act(async () => root.render(createElement(Harness)));
  mock.readSession.mockReturnValue({ session_token: 'owner' });
  const second = { ...company, id: 'STK00000000-0000-4000-8000-000000000002' };
  vi.mocked(port.search).mockResolvedValueOnce({ ...page([company]), has_more: true });
  await act(async () => mock.show());
  vi.mocked(port.search).mockResolvedValueOnce({ ...page([second]), has_more: true });
  await act(async () => state.loadMore());
  const old = deferred<Page>();
  vi.mocked(port.search).mockReturnValueOnce(old.promise);
  await act(async () => {
    void state.loadMore();
  });
  await act(async () => state.change(second, true));
  await act(async () => old.resolve(page([{ ...second, is_followed: false }])));
  expect(state.results).toHaveLength(2);
  expect(state.results[1].is_followed).toBe(true);
  expect(state.hasMore).toBe(true);
  expect(state.searchStatus).toBe('ready');
});

it('defers directory requests until page access is verified and discards data when disabled', async () => {
  mode = 'directory';
  enabled = false;
  await act(async () => root.render(createElement(Harness)));
  await act(async () => mock.show());
  expect(port.search).not.toHaveBeenCalled();
  mock.readSession.mockReturnValue({ session_token: 'signed-in' });
  vi.mocked(port.search).mockResolvedValue(page([company]));
  enabled = true;
  await act(async () => root.render(createElement(Harness)));
  expect(port.search).toHaveBeenCalledWith('', 'signed-in', 0);
  expect(state.results).toHaveLength(1);
  enabled = false;
  await act(async () => root.render(createElement(Harness)));
  expect(state.results).toEqual([]);
});
