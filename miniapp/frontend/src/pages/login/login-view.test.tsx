// @vitest-environment jsdom
import { act, createElement, type ReactNode } from 'react';
import { createRoot, type Root } from 'react-dom/client';
import { afterEach, beforeEach, expect, it, vi } from 'vitest';
import { LoginView, type LoginViewProps } from './login-view';

let phoneHandler: ((e: { detail: { code?: string } }) => void) | undefined;
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
    if (onGetPhoneNumber) phoneHandler = onGetPhoneNumber;
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
let props: LoginViewProps;
beforeEach(async () => {
  Object.assign(globalThis, { IS_REACT_ACT_ENVIRONMENT: true });
  host = document.createElement('div');
  document.body.appendChild(host);
  root = createRoot(host);
  props = {
    pendingAction: null,
    error: '',
    canLogin: true,
    onLogin: vi.fn().mockResolvedValue(true),
    onOpenPrivacy: vi.fn()
  };
  await act(async () => root.render(createElement(LoginView, props)));
});
afterEach(async () => {
  await act(async () => root.unmount());
  host.remove();
});
async function click(label: string) {
  const b = Array.from(host.querySelectorAll('button')).find((n) => n.textContent === label);
  expect(b).toBeDefined();
  await act(async () => b!.click());
}
async function agree() {
  await act(async () => host.querySelector<HTMLInputElement>('input[type=checkbox]')!.click());
}
it('requires explicit consent for primary login and does not check it when opening the policy', async () => {
  await click('一键注册/登录');
  expect(props.onLogin).not.toHaveBeenCalled();
  await click('《观潮家隐私政策》');
  expect(props.onOpenPrivacy).toHaveBeenCalledOnce();
  expect(host.querySelector<HTMLInputElement>('input')!.checked).toBe(false);
  await agree();
  await click('一键注册/登录');
  expect(props.onLogin).toHaveBeenCalledOnce();
});
it('handles canceled and successful phone authorization without bypassing consent', async () => {
  await act(async () => phoneHandler?.({ detail: { code: 'unconsented' } }));
  expect(props.onLogin).not.toHaveBeenCalled();
  await agree();
  await act(async () => phoneHandler?.({ detail: {} }));
  expect(props.onLogin).not.toHaveBeenCalled();
  expect(host.textContent).toContain('未完成手机号授权');
  await act(async () => phoneHandler?.({ detail: { code: 'phone' } }));
  expect(props.onLogin).toHaveBeenCalledWith('phone');
});
it('disables login during pending work and on unsupported platforms', async () => {
  props.pendingAction = 'login';
  await act(async () => root.render(createElement(LoginView, props)));
  const b = Array.from(host.querySelectorAll('button')).find((n) => n.textContent === '正在登录…');
  expect(b?.disabled).toBe(true);
  props.pendingAction = null;
  props.canLogin = false;
  await act(async () => root.render(createElement(LoginView, props)));
  expect(host.textContent).toContain('请在微信小程序中登录');
});
