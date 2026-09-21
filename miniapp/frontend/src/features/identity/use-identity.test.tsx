// @vitest-environment jsdom
import { act, createElement } from 'react';
import { createRoot, type Root } from 'react-dom/client';
import { afterEach, beforeEach, expect, it, vi } from 'vitest';
import { useIdentity } from './use-identity';
import { IdentityError } from './api';

const mock = vi.hoisted(() => ({
  show: () => {},
  me: vi.fn(),
  login: vi.fn(),
  logout: vi.fn(),
  updateNickname: vi.fn(),
  readSession: vi.fn(),
  saveSession: vi.fn(),
  clearSession: vi.fn(),
  requestWechatCode: vi.fn()
}));
vi.mock('@tarojs/taro', () => ({
  useDidShow: (fn: () => void) => {
    mock.show = fn;
  }
}));
vi.mock('../../platform/identity', () => mock);
vi.mock('./api', async (importOriginal) => ({
  ...(await importOriginal<typeof import('./api')>()),
  me: mock.me,
  login: mock.login,
  logout: mock.logout,
  updateNickname: mock.updateNickname
}));
let state: ReturnType<typeof useIdentity>;
let root: Root;
const profile = {
  user_id: 'f466d548-4ac3-4d3f-b994-cbeeb6eb3ef2',
  nickname: '昵称',
  status: 'active',
  expires_at: '2099-01-01T00:00:00Z'
};
const session = { session_token: 'A'.repeat(43), expires_at: profile.expires_at };
function Harness() {
  state = useIdentity();
  return null;
}
beforeEach(async () => {
  vi.clearAllMocks();
  Object.assign(globalThis, { IS_REACT_ACT_ENVIRONMENT: true });
  mock.readSession.mockReturnValue(session);
  mock.me.mockResolvedValue(profile);
  mock.requestWechatCode.mockResolvedValue('code');
  mock.login.mockResolvedValue({ ...profile, ...session });
  mock.logout.mockResolvedValue(undefined);
  root = createRoot(document.createElement('div'));
  await act(async () => root.render(createElement(Harness)));
});
afterEach(async () => {
  await act(async () => root.unmount());
});
it('restores verified identity and retains it on network failures', async () => {
  await act(async () => mock.show());
  expect(state.profile?.nickname).toBe('昵称');
  mock.me.mockRejectedValue(new IdentityError('网络失败'));
  await act(async () => state.refresh());
  expect(state.profile?.nickname).toBe('昵称');
  expect(mock.clearSession).not.toHaveBeenCalled();
});
it('clears rejected identity and storage', async () => {
  await act(async () => mock.show());
  mock.me.mockRejectedValue(new IdentityError('过期', true));
  await act(async () => state.refresh());
  expect(state.profile).toBeNull();
  expect(mock.clearSession).toHaveBeenCalledOnce();
});
it('does not claim logout until revocation succeeds', async () => {
  await act(async () => mock.show());
  mock.logout.mockRejectedValueOnce(new IdentityError('网络失败'));
  await act(async () => state.logout());
  expect(state.profile).not.toBeNull();
  expect(mock.clearSession).not.toHaveBeenCalled();
  await act(async () => state.logout());
  expect(state.profile).toBeNull();
  expect(mock.clearSession).toHaveBeenCalledOnce();
});
it('prevents duplicate login calls', async () => {
  await act(async () => {
    await Promise.all([state.login(), state.login()]);
  });
  expect(mock.requestWechatCode).toHaveBeenCalledOnce();
  expect(mock.login).toHaveBeenCalledOnce();
  expect(mock.saveSession).toHaveBeenCalledWith(session);
});
