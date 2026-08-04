import { Children, isValidElement, type ReactElement, type ReactNode } from 'react';
import { describe, expect, it, vi } from 'vitest';
import {
  mockResearchThemeDetail,
  mockResearchThemeFeed
} from '../../mocks/research-themes/mock-port';
import type { ResearchThemeHomepagePort } from '../../features/research-themes/contract';
import {
  ResearchThemeHomeSession,
  type ResearchThemeHomeSessionState
} from '../../features/research-themes/session';
import {
  IndexView,
  loadMoreAtPageBottom,
  navigateThemePeriod,
  refreshHomeFeed
} from './theme-list-page';

vi.mock('@tarojs/taro', () => ({
  default: {
    getSystemInfoSync: vi.fn(() => ({ statusBarHeight: 44, screenWidth: 390 })),
    getMenuButtonBoundingClientRect: vi.fn(() => ({ top: 50, bottom: 82, height: 32 })),
    navigateTo: vi.fn(),
    showToast: vi.fn(),
    stopPullDownRefresh: vi.fn()
  },
  usePullDownRefresh: vi.fn(),
  useReachBottom: vi.fn()
}));

vi.mock('@tarojs/components', () => ({
  Button: 'button',
  Image: 'image',
  Input: 'input',
  ScrollView: 'scroll-view',
  Text: 'text',
  View: 'view'
}));

describe('Theme homepage', () => {
  it('wires today/history navigation and history-only bottom loading', () => {
    const api = { navigateTo: vi.fn(), navigateBack: vi.fn() };
    const session = { loadMore: vi.fn().mockResolvedValue('updated') };

    navigateThemePeriod('today', api);
    navigateThemePeriod('history', api);
    loadMoreAtPageBottom('today', session as Pick<ResearchThemeHomeSession, 'loadMore'>);
    loadMoreAtPageBottom('history', session as Pick<ResearchThemeHomeSession, 'loadMore'>);

    expect(api.navigateTo).toHaveBeenCalledWith({
      url: '/pages/research-theme/history/index'
    });
    expect(api.navigateBack).toHaveBeenCalledOnce();
    expect(session.loadMore).toHaveBeenCalledOnce();
  });
  it('renders the Theme feed without the removed static category and tracking bar', () => {
    const state: ResearchThemeHomeSessionState = {
      feed: { status: 'ready', value: mockResearchThemeFeed },
      pagination: 'exhausted',
      selectedThemeId: null,
      detailsByThemeId: {}
    };

    const page = IndexView({
      state,
      query: '',
      chrome: { statusBarHeight: 44, navigationBarHeight: 44, rightReservedWidth: 16 },
      onQueryChange: vi.fn(),
      onRetryFeed: vi.fn(),
      onOpenEvents: vi.fn(),
      onCloseEvents: vi.fn(),
      onRetryEvents: vi.fn()
    });

    expect(findAllByClass(page, 'category-bar')).toEqual([]);
    expect(textContent(page)).not.toContain('跟踪中');
    expect(textContent(page)).toContain('今日主题');
    expect(textContent(page)).toContain(mockResearchThemeFeed.items[0].oneLineConclusion);
    const periodAction = findByClass(page, 'home-history-button');
    expect(periodAction.props.ariaLabel).toBe('查看历史主题');
    expect(textContent(periodAction)).toBe('');
  });

  it('preserves the last feed and always stops native refresh when refresh fails', async () => {
    const port: ResearchThemeHomepagePort = {
      list: vi.fn().mockResolvedValueOnce(mockResearchThemeFeed).mockRejectedValueOnce(new Error()),
      getDetail: vi.fn()
    };
    const session = new ResearchThemeHomeSession(port);
    const api = {
      showToast: vi.fn(),
      stopPullDownRefresh: vi.fn()
    };
    await session.start();

    await refreshHomeFeed(session, api);

    expect(session.getState().feed).toEqual({ status: 'ready', value: mockResearchThemeFeed });
    expect(api.showToast).toHaveBeenCalledWith({
      title: '刷新失败，请稍后重试',
      icon: 'none',
      duration: 1600
    });
    expect(api.stopPullDownRefresh).toHaveBeenCalledOnce();
  });

  it('includes newly appended history items in the active search', async () => {
    const appendedTheme = {
      ...mockResearchThemeFeed.items[0],
      id: 'bbbbbbbb-1111-4111-8111-111111111111',
      title: '历史独有主题'
    };
    const port: ResearchThemeHomepagePort = {
      list: vi
        .fn()
        .mockResolvedValueOnce({ ...mockResearchThemeFeed, nextCursor: 'next-page' })
        .mockResolvedValueOnce({
          ...mockResearchThemeFeed,
          items: [appendedTheme],
          nextCursor: null
        }),
      getDetail: vi.fn()
    };
    const session = new ResearchThemeHomeSession(port, { period: 'history' });
    await session.start();
    await session.loadMore();

    const page = IndexView({
      state: session.getState(),
      period: 'history',
      query: '历史独有',
      chrome: { statusBarHeight: 44, navigationBarHeight: 44, rightReservedWidth: 16 },
      onQueryChange: vi.fn(),
      onRetryFeed: vi.fn(),
      onOpenEvents: vi.fn(),
      onCloseEvents: vi.fn(),
      onRetryEvents: vi.fn()
    });

    expect(textContent(page)).toContain('历史独有主题');
    expect(textContent(page)).not.toContain(mockResearchThemeFeed.items[0].title);
  });

  it('opens the event timeline through the page interaction and closes it independently', async () => {
    const port: ResearchThemeHomepagePort = {
      list: vi.fn().mockResolvedValue(mockResearchThemeFeed),
      getDetail: vi.fn().mockResolvedValue(mockResearchThemeDetail)
    };
    const session = new ResearchThemeHomeSession(port);
    await session.start();
    const render = () =>
      IndexView({
        state: session.getState(),
        query: '',
        chrome: { statusBarHeight: 44, navigationBarHeight: 44, rightReservedWidth: 16 },
        onQueryChange: vi.fn(),
        onRetryFeed: () => void session.retryFeed(),
        onOpenEvents: (themeId) => session.openThemeEvents(themeId),
        onCloseEvents: () => session.closeThemeEvents(),
        onRetryEvents: () => session.retryThemeEvents()
      });

    findByClass(render(), 'theme-card__event-button').props.onClick?.(tapEvent());

    const loadingPage = render();
    expect(findByClass(loadingPage, 'theme-card__event-action').props.catchMove).toBe(true);
    expect(findByClass(loadingPage, 'theme-events-overlay').props.catchMove).toBe(true);
    expect(textContent(loadingPage)).toContain('正在整理关联事件');

    await flushPromises();
    const readyPage = render();
    expect(textContent(readyPage)).toContain('端口计划上调');
    expect(textContent(readyPage)).toContain('时间待确认');

    findByClass(readyPage, 'theme-events-sheet__close').props.onClick?.(tapEvent());
    expect(findAllByClass(render(), 'theme-events-overlay')).toEqual([]);
    expect(port.getDetail).toHaveBeenCalledOnce();
  });

  it('exposes a visible retry action for an initial feed error', () => {
    const onRetryFeed = vi.fn();
    const page = IndexView({
      state: {
        feed: { status: 'error' },
        pagination: 'idle',
        selectedThemeId: null,
        detailsByThemeId: {}
      },
      query: '',
      chrome: { statusBarHeight: 44, navigationBarHeight: 44, rightReservedWidth: 16 },
      onQueryChange: vi.fn(),
      onRetryFeed,
      onOpenEvents: vi.fn(),
      onCloseEvents: vi.fn(),
      onRetryEvents: vi.fn()
    });

    findByClass(page, 'home-state__retry').props.onClick?.(tapEvent());

    expect(textContent(page)).toContain('主题数据暂时不可用');
    expect(textContent(page)).toContain('重新加载');
    expect(onRetryFeed).toHaveBeenCalledOnce();
  });

  it('labels the history action and provides an explicit return-to-today action', () => {
    const onPeriodAction = vi.fn();
    const page = IndexView({
      state: {
        feed: { status: 'ready', value: mockResearchThemeFeed },
        pagination: 'exhausted',
        selectedThemeId: null,
        detailsByThemeId: {}
      },
      period: 'history',
      query: '',
      chrome: { statusBarHeight: 44, navigationBarHeight: 44, rightReservedWidth: 16 },
      onQueryChange: vi.fn(),
      onRetryFeed: vi.fn(),
      onOpenEvents: vi.fn(),
      onCloseEvents: vi.fn(),
      onRetryEvents: vi.fn(),
      onPeriodAction
    });

    const action = findByClass(page, 'home-history-button');
    action.props.onClick?.(tapEvent());

    expect(textContent(page)).toContain('历史主题');
    expect(action.props.ariaLabel).toBe('返回今日主题');
    expect(textContent(action)).toBe('');
    expect(onPeriodAction).toHaveBeenCalledOnce();
  });
});

interface TestElementProps {
  className?: string;
  children?: ReactNode;
  catchMove?: boolean;
  ariaLabel?: string;
  onClick?: (event: ReturnType<typeof tapEvent>) => void;
}

type TestElement = ReactElement<TestElementProps>;

function findByClass(root: ReactNode, className: string): TestElement {
  const match = findAllByClass(root, className)[0];
  if (!match) throw new Error(`missing element .${className}`);
  return match;
}

function findAllByClass(root: ReactNode, className: string): TestElement[] {
  return flattenElements(root).filter((element) =>
    element.props.className?.split(/\s+/).includes(className)
  );
}

function flattenElements(node: ReactNode): TestElement[] {
  if (!isValidElement<TestElementProps>(node)) return [];
  if (typeof node.type === 'function') {
    const component = node.type as (props: TestElementProps) => ReactNode;
    return flattenElements(component(node.props));
  }
  return [
    node,
    ...Children.toArray(node.props.children).flatMap((child) => flattenElements(child))
  ];
}

function textContent(node: ReactNode): string {
  if (typeof node === 'string' || typeof node === 'number') return String(node);
  if (!isValidElement<TestElementProps>(node)) return '';
  if (typeof node.type === 'function') {
    const component = node.type as (props: TestElementProps) => ReactNode;
    return textContent(component(node.props));
  }
  return Children.toArray(node.props.children).map(textContent).join('');
}

function tapEvent() {
  return { stopPropagation: vi.fn() };
}

async function flushPromises() {
  await Promise.resolve();
  await Promise.resolve();
}
