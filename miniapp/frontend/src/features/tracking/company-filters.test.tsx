// @vitest-environment jsdom
import {
  act,
  createElement,
  type ReactNode,
  type HTMLAttributes,
  type ButtonHTMLAttributes,
  type InputHTMLAttributes,
  type FormEvent
} from 'react';
import { createRoot, type Root } from 'react-dom/client';
import { afterEach, beforeEach, expect, it, vi } from 'vitest';
import { hideOverlayNavigation, restoreOverlayNavigation } from '../../platform/overlay-navigation';
import { CompanyFilters } from './company-filters';
import { emptyFilters, type FilterOptions } from './contract';

type ContainerProps = HTMLAttributes<HTMLDivElement> & {
  children?: ReactNode;
  scrollY?: boolean;
  onScrollToLower?: () => void;
};
type InputProps = Omit<InputHTMLAttributes<HTMLInputElement>, 'onInput'> & {
  maxlength?: number;
  onInput: (e: { detail: { value: string } }) => void;
};
vi.mock('@tarojs/components', () => ({
  View: ({ children, ...p }: ContainerProps) => createElement('div', p, children),
  Text: ({ children, ...p }: ContainerProps) => createElement('span', p, children),
  Button: ({
    children,
    ariaLabel,
    ...p
  }: ButtonHTMLAttributes<HTMLButtonElement> & { ariaLabel?: string }) =>
    createElement('button', { 'aria-label': ariaLabel, ...p }, children),
  Input: ({ onInput, maxlength, ...p }: InputProps) =>
    createElement('input', {
      maxLength: maxlength,
      ...p,
      onInput: (e: FormEvent<HTMLInputElement>) =>
        onInput({ detail: { value: e.currentTarget.value } })
    }),
  ScrollView: ({ children, scrollY, onScrollToLower, ...p }: ContainerProps) =>
    createElement('div', p, children)
}));
vi.mock('../../platform/overlay-root', () => ({
  OverlayRoot: ({ children }: { children?: ReactNode }) => children
}));
vi.mock('../../platform/overlay-navigation', () => ({
  hideOverlayNavigation: vi.fn().mockResolvedValue(undefined),
  restoreOverlayNavigation: vi.fn().mockResolvedValue(undefined)
}));
vi.mock('./api', () => ({ loadFilterOptions: vi.fn() }));
let root: Root;
let container: HTMLDivElement;
const options: FilterOptions = {
  industries: [{ id: 'i', name: '信息技术', children: [{ id: 'j', name: '软件与服务' }] }],
  concepts: [{ id: 'c', name: '人工智能' }],
  industry_chains: [{ id: 'h', name: '芯片产业链' }]
};
beforeEach(() => {
  Object.assign(globalThis, { IS_REACT_ACT_ENVIRONMENT: true });
  container = document.createElement('div');
  document.body.append(container);
  root = createRoot(container);
});
afterEach(async () => {
  await act(async () => root.unmount());
  container.remove();
});
async function click(label: string) {
  const b = Array.from(container.querySelectorAll('button')).find((x) => x.textContent === label);
  expect(b).toBeTruthy();
  await act(async () => b!.click());
}
it('keeps drafts private until apply, cancels safely and resets only the current dimension', async () => {
  const apply = vi.fn();
  const value = { ...emptyFilters(), concept_ids: ['c'] };
  const load = vi.fn().mockResolvedValue(options);
  await act(async () =>
    root.render(createElement(CompanyFilters, { value, onApply: apply, load }))
  );
  await click('行业⌄');
  await click('软件与服务信息技术＋');
  expect(apply).not.toHaveBeenCalled();
  await click('取消');
  expect(apply).not.toHaveBeenCalled();
  await click('行业⌄');
  await click('软件与服务信息技术＋');
  await click('应用筛选（1）');
  expect(apply).toHaveBeenLastCalledWith({ ...value, industry_ids: ['j'] });
  await click('概念 · 1⌄');
  await click('重置本类');
  await click('应用筛选（0）');
  expect(apply).toHaveBeenLastCalledWith(emptyFilters());
  expect(load).toHaveBeenCalledTimes(1);
});
it('shows a retryable failure instead of an empty catalog', async () => {
  const load = vi.fn().mockRejectedValueOnce(new Error('offline')).mockResolvedValue(options);
  await act(async () =>
    root.render(createElement(CompanyFilters, { value: emptyFilters(), onApply: vi.fn(), load }))
  );
  await click('产业链⌄');
  expect(container.textContent).toContain('暂不可用');
  expect(container.querySelector('button[disabled]')).not.toBeNull();
  await click('重试加载');
  expect(container.textContent).toContain('芯片产业链');
});

it('closes a filter that cannot hide native navigation and restores navigation on cancel', async () => {
  vi.mocked(hideOverlayNavigation).mockRejectedValueOnce(new Error('unsupported'));
  await act(async () =>
    root.render(
      createElement(CompanyFilters, {
        value: emptyFilters(),
        onApply: vi.fn(),
        load: async () => options
      })
    )
  );
  await click('行业⌄');
  expect(container.textContent).toContain('暂时无法打开筛选');
  expect(container.querySelector('.company-filter-overlay')).toBeNull();
  expect(restoreOverlayNavigation).toHaveBeenCalled();
});
