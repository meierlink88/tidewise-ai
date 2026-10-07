// @vitest-environment jsdom
import { act, createElement, type ReactNode } from 'react';
import { createRoot, type Root } from 'react-dom/client';
import { beforeEach, afterEach, expect, it, vi } from 'vitest';
import { CompanyCard } from './company-card';
import type { Company } from './contract';

const navigateTo = vi.hoisted(() => vi.fn().mockResolvedValue(undefined));
vi.mock('@tarojs/taro', () => ({ default: { navigateTo, showToast: vi.fn() } }));
vi.mock('@tarojs/components', () => ({
  View: 'div',
  Text: 'span',
  Image: () => null,
  Button: ({ children, ariaLabel, ...props }: { children?: ReactNode; ariaLabel?: string }) =>
    createElement('button', { ...props, 'aria-label': ariaLabel }, children)
}));
vi.mock('../design-preview/shell', () => ({
  PreviewSheet: ({ children }: { children: ReactNode }) => createElement('section', {}, children)
}));
let root: Root;
let host: HTMLDivElement;
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
beforeEach(() => {
  Object.assign(globalThis, { IS_REACT_ACT_ENVIRONMENT: true });
  vi.clearAllMocks();
  host = document.createElement('div');
  root = createRoot(host);
});
afterEach(() => act(() => root.unmount()));
it('keeps tracking separate and navigates every company to its report detail', async () => {
  const follow = vi.fn();
  await act(async () =>
    root.render(<CompanyCard company={company} busy={false} pending={false} onFollow={follow} />)
  );
  await act(async () =>
    host.querySelector<HTMLButtonElement>('[aria-label="跟踪平安银行"]')!.click()
  );
  expect(follow).toHaveBeenCalledOnce();
  expect(host.textContent).not.toContain('跟踪中');
  await act(async () =>
    host.querySelector<HTMLButtonElement>('[aria-label="查看平安银行洞察报告"]')!.click()
  );
  expect(host.textContent).not.toContain('暂无报告样例');
  const url = navigateTo.mock.calls.at(-1)![0].url;
  const params = new URLSearchParams(url.split('?')[1]);
  expect(params.get('symbol')).toBe('000001.SZ');
  expect(JSON.parse(params.get('display')!)).toEqual({
    stockName: company.stock_name,
    companyName: company.title
  });
  await act(async () =>
    root.render(
      <CompanyCard
        key='sample'
        company={{ ...company, symbol: '603179.SH', is_followed: true }}
        busy={false}
        pending={false}
      />
    )
  );
  expect(host.textContent).toContain('跟踪中');
  await act(async () =>
    host.querySelector<HTMLButtonElement>('[aria-label="查看平安银行洞察报告"]')!.click()
  );
  expect(navigateTo.mock.calls.at(-1)![0].url).toContain(
    '/pages/company/report/index?symbol=603179.SH&display='
  );
});
it('offers cancellation on the personal list and disables it while submitting', async () => {
  const remove = vi.fn();
  await act(async () =>
    root.render(
      <CompanyCard company={{ ...company, is_followed: true }} busy pending onRemove={remove} />
    )
  );
  const button = host.querySelector<HTMLButtonElement>('[aria-label="取消跟踪平安银行"]')!;
  expect(button.disabled).toBe(true);
  expect(host.textContent).not.toContain('＋ 跟踪');
  await act(async () =>
    root.render(
      <CompanyCard
        company={{ ...company, is_followed: true }}
        busy={false}
        pending={false}
        onRemove={remove}
      />
    )
  );
  await act(async () => button.click());
  expect(remove).toHaveBeenCalledOnce();
});
