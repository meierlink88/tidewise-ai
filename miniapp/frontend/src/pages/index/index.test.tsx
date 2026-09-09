import { Children, isValidElement, type ReactNode } from 'react';
import { renderToStaticMarkup } from 'react-dom/server';
import { describe, expect, it, vi } from 'vitest';
import { HomeHeader } from './components/home-header';
import { normalizedMockReportPort } from '../../mocks/reports/mock-port';
import { IndexView, stopHomeRefresh } from './index';
import { NormalizedHome } from './normalized-home';

vi.mock('@tarojs/taro', () => ({ default: {}, usePullDownRefresh: vi.fn() }));
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
  const props = {
    chrome: { statusBarHeight: 44, navigationBarHeight: 44, rightReservedWidth: 102 },
    query: '',
    onQueryChange: vi.fn()
  };
  it('uses the report publication in Shanghai, including date and weekday rollover', () => {
    const html = renderToStaticMarkup(<HomeHeader {...props} publishedAt='2026-12-31T18:05:00Z' />);
    expect(html).toContain('01.01 周五');
    expect(html).toContain('截至 02:05');
    expect(html).not.toContain('过去24小时');
    expect(html).not.toContain('07.07');
  });
  it('does not fabricate a date or cutoff without a report', () => {
    const html = renderToStaticMarkup(<HomeHeader {...props} />);
    expect(html).toContain('全球政经事件');
    expect(html).not.toContain('截至');
    expect(html).not.toContain('周');
  });
});
