// @vitest-environment jsdom
import { act, createElement, type ReactNode } from 'react';
import { createRoot, type Root } from 'react-dom/client';
import { afterEach, beforeEach, expect, it, vi } from 'vitest';
import { ProfileView, type ProfileViewProps } from './profile-view';

vi.mock('@tarojs/components', () => ({
  View: ({ children, ...props }: { children?: ReactNode }) => createElement('div', props, children),
  Text: 'span',
  Checkbox: ({ children }: { children?: ReactNode }) => createElement('span', {}, children),
  CheckboxGroup: ({
    children,
    onChange
  }: {
    children?: ReactNode;
    onChange: (e: { detail: { value: string[] } }) => void;
  }) =>
    createElement(
      'label',
      {},
      createElement('input', {
        type: 'checkbox',
        onChange: (e: { target: HTMLInputElement }) =>
          onChange({ detail: { value: e.target.checked ? ['privacy'] : [] } })
      }),
      children
    ),
  Image: 'img',
  Button: ({
    children,
    hoverClass: _hover,
    openType: _openType,
    onGetPhoneNumber,
    ...props
  }: {
    children?: ReactNode;
    hoverClass?: string;
    openType?: string;
    onGetPhoneNumber?: (e: { detail: { code?: string } }) => void;
  }) => {
    return createElement('button', props, children);
  },
  Input: ({
    onInput,
    maxlength,
    focus: _focus,
    confirmType: _type,
    onConfirm: _confirm,
    ...props
  }: {
    onInput: (event: { detail: { value: string } }) => void;
    maxlength?: number;
    focus?: boolean;
    confirmType?: string;
    onConfirm?: () => void;
  }) =>
    createElement('input', {
      ...props,
      maxLength: maxlength,
      onChange: () => {},
      onInput: (event: { currentTarget: HTMLInputElement }) =>
        onInput({ detail: { value: event.currentTarget.value } })
    })
}));
let root: Root;
let host: HTMLDivElement;
let props: ProfileViewProps;
beforeEach(async () => {
  Object.assign(globalThis, { IS_REACT_ACT_ENVIRONMENT: true });
  host = document.createElement('div');
  document.body.appendChild(host);
  root = createRoot(host);
  props = {
    profile: {
      user_id: 'user',
      nickname: 'david',
      status: 'active',
      expires_at: '2099-01-01T00:00:00Z'
    },
    pendingAction: null,
    error: '',
    onOpenLogin: vi.fn(),
    onOpenInformation: vi.fn(),
    onSaveNickname: vi.fn().mockResolvedValue(true),
    onLogout: vi.fn().mockResolvedValue(false),
    onRetry: vi.fn().mockResolvedValue(true)
  };
  await render();
  await act(async () =>
    host.querySelector<HTMLButtonElement>('.profile-page__identity-button')!.click()
  );
});
afterEach(async () => {
  await act(async () => root.unmount());
  host.remove();
});
async function render() {
  await act(async () => root.render(createElement(ProfileView, props)));
}
async function click(label: string) {
  const button = Array.from(host.querySelectorAll('button')).find(
    (node) => node.textContent === label
  );
  expect(button).toBeDefined();
  await act(async () => button!.click());
}
async function input(value: string) {
  const element = host.querySelector('input')!;
  await act(async () => {
    Object.getOwnPropertyDescriptor(HTMLInputElement.prototype, 'value')!.set!.call(element, value);
    element.dispatchEvent(new Event('input', { bubbles: true }));
  });
}
it('separates personal details from the account session action', () => {
  expect(host.textContent).toContain('个人资料');
  expect(host.querySelector('input')).toBeNull();
  expect(host.querySelector('.profile-page__card')?.textContent).not.toContain('退出登录');
  expect(host.querySelector('.profile-page__session')?.textContent).toContain('退出登录');
});
it('cancels edits without writing personal data', async () => {
  await click('编辑资料');
  await input('其他名字');
  await click('取消');
  expect(props.onSaveNickname).not.toHaveBeenCalled();
  expect(host.querySelector('input')).toBeNull();
  await click('编辑资料');
  expect(host.querySelector('input')?.value).toBe('david');
});
it('keeps failed edits and closes with feedback after successful save', async () => {
  vi.mocked(props.onSaveNickname).mockResolvedValueOnce(false).mockResolvedValueOnce(true);
  await click('编辑资料');
  await input(' 新昵称 ');
  await click('保存修改');
  expect(host.querySelector('input')?.value).toBe(' 新昵称 ');
  await click('保存修改');
  expect(props.onSaveNickname).toHaveBeenLastCalledWith('新昵称');
  expect(host.querySelector('input')).toBeNull();
  expect(host.textContent).toContain('个人资料已更新');
});
it('opens the separate login page from the guest identity header', async () => {
  props.profile = null;
  await render();
  await click('登录/注册');
  expect(props.onOpenLogin).toHaveBeenCalledOnce();
  expect(host.textContent).not.toContain('我的服务');
  expect(host.textContent).not.toContain('余额');
  expect(host.textContent).not.toContain('手机号快捷登录');
});

it('keeps editing focused on personal details and disables invalid submissions', async () => {
  await click('编辑资料');
  expect(host.textContent).not.toContain('退出登录');
  await input('   ');
  const save = Array.from(host.querySelectorAll('button')).find(
    (node) => node.textContent === '保存修改'
  )!;
  expect(save.disabled).toBe(true);
  await act(async () => save.click());
  expect(props.onSaveNickname).not.toHaveBeenCalled();
  props.pendingAction = 'nickname';
  await render();
  expect(host.textContent).toContain('保存中…');
  expect(host.querySelector('input')?.disabled).toBe(true);
});

it('opens about from the landing page', async () => {
  await click('返回我的');
  const entry = host.querySelector<HTMLButtonElement>('.profile-page__about-row');
  await act(async () => entry!.click());
  expect(props.onOpenInformation).toHaveBeenCalledWith('about');
});
