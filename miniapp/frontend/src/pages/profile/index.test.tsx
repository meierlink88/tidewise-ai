// @vitest-environment jsdom
import { act, createElement } from 'react';
import { createRoot } from 'react-dom/client';
import { afterEach, expect, it, vi } from 'vitest';
import ProfilePage from './index';

const state = vi.hoisted(() => ({
  profile: null as null | { user_id: string; nickname: string },
  pendingAction: null as string | null,
  error: '',
  busy: false,
  login: vi.fn(),
  logout: vi.fn(),
  saveNickname: vi.fn(),
  refresh: vi.fn()
}));
const nav = vi.hoisted(() => ({
  tracking: vi.fn(),
  info: vi.fn(),
  editor: vi.fn(),
  confirm: vi.fn()
}));
vi.mock('../../features/identity/use-identity', () => ({ useIdentity: () => state }));
vi.mock('../../platform/system-ui', () => ({ getHomeChromeMetrics: () => ({}) }));
vi.mock('../../platform/navigation-bar', () => ({
  NavigationBar: ({ title }: { title: string }) => createElement('h1', {}, title)
}));
vi.mock('../../platform/identity', () => ({
  supportsWechatLogin: true,
  leaveProfile: vi.fn(),
  openLogin: vi.fn(),
  openProfileEditor: nav.editor,
  openProfileInformation: nav.info,
  confirmLogout: nav.confirm
}));
vi.mock('../../platform/tracking', () => ({ openTracking: nav.tracking }));
vi.mock('@tarojs/taro', () => ({ default: {} }));
vi.mock('@tarojs/components', () => ({ View: 'div', Button: 'button', Text: 'span' }));
vi.mock('../login/login-view', () => ({
  LoginView: ({
    onLogin,
    onOpenPrivacy
  }: {
    onLogin: (code?: string) => void;
    onOpenPrivacy: () => void;
  }) =>
    createElement(
      'div',
      {},
      createElement('button', { onClick: () => onLogin('phone-code') }, '真实登录'),
      createElement('button', { onClick: onOpenPrivacy }, '隐私政策')
    )
}));
vi.mock('./profile-view', () => ({
  ProfileView: ({
    onOpenEditor,
    onLogout,
    onOpenTracking
  }: {
    onOpenEditor: () => void;
    onLogout: () => void;
    onOpenTracking: () => void;
  }) =>
    createElement(
      'div',
      {},
      createElement('button', { onClick: onOpenEditor }, '保存资料'),
      createElement('button', { onClick: onLogout }, '退出登录'),
      createElement('button', { onClick: onOpenTracking }, '我的跟踪')
    )
}));
let dispose: () => Promise<void>;
afterEach(async () => {
  await dispose?.();
  vi.clearAllMocks();
  state.profile = null;
  state.pendingAction = null;
});
async function mount() {
  Object.assign(globalThis, { IS_REACT_ACT_ENVIRONMENT: true });
  const host = document.createElement('div');
  document.body.append(host);
  const root = createRoot(host);
  await act(async () => root.render(createElement(ProfilePage)));
  dispose = async () => {
    await act(async () => root.unmount());
    host.remove();
  };
  return host;
}
async function click(host: HTMLElement, text: string) {
  const button = [...host.querySelectorAll('button')].find((b) => b.textContent === text)!;
  expect(button).toBeDefined();
  await act(async () => button.click());
}
it('opens real login directly for guests, forwarding the phone code to the identity feature', async () => {
  const host = await mount();
  expect(host.textContent).toContain('欢迎登录观潮家');
  expect(host.textContent).not.toContain('登录/注册');
  await click(host, '真实登录');
  expect(state.login).toHaveBeenCalledWith('phone-code');
  expect(state.saveNickname).not.toHaveBeenCalled();
});
it('keeps profile save, tracking and confirmed logout wired to real owners', async () => {
  state.profile = { user_id: 'real-user', nickname: '用户' };
  const host = await mount();
  await click(host, '保存资料');
  expect(nav.editor).toHaveBeenCalledOnce();
  await click(host, '我的跟踪');
  expect(nav.tracking).toHaveBeenCalledOnce();
  nav.confirm.mockResolvedValueOnce(false);
  await click(host, '退出登录');
  expect(state.logout).not.toHaveBeenCalled();
  nav.confirm.mockResolvedValueOnce(true);
  await click(host, '退出登录');
  expect(state.logout).toHaveBeenCalledOnce();
  expect(host.textContent).not.toContain('真实登录');
});
it('shows restoration state instead of briefly offering guest login', async () => {
  state.pendingAction = 'refresh';
  const host = await mount();
  expect(host.textContent).toContain('正在检查登录状态');
  expect(host.textContent).not.toContain('真实登录');
});
