import { Children, isValidElement, type ReactElement, type ReactNode } from 'react';
import { describe, expect, it, vi } from 'vitest';
import { mockResearchThemeFeed } from '../../mocks/research-themes/mock-port';
import type { ResearchThemeFeedPort } from '../../features/research-themes/contract';
import {
  ResearchThemeHomeSession,
  type ResearchThemeHomeSessionState
} from '../../features/research-themes/session';
import { IndexView, refreshHomeFeed } from './index';

vi.mock('@tarojs/taro', () => ({
  default: {
    getSystemInfoSync: vi.fn(() => ({ statusBarHeight: 44, screenWidth: 390 })),
    getMenuButtonBoundingClientRect: vi.fn(() => ({ top: 50, bottom: 82, height: 32 })),
    navigateTo: vi.fn(),
    showToast: vi.fn(),
    stopPullDownRefresh: vi.fn()
  },
  usePullDownRefresh: vi.fn()
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
  it('renders the Theme feed without the removed static category and tracking bar', () => {
    const state: ResearchThemeHomeSessionState = {
      feed: { status: 'ready', value: mockResearchThemeFeed },
      selectedThemeId: null,
      detailsByThemeId: {}
    };

    const page = IndexView({
      state,
      query: '',
      chrome: { statusBarHeight: 44, navigationBarHeight: 44, rightReservedWidth: 16 },
      onQueryChange: vi.fn(),
      onOpenEvents: vi.fn(),
      onCloseEvents: vi.fn(),
      onRetryEvents: vi.fn()
    });

    expect(findAllByClass(page, 'category-bar')).toEqual([]);
    expect(textContent(page)).not.toContain('跟踪中');
    expect(textContent(page)).toContain('今日推理主线');
    expect(textContent(page)).toContain(mockResearchThemeFeed.items[0].oneLineConclusion);
  });

  it('preserves the last feed and always stops native refresh when refresh fails', async () => {
    const port: ResearchThemeFeedPort = {
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
});

interface TestElementProps {
  className?: string;
  children?: ReactNode;
}

type TestElement = ReactElement<TestElementProps>;

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
