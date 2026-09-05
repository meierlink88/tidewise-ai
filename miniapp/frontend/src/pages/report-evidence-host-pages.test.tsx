// @vitest-environment jsdom

import { act, createElement, type ReactNode } from 'react';
import { createRoot, type Root } from 'react-dom/client';
import { afterAll, afterEach, beforeAll, beforeEach, describe, expect, it, vi } from 'vitest';
import type { ReportResourceState } from '../features/reports/session';
import { mockReportPort } from '../mocks/reports/mock-port';
import IndexPage from './index/index';
import ReportDetailPage, { type LoadedReportDetail } from './report/detail/index';

const harness = vi.hoisted(() => ({
  states: new Map<string, ReportResourceState<unknown>>(),
  reads: new Map<string, number>(),
  pageScrollTo: vi.fn()
}));

vi.mock('@tarojs/taro', () => ({
  default: {
    getCurrentInstance: () => ({
      router: {
        params: {
          reportId: 'RPT11111111-1111-4111-8111-111111111111',
          targetType: 'industry_chain',
          targetKey: 'chn-01'
        }
      }
    }),
    getWindowInfo: () => ({ statusBarHeight: 44, windowWidth: 390 }),
    getMenuButtonBoundingClientRect: () => ({ top: 50, left: 300, width: 80, height: 32 }),
    navigateTo: vi.fn(),
    pageScrollTo: harness.pageScrollTo,
    pxTransform: (value: number) => `${value}px`,
    setNavigationBarTitle: vi.fn(),
    showToast: vi.fn(),
    stopPullDownRefresh: vi.fn()
  },
  usePullDownRefresh: vi.fn()
}));

function element(tag: string) {
  return ({
    children,
    className,
    ariaLabel,
    onClick
  }: Readonly<{
    children?: ReactNode;
    className?: string;
    ariaLabel?: string;
    onClick?: () => void;
  }>) => createElement(tag, { className, 'aria-label': ariaLabel, onClick }, children);
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
  harness.states.clear();
  harness.reads.clear();
  harness.pageScrollTo.mockClear();
  container = document.createElement('div');
  document.body.append(container);
});

afterEach(() => {
  if (root) act(() => root?.unmount());
  container?.remove();
  root = undefined;
  container = undefined;
});

describe('page-local Report Evidence hosts', () => {
  it('keeps the homepage overlay host and report resource stable when closing by icon', async () => {
    const home = await mockReportPort.getHome();
    harness.states.set('report-home', {
      status: 'ready',
      data: home,
      refreshing: false,
      refreshFailed: false
    });
    mount(createElement(IndexPage));
    const pageBefore = requiredElement('.home-page');
    const scrollBefore = requiredElement('.home-report-scroll');
    const hostBefore = requiredElement('.report-overlay-host');
    scrollBefore.scrollTop = 780;

    click(requiredElement('[aria-label="查看地缘政治依据"]'));

    expect(requiredElement('.report-evidence-sheet')).toBeDefined();
    expect(harness.reads.get('report-home')).toBe(1);
    expect(requiredElement('.home-page')).toBe(pageBefore);
    expect(requiredElement('.home-report-scroll')).toBe(scrollBefore);
    expect(scrollBefore.scrollTop).toBe(780);

    click(requiredElement('.report-evidence-sheet__close'));

    expect(container?.querySelector('.report-evidence-sheet')).toBeNull();
    expect(requiredElement('.report-overlay-host')).toBe(hostBefore);
    expect(harness.reads.get('report-home')).toBe(1);
    expect(requiredElement('.home-page')).toBe(pageBefore);
    expect(requiredElement('.home-report-scroll')).toBe(scrollBefore);
    expect(scrollBefore.scrollTop).toBe(780);
  });

  it('stops mask clicks at the persistent detail overlay host', async () => {
    const detail = await mockReportPort.getIndustryChain(
      'RPT11111111-1111-4111-8111-111111111111',
      'chn-01'
    );
    const resourceKey =
      'report-detail:RPT11111111-1111-4111-8111-111111111111:industry_chain:chn-01';
    harness.states.set(resourceKey, {
      status: 'ready',
      data: { targetType: 'industry_chain', detail } satisfies LoadedReportDetail,
      refreshing: false,
      refreshFailed: false
    });
    mount(createElement(ReportDetailPage));
    const pageBefore = requiredElement('.report-detail-page');
    const hostBefore = requiredElement('.report-overlay-host');
    const evidenceAction = requiredElement('[aria-label$="证据：依据"]');
    const initialPageScrollCalls = harness.pageScrollTo.mock.calls.length;

    click(evidenceAction);

    expect(requiredElement('.report-evidence-sheet')).toBeDefined();
    expect(harness.reads.get(resourceKey)).toBe(1);
    expect(requiredElement('.report-detail-page')).toBe(pageBefore);
    expect(harness.pageScrollTo).toHaveBeenCalledTimes(initialPageScrollCalls);

    const bubbledClick = vi.fn();
    document.body.addEventListener('click', bubbledClick);
    click(requiredElement('.report-evidence-sheet__overlay'));
    document.body.removeEventListener('click', bubbledClick);

    expect(container?.querySelector('.report-evidence-sheet')).toBeNull();
    expect(requiredElement('.report-overlay-host')).toBe(hostBefore);
    expect(bubbledClick).not.toHaveBeenCalled();
    expect(harness.reads.get(resourceKey)).toBe(1);
    expect(requiredElement('.report-detail-page')).toBe(pageBefore);
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
