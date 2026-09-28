// @vitest-environment jsdom
import { act, createElement } from 'react';
import { createRoot, type Root } from 'react-dom/client';
import { afterEach, beforeEach, expect, it, vi } from 'vitest';
import { AccountNavButton } from './account-nav-button';

const mock = vi.hoisted(() => ({
  show: () => {},
  me: vi.fn(),
  avatar: vi.fn(),
  readSession: vi.fn()
}));
vi.mock('@tarojs/taro', () => ({
  useDidShow: (fn: () => void) => {
    mock.show = fn;
  }
}));
vi.mock('@tarojs/components', () => ({
  Button: ({ children }: { children: React.ReactNode }) => createElement('button', null, children),
  View: ({ children }: { children: React.ReactNode }) => createElement('div', null, children),
  Text: ({ children }: { children: React.ReactNode }) => createElement('span', null, children),
  Image: ({ src }: { src: string }) => createElement('img', { src })
}));
vi.mock('../../platform/identity', () => ({
  readSession: mock.readSession,
  openProfile: vi.fn(),
  clearSession: vi.fn()
}));
vi.mock('../../platform/avatar', () => ({
  displayAvatar: vi.fn().mockResolvedValue('wxfile://current-avatar.jpg')
}));
vi.mock('./api', async (original) => ({
  ...(await original<typeof import('./api')>()),
  me: mock.me,
  avatar: mock.avatar
}));
let root: Root;
let host: HTMLDivElement;
const profile = {
  user_id: 'f466d548-4ac3-4d3f-b994-cbeeb6eb3ef2',
  nickname: '当前用户',
  status: 'active',
  expires_at: '2099-01-01T00:00:00Z'
};
beforeEach(() => {
  vi.clearAllMocks();
  Object.assign(globalThis, { IS_REACT_ACT_ENVIRONMENT: true });
  mock.me.mockResolvedValue(profile);
  mock.avatar.mockResolvedValue('avatar-data');
  mock.readSession.mockReturnValue({
    session_token: 'A'.repeat(43),
    expires_at: profile.expires_at
  });
  host = document.createElement('div');
  root = createRoot(host);
});
afterEach(async () => {
  await act(async () => root.unmount());
});
it('loads nickname and avatar when mounted after the page show event by the login gate', async () => {
  // onShow has already fired before the protected page mounts its header.
  await act(async () => root.render(createElement(AccountNavButton)));
  expect(host.textContent).toContain('当前用户');
  expect(host.querySelector('img')?.getAttribute('src')).toBe('wxfile://current-avatar.jpg');
  expect(mock.me).toHaveBeenCalledOnce();
});
it('refreshes the nickname on the next page show and clears it after logout', async () => {
  await act(async () => root.render(createElement(AccountNavButton)));
  mock.me.mockResolvedValue({ ...profile, nickname: '更新昵称' });
  await act(async () => mock.show());
  expect(host.textContent).toContain('更新昵称');
  mock.readSession.mockReturnValue(null);
  await act(async () => mock.show());
  expect(host.textContent).not.toContain('更新昵称');
  expect(host.querySelector('img')?.getAttribute('src')).not.toBe('wxfile://current-avatar.jpg');
});

it('does not duplicate an in-flight mount refresh when the page show event also arrives', async () => {
  let resolve!: (value: typeof profile) => void;
  mock.me.mockReturnValue(
    new Promise((done) => {
      resolve = done;
    })
  );
  await act(async () => root.render(createElement(AccountNavButton)));
  await act(async () => mock.show());
  expect(mock.me).toHaveBeenCalledOnce();
  await act(async () => resolve(profile));
  expect(host.textContent).toContain('当前用户');
  expect(host.querySelector('img')?.getAttribute('src')).toBe('wxfile://current-avatar.jpg');
});
