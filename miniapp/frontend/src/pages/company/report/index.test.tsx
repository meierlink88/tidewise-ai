// @vitest-environment jsdom
import { act, createElement, type ReactNode } from 'react';
import { createRoot, type Root } from 'react-dom/client';
import { afterEach, beforeEach, expect, it, vi } from 'vitest';
import CompanyReportPage from './index';

const state = vi.hoisted(() => ({
  symbol: undefined as string | undefined,
  display: undefined as string | undefined,
  allowed: true,
  navigateBack: vi.fn().mockResolvedValue(undefined),
  switchTab: vi.fn().mockResolvedValue(undefined),
  pages: [{}, {}]
}));
vi.mock('@tarojs/taro', () => ({
  useRouter: () => ({ params: { symbol: state.symbol, display: state.display } }),
  default: {
    getCurrentPages: () => state.pages,
    navigateBack: state.navigateBack,
    switchTab: state.switchTab,
    showToast: vi.fn()
  }
}));
vi.mock('@tarojs/components', () => ({
  View: 'div',
  ScrollView: ({ children, className }: { children: ReactNode; className?: string }) =>
    createElement('div', { className }, children),
  Text: 'span',
  Button: ({ children, ariaLabel, ...props }: { children?: ReactNode; ariaLabel?: string }) =>
    createElement('button', { ...props, 'aria-label': ariaLabel }, children)
}));
vi.mock('../../../features/identity/use-research-access', () => ({
  useResearchAccess: () => ({ allowed: state.allowed, error: '', retry: vi.fn() })
}));
vi.mock('../../../features/identity/research-access-state', () => ({
  ResearchAccessState: () => createElement('span', {}, '验证登录')
}));
vi.mock('../../../platform/system-ui', () => ({ getHomeChromeMetrics: () => ({}) }));
vi.mock('../../../platform/navigation-bar', () => ({
  NavigationBar: ({ title, leading }: { title: string; leading: ReactNode }) =>
    createElement('nav', {}, leading, title)
}));
vi.mock('./standard/StandardReport', () => ({
  default: () => createElement('article', {}, '新泉股份样例正文')
}));
let host: HTMLDivElement, root: Root;
beforeEach(() => {
  Object.assign(globalThis, { IS_REACT_ACT_ENVIRONMENT: true });
  state.allowed = true;
  state.symbol = undefined;
  state.display = undefined;
  state.pages = [{}, {}];
  vi.clearAllMocks();
  host = document.createElement('div');
  root = createRoot(host);
});
afterEach(() => act(() => root.unmount()));
it.each(['000001.SZ', undefined, '603179', '603179.SH&symbol=000001.SZ'])(
  'shows no report for unavailable or invalid identities: %s',
  (symbol) => {
    state.symbol = symbol;
    act(() => root.render(<CompanyReportPage />));
    expect(host.textContent).toContain('洞察报告');
    expect(host.textContent).toContain('暂无报告');
    expect(host.querySelector('article')).toBeNull();
  }
);
it('renders the sample only for its complete symbol, including repeat entry', () => {
  state.symbol = '603179.SH';
  act(() => root.render(<CompanyReportPage />));
  expect(host.textContent).toContain('新泉股份样例正文');
  expect(host.textContent).not.toContain('暂无报告');
  state.symbol = '000001.SZ';
  act(() => root.render(<CompanyReportPage />));
  expect(host.textContent).toContain('暂无报告');
  expect(host.querySelector('article')).toBeNull();
});
it('hides all report content until login is validated', () => {
  state.symbol = '603179.SH';
  state.allowed = false;
  act(() => root.render(<CompanyReportPage />));
  expect(host.textContent).toBe('验证登录');
});
it('returns empty reports to the previous page, or company tab for direct entry', async () => {
  act(() => root.render(<CompanyReportPage />));
  await act(async () => host.querySelector<HTMLButtonElement>('button')!.click());
  expect(state.navigateBack).toHaveBeenCalledOnce();
  state.pages = [{}];
  await act(async () => host.querySelector<HTMLButtonElement>('button')!.click());
  expect(state.switchTab).toHaveBeenCalledWith({ url: '/pages/company/index' });
});

it('identifies the selected stock and returns from the empty-state action', async () => {
  state.symbol = '000980.SZ';
  act(() => root.render(<CompanyReportPage />));
  expect(host.textContent).toContain('000980.SZ');
  expect(host.textContent).toContain('当前暂无可展示的洞察报告内容');
  await act(async () =>
    host.querySelector<HTMLButtonElement>('[aria-label="返回股票列表"]')!.click()
  );
  expect(state.navigateBack).toHaveBeenCalledOnce();
  state.symbol = 'bad-input';
  act(() => root.render(<CompanyReportPage />));
  expect(host.textContent).not.toContain('股票代码');
  state.pages = [{}];
  await act(async () =>
    host.querySelector<HTMLButtonElement>('[aria-label="返回股票列表"]')!.click()
  );
  expect(state.switchTab).toHaveBeenCalledWith({ url: '/pages/company/index' });
});

it('shows the stock and full company names without borrowing another report identity', () => {
  state.symbol = '000980.SZ';
  state.display = encodeURIComponent(
    JSON.stringify({ stockName: '众泰汽车', companyName: '众泰汽车股份有限公司' })
  );
  act(() => root.render(<CompanyReportPage />));
  expect(host.textContent).toContain('众泰汽车');
  expect(host.textContent).toContain('众泰汽车股份有限公司');
  expect(host.textContent).toContain('000980.SZ');
  state.symbol = 'bad';
  act(() => root.render(<CompanyReportPage />));
  expect(host.textContent).not.toContain('众泰汽车');
});
