// @vitest-environment jsdom
import { act, createElement, type ReactNode } from 'react';
import { createRoot, type Root } from 'react-dom/client';
import { afterEach, beforeEach, expect, it, vi } from 'vitest';
import { ProfileView, type ProfileViewProps } from './profile-view';
import { ProfileEditorView, type ProfileEditorViewProps } from './profile-editor-view';

const capabilities = vi.hoisted(() => ({
  supported: false,
  choose: undefined as undefined | ((e: { detail: { avatarUrl: string } }) => void)
}));
vi.mock('../../platform/identity', () => ({
  get profileSubmitOnClick() {
    return !capabilities.supported;
  },
  get supportsWechatLogin() {
    return capabilities.supported;
  }
}));

vi.mock('@tarojs/components', () => ({
  View: ({ children, ...props }: { children?: ReactNode }) => createElement('div', props, children),
  Form: ({
    children,
    onSubmit
  }: {
    children?: ReactNode;
    onSubmit: (e: { detail: { value: Record<string, string> } }) => void;
  }) =>
    createElement(
      'form',
      {
        onSubmit: (e: React.FormEvent<HTMLFormElement>) => {
          e.preventDefault();
          const data = new FormData(e.currentTarget);
          onSubmit({ detail: { value: Object.fromEntries(data) as Record<string, string> } });
        }
      },
      children
    ),
  Text: 'span',
  ScrollView: ({
    children,
    scrollY: _scrollY,
    ...props
  }: {
    children?: ReactNode;
    scrollY?: boolean;
  }) => createElement('div', props, children),
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
    formType,
    onChooseAvatar,
    ...props
  }: {
    formType?: string;
    onChooseAvatar?: (e: { detail: { avatarUrl: string } }) => void;
    children?: ReactNode;
    hoverClass?: string;
    openType?: string;
    onGetPhoneNumber?: (e: { detail: { code?: string } }) => void;
  }) => {
    if (onChooseAvatar) capabilities.choose = onChooseAvatar;
    return createElement(
      'button',
      { ...props, type: formType === 'submit' ? 'submit' : 'button' },
      children
    );
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
let props: ProfileEditorViewProps & Omit<ProfileViewProps, 'profile'>;
let editor = true;
beforeEach(async () => {
  editor = true;
  capabilities.supported = false;
  capabilities.choose = undefined;
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
    onOpenEditor: vi.fn(),
    onSaved: vi.fn(),
    onCancel: vi.fn(),
    onOpenInformation: vi.fn(),
    onSaveNickname: vi.fn().mockResolvedValue(true),
    onLogout: vi.fn().mockResolvedValue(false),
    onRetry: vi.fn().mockResolvedValue(true)
  };
  await render();
});
afterEach(async () => {
  await act(async () => root.unmount());
  host.remove();
});
async function render() {
  await act(async () =>
    root.render(
      editor ? createElement(ProfileEditorView, props) : createElement(ProfileView, props)
    )
  );
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
it('opens editing directly from both identity and personal information', async () => {
  editor = false;
  await render();
  await act(async () =>
    host.querySelector<HTMLButtonElement>('.profile-page__identity-button')!.click()
  );
  const row = Array.from(host.querySelectorAll<HTMLButtonElement>('button')).find((b) =>
    b.textContent?.includes('头像与昵称')
  )!;
  await act(async () => row.click());
  expect(props.onOpenEditor).toHaveBeenCalledTimes(2);
  expect(host.querySelector('input')).toBeNull();
});
it('cancels edits without writing personal data', async () => {
  await input('其他名字');
  await click('放弃修改');
  expect(props.onSaveNickname).not.toHaveBeenCalled();
  expect(props.onCancel).toHaveBeenCalledOnce();
});
it('keeps failed edits and returns only after successful save', async () => {
  vi.mocked(props.onSaveNickname).mockResolvedValueOnce(false).mockResolvedValueOnce(true);
  await input(' 新昵称 ');
  await click('保存修改');
  expect(host.querySelector('input')?.value).toBe(' 新昵称 ');
  expect(props.onSaved).not.toHaveBeenCalled();
  await click('保存修改');
  expect(props.onSaveNickname).toHaveBeenLastCalledWith('新昵称', undefined);
  expect(props.onSaved).toHaveBeenCalledOnce();
});
it('keeps editing focused on personal details and disables invalid submissions', async () => {
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
  editor = false;
  await render();
  const entry = Array.from(
    host.querySelectorAll<HTMLButtonElement>('.profile-page__about-row')
  ).find((node) => node.textContent?.includes('关于观潮家'));
  await act(async () => entry!.click());
  expect(props.onOpenInformation).toHaveBeenCalledWith('about');
});

it('keeps chosen avatar local until saving and preserves it after failure', async () => {
  capabilities.supported = true;
  await render();
  await act(async () => capabilities.choose!({ detail: { avatarUrl: 'wxfile://tmp/avatar.jpg' } }));
  expect(props.onSaveNickname).not.toHaveBeenCalled();
  vi.mocked(props.onSaveNickname).mockResolvedValueOnce(false).mockResolvedValueOnce(true);
  await click('保存修改');
  expect(props.onSaveNickname).toHaveBeenLastCalledWith('david', 'wxfile://tmp/avatar.jpg');
  expect(host.querySelector('img')?.getAttribute('src')).toBe('wxfile://tmp/avatar.jpg');
  await click('保存修改');
  expect(props.onSaved).toHaveBeenCalledOnce();
});
it('canceling discards the selected avatar without a write', async () => {
  capabilities.supported = true;
  await render();
  await act(async () => capabilities.choose!({ detail: { avatarUrl: 'wxfile://tmp/avatar.jpg' } }));
  await click('放弃修改');
  expect(props.onCancel).toHaveBeenCalledOnce();
  expect(props.onSaveNickname).not.toHaveBeenCalled();
});

it('opens privacy from the signed-in support group without changing account data', async () => {
  editor = false;
  await render();
  await click('隐私政策');
  expect(props.onOpenInformation).toHaveBeenCalledWith('privacy');
  expect(props.onSaveNickname).not.toHaveBeenCalled();
  expect(props.onLogout).not.toHaveBeenCalled();
});
