// @vitest-environment jsdom

import { act, createElement, type ReactNode } from 'react';
import { createRoot, type Root } from 'react-dom/client';
import { afterAll, afterEach, beforeAll, beforeEach, describe, expect, it, vi } from 'vitest';
import { parseReportDetailRoute } from '../features/reports/navigation';
import type { ReportResourceState } from '../features/reports/session';
import { normalizedMockReportPort } from '../mocks/reports/mock-port';
import IndexPage from './index/index';
import ReportDetailPage, { type LoadedReportDetail } from './report/detail/index';

const harness = vi.hoisted(() => ({
  states: new Map<string, ReportResourceState<unknown>>(),
  reads: new Map<string, number>(),
  pageScrollTo: vi.fn(),
  navigateTo: vi.fn(),
  scene: 1001,
  pageCount: 2,
  navigateBack: vi.fn(),
  reLaunch: vi.fn(),
  showToast: vi.fn(),
  friend: vi.fn<(callback: () => { title: string; path: string; imageUrl: string }) => void>(),
  timeline: vi.fn<(callback: () => { title: string; query: string; imageUrl: string }) => void>()
}));

vi.mock('../assets/share-cover.jpg', () => ({ default: 'assets/share-cover.jpg' }));

vi.mock('@tarojs/taro', () => ({
  default: {
    getCurrentInstance: () => ({
      router: {
        params: {
          reportId: 'RPT11111111-1111-4111-8111-111111111111',
          targetType: 'geopolitical_stories',
          targetKey: 'g1'
        }
      }
    }),
    getWindowInfo: () => ({ statusBarHeight: 44, windowWidth: 390 }),
    getMenuButtonBoundingClientRect: () => ({ top: 50, left: 300, width: 80, height: 32 }),
    navigateTo: harness.navigateTo,
    getCurrentPages: () => Array(harness.pageCount).fill({}),
    navigateBack: harness.navigateBack,
    reLaunch: harness.reLaunch,
    getLaunchOptionsSync: () => ({ scene: harness.scene }),
    pageScrollTo: harness.pageScrollTo,
    pxTransform: (value: number) => `${value}px`,
    setNavigationBarTitle: vi.fn(),
    showToast: harness.showToast,
    stopPullDownRefresh: vi.fn()
  },
  usePullDownRefresh: vi.fn(),
  useDidShow: vi.fn(),
  useShareAppMessage: harness.friend,
  useShareTimeline: harness.timeline
}));

function element(tag: string) {
  return ({
    children,
    className,
    ariaLabel,
    onClick,
    disabled
  }: Readonly<{
    children?: ReactNode;
    className?: string;
    ariaLabel?: string;
    onClick?: () => void;
    disabled?: boolean;
  }>) => createElement(tag, { className, 'aria-label': ariaLabel, onClick, disabled }, children);
}

vi.mock('@tarojs/components', () => ({
  Button: element('button'),
  Image: element('img'),
  Input: element('input'),
  RootPortal: element('aside'),
  ScrollView: element('section'),
  Text: element('span'),
  View: element('div')
}));

vi.mock('../features/reports/port', () => ({
  getReportPort: () => ({})
}));

vi.mock('../features/reports/use-report-resource', () => ({
  useReportResource: (resourceKey: string) => {
    harness.reads.set(resourceKey, (harness.reads.get(resourceKey) ?? 0) + 1);
    const state = harness.states.get(resourceKey) ?? { status: 'loading' };
    return {
      state,
      retry: vi.fn(),
      refresh: vi.fn(),
      snapshot: () => state
    };
  }
}));

let root: Root | undefined;
let container: HTMLDivElement | undefined;
const reactActEnvironment = globalThis as typeof globalThis & {
  IS_REACT_ACT_ENVIRONMENT?: boolean;
};

beforeAll(() => {
  reactActEnvironment.IS_REACT_ACT_ENVIRONMENT = true;
});

afterAll(() => {
  delete reactActEnvironment.IS_REACT_ACT_ENVIRONMENT;
});

beforeEach(() => {
  vi.stubEnv('TARO_ENV', 'weapp');
  harness.scene = 1001;
  harness.pageCount = 2;
  harness.navigateBack.mockReset();
  harness.reLaunch.mockReset();
  harness.showToast.mockClear();
  harness.friend.mockClear();
  harness.timeline.mockClear();
  harness.navigateTo.mockClear();
  harness.states.clear();
  harness.reads.clear();
  harness.pageScrollTo.mockClear();
  container = document.createElement('div');
  document.body.append(container);
});

afterEach(() => {
  if (root) act(() => root?.unmount());
  vi.unstubAllEnvs();
  container?.remove();
  root = undefined;
  container = undefined;
});

describe('page-local Report Evidence hosts', () => {
  it('keeps the homepage overlay host and report resource stable when closing by icon', async () => {
    const home = await normalizedMockReportPort.getHome();
    harness.states.set('report-home', {
      status: 'ready',
      data: home,
      refreshing: false,
      refreshFailed: false
    });
    mount(createElement(IndexPage));
    const pageBefore = requiredElement('.home-page');
    const scrollBefore = requiredElement('.normalized-home-scroll');
    const hostBefore = requiredElement('.report-overlay-host');
    scrollBefore.scrollTop = 780;

    click(requiredElement('.normalized-card-evidence'));

    expect(requiredElement('.report-evidence-sheet')).toBeDefined();
    expect(harness.reads.get('report-home')).toBe(1);
    expect(requiredElement('.home-page')).toBe(pageBefore);
    expect(requiredElement('.normalized-home-scroll')).toBe(scrollBefore);
    expect(scrollBefore.scrollTop).toBe(780);

    click(requiredElement('.report-evidence-sheet__close'));

    expect(container?.querySelector('.report-evidence-sheet')).toBeNull();
    expect(requiredElement('.report-overlay-host')).toBe(hostBefore);
    expect(harness.reads.get('report-home')).toBe(1);
    expect(requiredElement('.home-page')).toBe(pageBefore);
    expect(requiredElement('.normalized-home-scroll')).toBe(scrollBefore);
    expect(scrollBefore.scrollTop).toBe(780);
  });

  it('stops mask clicks at the persistent detail overlay host', async () => {
    const detail = await normalizedMockReportPort.getAnalysis(
      'RPT11111111-1111-4111-8111-111111111111',
      'geopolitical_stories',
      'g1'
    );
    const resourceKey =
      'report-detail:RPT11111111-1111-4111-8111-111111111111:geopolitical_stories:g1';
    harness.states.set(resourceKey, {
      status: 'ready',
      data: {
        targetType: 'analysis',
        detail,
        reportId: 'RPT11111111-1111-4111-8111-111111111111',
        kind: 'geopolitical_stories'
      } satisfies LoadedReportDetail,
      refreshing: false,
      refreshFailed: false
    });
    mount(createElement(ReportDetailPage));
    const pageBefore = requiredElement('.normalized-detail');
    const hostBefore = requiredElement('.report-overlay-host');
    const evidenceAction = requiredElement('.normalized-evidence');
    const initialPageScrollCalls = harness.pageScrollTo.mock.calls.length;

    click(evidenceAction);

    expect(requiredElement('.report-evidence-sheet')).toBeDefined();
    expect(harness.reads.get(resourceKey)).toBe(1);
    expect(requiredElement('.normalized-detail')).toBe(pageBefore);
    expect(harness.pageScrollTo).toHaveBeenCalledTimes(initialPageScrollCalls);

    const bubbledClick = vi.fn();
    document.body.addEventListener('click', bubbledClick);
    click(requiredElement('.report-evidence-sheet__overlay'));
    document.body.removeEventListener('click', bubbledClick);

    expect(container?.querySelector('.report-evidence-sheet')).toBeNull();
    expect(requiredElement('.report-overlay-host')).toBe(hostBefore);
    expect(bubbledClick).not.toHaveBeenCalled();
    expect(harness.reads.get(resourceKey)).toBe(1);
    expect(requiredElement('.normalized-detail')).toBe(pageBefore);
    expect(harness.pageScrollTo).toHaveBeenCalledTimes(initialPageScrollCalls);
  });
});

function mount(node: ReactNode): void {
  if (!container) throw new Error('test container is unavailable');
  root = createRoot(container);
  act(() => root?.render(node));
}

function requiredElement(selector: string): HTMLElement {
  const match = container?.querySelector<HTMLElement>(selector);
  if (!match) throw new Error(`expected element matching ${selector}`);
  return match;
}

function click(target: HTMLElement): void {
  act(() => target.click());
}

describe('right-menu report sharing', () => {
  it('shares the homepage without transient parameters and keeps normal navigation', async () => {
    const home = await normalizedMockReportPort.getHome();
    harness.states.set('report-home', {
      status: 'ready',
      data: home,
      refreshing: false,
      refreshFailed: false
    });
    mount(createElement(IndexPage));
    expect(harness.friend.mock.lastCall?.[0]()).toEqual({
      title: '观潮家 · 今日观潮',
      path: '/pages/index/index',
      imageUrl: '/assets/share-cover.jpg'
    });
    expect(harness.timeline.mock.lastCall?.[0]()).toEqual({
      title: '观潮家 · 今日观潮',
      query: '',
      imageUrl: '/assets/share-cover.jpg'
    });
    click(requiredElement('.normalized-card-path'));
    expect(harness.navigateTo).toHaveBeenCalledOnce();
    expect(container?.querySelector('.navigation-bar')).not.toBeNull();
  });

  it('disables single-page navigation while retaining readable content and evidence', async () => {
    harness.scene = 1154;
    const home = await normalizedMockReportPort.getHome();
    harness.states.set('report-home', {
      status: 'ready',
      data: home,
      refreshing: false,
      refreshFailed: false
    });
    mount(createElement(IndexPage));
    const detailButton = requiredElement('.normalized-card-path') as HTMLButtonElement;
    expect(detailButton.disabled).toBe(true);
    click(detailButton);
    expect(harness.navigateTo).not.toHaveBeenCalled();
    expect(container?.querySelector('.navigation-bar')).toBeNull();
    expect(container?.querySelector('.home-hero-spacer')).toBeNull();
    expect(requiredElement('.normalized-card-conclusion').textContent).not.toBe('');
    click(requiredElement('.normalized-card-evidence'));
    expect(requiredElement('.report-evidence-sheet')).toBeDefined();
  });

  it('does not apply WeChat single-page restrictions to tt', async () => {
    vi.stubEnv('TARO_ENV', 'tt');
    harness.scene = 1154;
    const home = await normalizedMockReportPort.getHome();
    harness.states.set('report-home', {
      status: 'ready',
      data: home,
      refreshing: false,
      refreshFailed: false
    });
    mount(createElement(IndexPage));
    expect((requiredElement('.normalized-card-path') as HTMLButtonElement).disabled).toBe(false);
    expect(container?.querySelector('.navigation-bar')).not.toBeNull();
  });

  it('keeps the exact detail route through loading and refreshes the share title when ready', async () => {
    mount(createElement(ReportDetailPage));
    const before = harness.friend.mock.lastCall?.[0]();
    expect(before?.title).toBe('观潮家 · 深度分析');
    const detail = await normalizedMockReportPort.getAnalysis(
      'RPT11111111-1111-4111-8111-111111111111',
      'geopolitical_stories',
      'g1'
    );
    harness.states.set(
      'report-detail:RPT11111111-1111-4111-8111-111111111111:geopolitical_stories:g1',
      {
        status: 'ready',
        refreshing: false,
        refreshFailed: false,
        data: {
          targetType: 'analysis',
          detail,
          reportId: 'RPT11111111-1111-4111-8111-111111111111',
          kind: 'geopolitical_stories'
        } satisfies LoadedReportDetail
      }
    );
    act(() => root?.render(createElement(ReportDetailPage)));
    const friend = harness.friend.mock.lastCall?.[0]();
    const timeline = harness.timeline.mock.lastCall?.[0]();
    expect(friend?.title).toBe(`${detail.summary.title} · 观潮家`);
    expect(timeline?.title).toBe(friend?.title);
    expect(friend?.imageUrl).toBe('/assets/share-cover.jpg');
    expect(timeline?.imageUrl).toBe(friend?.imageUrl);
    expect(before?.imageUrl).toBe(friend?.imageUrl);
    expect(friend?.path).toBe(before?.path);
    expect(friend?.path).toBe(`/pages/report/detail/index?${timeline?.query}`);
    expect(
      parseReportDetailRoute(Object.fromEntries(new URLSearchParams(timeline?.query)))
    ).toEqual({
      reportId: 'RPT11111111-1111-4111-8111-111111111111',
      targetType: 'geopolitical_stories',
      targetKey: 'g1'
    });
  });
});

describe('shared navigation', () => {
  it('renders the shared title on the homepage and detail loading state', () => {
    mount(createElement(IndexPage));
    expect(requiredElement('.navigation-bar__title').textContent).toBe('观潮家');
    act(() => root?.render(createElement(ReportDetailPage)));
    expect(requiredElement('.navigation-bar__title').textContent).toBe('深度分析');
  });
  it('returns to the previous page', async () => {
    mount(createElement(ReportDetailPage));
    await act(async () => requiredElement('.report-detail-navigation__back').click());
    expect(harness.navigateBack).toHaveBeenCalledWith({ delta: 1 });
    expect(harness.reLaunch).not.toHaveBeenCalled();
  });
  it('returns to home when the shared detail has no previous page', async () => {
    harness.pageCount = 1;
    mount(createElement(ReportDetailPage));
    await act(async () => requiredElement('.report-detail-navigation__back').click());
    expect(harness.reLaunch).toHaveBeenCalledWith({ url: '/pages/index/index' });
    expect(harness.navigateBack).not.toHaveBeenCalled();
  });
  it('shows a retry message when returning fails', async () => {
    harness.navigateBack.mockRejectedValueOnce(new Error('navigation failed'));
    mount(createElement(ReportDetailPage));
    await act(async () => requiredElement('.report-detail-navigation__back').click());
    expect(harness.showToast).toHaveBeenCalledWith({ title: '返回失败，请重试', icon: 'none' });
  });
  it('leaves timeline single-page navigation to the system', () => {
    vi.stubEnv('TARO_ENV', 'weapp');
    harness.scene = 1154;
    mount(createElement(ReportDetailPage));
    expect(container?.querySelector('.navigation-bar')).toBeNull();
  });
});
