// @vitest-environment jsdom
import { act, createElement } from 'react';
import { createRoot, type Root } from 'react-dom/client';
import { afterEach, beforeEach, expect, it, vi } from 'vitest';
import { useResearchAccess } from './use-research-access';
import { IdentityError } from './api';

const mock = vi.hoisted(() => ({
  show: () => {},
  hide: () => {},
  readSession: vi.fn(),
  clearSession: vi.fn(),
  requireResearchLogin: vi.fn(),
  me: vi.fn()
}));
vi.mock('@tarojs/taro', () => ({
  useDidShow: (fn: () => void) => {
    mock.show = fn;
  },
  useDidHide: (fn: () => void) => {
    mock.hide = fn;
  }
}));
vi.mock('../../platform/identity', () => mock);
vi.mock('./api', async (original) => ({
  ...(await original<typeof import('./api')>()),
  me: mock.me
}));
let root: Root;
let state: ReturnType<typeof useResearchAccess>;
const session = { session_token: 'A'.repeat(43), expires_at: '2099-01-01T00:00:00Z' };
function Harness() {
  state = useResearchAccess('company');
  return null;
}
beforeEach(async () => {
  vi.clearAllMocks();
  vi.useFakeTimers();
  Object.assign(globalThis, { IS_REACT_ACT_ENVIRONMENT: true });
  mock.readSession.mockReturnValue(session);
  mock.me.mockResolvedValue({});
  mock.requireResearchLogin.mockResolvedValue(undefined);
  root = createRoot(document.createElement('div'));
  await act(async () => root.render(createElement(Harness)));
});
afterEach(async () => {
  await act(async () => root.unmount());
  vi.useRealTimers();
});
it('does not expose a guest page and opens login with its destination', async () => {
  mock.readSession.mockReturnValue(null);
  await act(async () => mock.show());
  expect(state.allowed).toBe(false);
  expect(mock.me).not.toHaveBeenCalled();
  expect(mock.requireResearchLogin).toHaveBeenCalledWith('company', undefined, undefined);
});
it('allows verified users, hides on leave, and rechecks after logout', async () => {
  expect(state.allowed).toBe(false);
  await act(async () => mock.show());
  expect(state.allowed).toBe(true);
  await act(async () => mock.hide());
  expect(state.allowed).toBe(false);
  mock.readSession.mockReturnValue(null);
  await act(async () => mock.show());
  expect(state.allowed).toBe(false);
  expect(mock.requireResearchLogin).toHaveBeenCalledOnce();
});
it('retains credentials on network failure, blocks content and allows retry', async () => {
  mock.me.mockRejectedValueOnce(new IdentityError('offline'));
  await act(async () => mock.show());
  expect(state.allowed).toBe(false);
  expect(state.error).toBe('offline');
  expect(mock.clearSession).not.toHaveBeenCalled();
  expect(mock.requireResearchLogin).not.toHaveBeenCalled();
  await act(async () => state.retry());
  expect(state.allowed).toBe(true);
});
it('rejects revoked sessions and handles failed login navigation with retry', async () => {
  mock.me.mockRejectedValue(new IdentityError('expired', true));
  mock.requireResearchLogin.mockRejectedValueOnce(new Error('navigation'));
  await act(async () => mock.show());
  expect(state.allowed).toBe(false);
  expect(mock.clearSession).toHaveBeenCalledOnce();
  expect(state.error).not.toBe('');
});
it('ignores a late successful verification after leaving the page', async () => {
  let resolve!: (v: object) => void;
  mock.me.mockReturnValue(
    new Promise((done) => {
      resolve = done;
    })
  );
  await act(async () => mock.show());
  await act(async () => mock.hide());
  await act(async () => resolve({}));
  expect(state.allowed).toBe(false);
  expect(mock.requireResearchLogin).not.toHaveBeenCalled();
});
it('rechecks an open page when the stored session expires', async () => {
  mock.readSession.mockReturnValue({
    ...session,
    expires_at: new Date(Date.now() + 1000).toISOString()
  });
  await act(async () => mock.show());
  expect(state.allowed).toBe(true);
  mock.readSession.mockReturnValue(null);
  await act(async () => vi.advanceTimersByTimeAsync(1000));
  expect(state.allowed).toBe(false);
  expect(mock.requireResearchLogin).toHaveBeenCalledOnce();
});

it('carries the selected company through guest and expired-session login', async () => {
  function ReportHarness() {
    state = useResearchAccess('companyReport', '000001.SZ', {
      stockName: '平安银行',
      companyName: '平安银行股份有限公司'
    });
    return null;
  }
  await act(async () => root.render(createElement(ReportHarness)));
  mock.readSession.mockReturnValue(null);
  await act(async () => mock.show());
  expect(mock.requireResearchLogin).toHaveBeenLastCalledWith('companyReport', '000001.SZ', {
    stockName: '平安银行',
    companyName: '平安银行股份有限公司'
  });
  mock.readSession.mockReturnValue(session);
  mock.me.mockRejectedValueOnce(new IdentityError('expired', true));
  await act(async () => mock.show());
  expect(state.allowed).toBe(false);
  expect(mock.requireResearchLogin).toHaveBeenLastCalledWith('companyReport', '000001.SZ', {
    stockName: '平安银行',
    companyName: '平安银行股份有限公司'
  });
});
