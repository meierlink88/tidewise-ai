import { Children, isValidElement, type ReactNode } from 'react';
import { renderToStaticMarkup } from 'react-dom/server';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { HomeHeader } from './components/home-header';
import { normalizedMockReportPort } from '../../mocks/reports/mock-port';
import { IndexView, stopHomeRefresh } from './index';
import { NormalizedHome } from './normalized-home';

vi.mock('@tarojs/taro', () => ({ default: {}, usePullDownRefresh: vi.fn(), useDidShow: vi.fn() }));
vi.mock('@tarojs/components', () => ({
  View: 'view',
  Text: 'text',
  Input: 'input',
  Button: 'button',
  Image: 'image',
  ScrollView: 'scroll-view'
}));
function find(node: ReactNode): boolean {
  return Children.toArray(node).some(
    (child) =>
      isValidElement<{ children?: ReactNode }>(child) &&
      (child.type === NormalizedHome || find(child.props.children))
  );
}
it('routes the current report to the normalized home without legacy pagination', async () => {
  const home = await normalizedMockReportPort.getHome();
  const page = IndexView({
    chrome: { statusBarHeight: 44, navigationBarHeight: 44, rightReservedWidth: 102 },
    query: '',
    onQueryChange: vi.fn(),
    state: { status: 'ready', data: home, refreshing: false, refreshFailed: false },
    onRetry: vi.fn(),
    onRefresh: vi.fn(),
    onOpenDetail: vi.fn(),
    onOpenEvidence: vi.fn()
  });
  expect(find(page)).toBe(true);
});
describe('home refresh', () => {
  it('stops the native refresh indicator', async () => {
    const stopPullDownRefresh = vi.fn();
    await stopHomeRefresh({ stopPullDownRefresh, showToast: vi.fn() });
    expect(stopPullDownRefresh).toHaveBeenCalledOnce();
  });
});

describe('home publication header', () => {
  beforeEach(() => {
    vi.useFakeTimers();
    vi.setSystemTime(new Date('2026-09-11T16:05:00Z'));
  });
  afterEach(() => vi.useRealTimers());
  const props = {
    chrome: { statusBarHeight: 44, navigationBarHeight: 44, rightReservedWidth: 102 },
    query: '',
    onQueryChange: vi.fn()
  };
  it('uses the current Shanghai date independently of the report publication', () => {
    const html = renderToStaticMarkup(<HomeHeader {...props} publishedAt='2026-12-31T18:05:00Z' />);
    expect(html).toContain('09.12 周六');
    expect(html).toContain('截至 01.01 02:05');
    expect(html).not.toContain('过去24小时');
    expect(html).not.toContain('07.07');
  });
  it('hides invalid publication timestamps', () => {
    const html = renderToStaticMarkup(<HomeHeader {...props} publishedAt='invalid' />);
    expect(html).toContain('09.12 周六');
    expect(html).not.toContain('截至');
  });
  it('shows the current date without fabricating a cutoff without a report', () => {
    const html = renderToStaticMarkup(<HomeHeader {...props} />);
    expect(html).toContain('全球政经事件');
    expect(html).not.toContain('截至');
    expect(html).toContain('09.12 周六');
  });
});
