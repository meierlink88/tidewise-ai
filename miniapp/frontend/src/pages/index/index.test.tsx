import { Children, isValidElement, type ReactElement, type ReactNode } from 'react';
import { describe, expect, it, vi } from 'vitest';
import { ReportError } from '../../features/reports/contract';
import { mockReportPort } from '../../mocks/reports/mock-port';
import { IndexView, stopHomeRefresh } from './index';

vi.mock('@tarojs/taro', () => ({
  default: {
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

const chrome = { statusBarHeight: 44, navigationBarHeight: 44, rightReservedWidth: 102 };

describe('Report homepage', () => {
  it('keeps the fixed application shell and renders the first paged card batch', async () => {
    const home = await mockReportPort.getHome();
    const onRefresh = vi.fn();
    const page = IndexView({
      chrome,
      query: '',
      onQueryChange: vi.fn(),
      state: { status: 'ready', data: home, refreshing: false, refreshFailed: false },
      onRetry: vi.fn(),
      onRefresh,
      onOpenDetail: vi.fn(),
      onOpenEvidence: vi.fn()
    });
    const copy = textContent(page);

    expect(copy).toContain('观潮家');
    expect(copy).toContain('今日观潮');
    expect(copy).not.toContain('今日推理');
    expect(copy).not.toContain('当前事件如何从地缘政治与宏观经济传导至产业链（动态传导）');
    expect(copy).toContain('发布时间');
    expect(copy).toContain('2026.09.01 12:45');
    expect(copy).toContain('地缘政治');
    expect(copy).toContain('宏观经济');
    expect(copy).toContain('54 条真实产业链 · 首页展示 20 条');
    expect(copy).toContain('人形机器人产业链');
    expect(copy).toContain('AI数据中心液冷服务器产业链');
    expect(copy).toContain('AI算力基础设施服务产业链');
    expect(copy).toContain('AI视频生成服务产业链');
    expect(copy).not.toContain('RPT11111111');
    expect(copy).not.toContain('EVT');
    expect(findAllByClass(page, 'home-search__send')).toHaveLength(1);
    expect(findAllByClass(page, 'home-section-heading__summary')).toHaveLength(0);
    expect(findAllByClass(page, 'home-card-evidence-action')).toHaveLength(22);
    expect(findAllByClass(page, 'home-card-detail-action')).toHaveLength(22);
    expect(findAllByClass(page, 'home-report-section__kind-icon')).toHaveLength(3);
    expect(findAllByClass(page, 'home-report-card__arrow')).toHaveLength(22);
    expect(findAllByClass(page, 'home-company-boundary__icon')).toHaveLength(0);
    expect(findAllByClass(page, 'home-industry-identity')).toHaveLength(20);
    expect(findAllByClass(page, 'home-impact-item')).toHaveLength(47);
    expect(findAllByClass(page, 'home-impact-signal__result-icon')).toHaveLength(47);
    expect(findAllByClass(page, 'home-impact-signal__confidence-icon')).toHaveLength(47);
    expect(findAllByClass(page, 'home-impact-signal__window-icon')).toHaveLength(47);
    expect(findAllByClass(page, 'home-report-card__signals')).toHaveLength(0);
    const scroll = findAllByClass(page, 'home-report-scroll')[0];
    expect(scroll).toBeDefined();
    expect(scroll?.props).toMatchObject({
      scrollY: true,
      enhanced: true,
      refresherEnabled: true
    });
    expect(findAllByClass(scroll?.props.children, 'home-report-group__header')).toHaveLength(0);
    expect(findAllByClass(page, 'home-report-group__header')).toHaveLength(1);
    scroll?.props.onRefresherRefresh?.();
    expect(onRefresh).toHaveBeenCalledOnce();
    expect(
      findAllByClass(page, 'home-report-card').every((card) => card.props.role !== 'button')
    ).toBe(true);
    expect(findAllByClass(page, 'home-report-section__kind-icon').every(hasSvgSource)).toBe(true);
    expect(findAllByClass(page, 'home-report-card__arrow').every(hasSvgSource)).toBe(true);
    expect(findAllByClass(page, 'home-company-boundary__icon').every(hasSvgSource)).toBe(true);
  });

  it('opens card-level Evidence only, without exposing anchor or node Evidence controls', async () => {
    const home = await mockReportPort.getHome();
    const onOpenEvidence = vi.fn();
    const onOpenDetail = vi.fn();
    const page = IndexView({
      chrome,
      query: '',
      onQueryChange: vi.fn(),
      state: { status: 'ready', data: home, refreshing: false, refreshFailed: false },
      onRetry: vi.fn(),
      onRefresh: vi.fn(),
      onOpenDetail,
      onOpenEvidence
    });
    const button = findByAriaLabel(page, '查看地缘政治依据');
    const stopPropagation = vi.fn();

    button.props.onClick?.({ stopPropagation });

    expect(stopPropagation).toHaveBeenCalledOnce();
    expect(onOpenEvidence).toHaveBeenCalledWith({
      reportId: 'RPT11111111-1111-4111-8111-111111111111',
      scopeToken: home.reports[0]?.cards[0]?.evidenceScopeToken,
      title: '地缘政治证据'
    });
    expect(onOpenDetail).not.toHaveBeenCalled();
    expect(() => findByAriaLabel(page, '查看伊朗—美以及海湾安全对抗证据')).toThrow();
  });

  it('opens detail only from the dedicated transmission action', async () => {
    const home = await mockReportPort.getHome();
    const onOpenDetail = vi.fn();
    const page = IndexView({
      chrome,
      query: '',
      onQueryChange: vi.fn(),
      state: { status: 'ready', data: home, refreshing: false, refreshFailed: false },
      onRetry: vi.fn(),
      onRefresh: vi.fn(),
      onOpenDetail,
      onOpenEvidence: vi.fn()
    });

    findByAriaLabel(page, '查看地缘政治传导详情').props.onClick?.({
      stopPropagation: vi.fn()
    });

    expect(onOpenDetail).toHaveBeenCalledWith({
      reportId: 'RPT11111111-1111-4111-8111-111111111111',
      targetType: 'layer',
      targetKey: 'geopolitics'
    });
  });

  it('labels the historical fallback without changing the actual publication time', async () => {
    const home = await mockReportPort.getHome();
    const page = IndexView({
      chrome,
      query: '',
      onQueryChange: vi.fn(),
      state: {
        status: 'ready',
        data: { ...home, selection: { ...home.selection, mode: 'latest_fallback' } },
        refreshing: false,
        refreshFailed: false
      },
      onRetry: vi.fn(),
      onRefresh: vi.fn(),
      onOpenDetail: vi.fn(),
      onOpenEvidence: vi.fn()
    });
    expect(textContent(page)).toContain('最近发布');
    expect(textContent(page)).toContain('2026.09.01 12:45');
  });

  it('appends a bounded page, deduplicates cards, and exposes scroll and retry loading controls', async () => {
    const home = await mockReportPort.getHome();
    const group = home.reports[0]!;
    const cursor = group.nextCursor!;
    const nextPage = await mockReportPort.getIndustryChains(group.report.id, cursor, 20);
    const duplicate = group.cards.find((card) => card.kind === 'industry_chain')!;
    const onLoadMoreChains = vi.fn();
    const page = IndexView({
      chrome,
      query: '',
      onQueryChange: vi.fn(),
      state: { status: 'ready', data: home, refreshing: false, refreshFailed: false },
      onRetry: vi.fn(),
      onRefresh: vi.fn(),
      onOpenDetail: vi.fn(),
      onOpenEvidence: vi.fn(),
      chainPages: {
        [group.report.id]: {
          items: [duplicate, ...nextPage.items],
          nextCursor: nextPage.nextCursor,
          loading: false,
          failed: true
        }
      },
      onLoadMoreChains
    });

    expect(findAllByClass(page, 'home-industry-identity')).toHaveLength(40);
    expect(textContent(page)).toContain('首页展示 40 条');
    const scroll = findAllByClass(page, 'home-report-scroll')[0];
    scroll?.props.onScrollToLower?.();
    findAllByClass(page, 'home-chain-page-state--retry')[0]?.props.onClick?.({
      stopPropagation: vi.fn()
    });
    expect(onLoadMoreChains).toHaveBeenCalledTimes(2);
    expect(onLoadMoreChains).toHaveBeenLastCalledWith(group.report.id, nextPage.nextCursor);
  });

  it('renders explicit loading, empty and retryable error states', () => {
    const base = {
      chrome,
      query: '',
      onQueryChange: vi.fn(),
      onRetry: vi.fn(),
      onRefresh: vi.fn(),
      onOpenDetail: vi.fn(),
      onOpenEvidence: vi.fn()
    };
    expect(textContent(IndexView({ ...base, state: { status: 'loading' } }))).toContain(
      '正在读取报告'
    );
    expect(
      textContent(
        IndexView({
          ...base,
          state: { status: 'empty', refreshing: false, refreshFailed: false }
        })
      )
    ).toContain('暂无推理报告');
    expect(
      textContent(
        IndexView({
          ...base,
          state: { status: 'error', error: new ReportError('serviceUnavailable') }
        })
      )
    ).toContain('重新加载');
  });

  it('always stops the native pull-down refresh', async () => {
    const api = { stopPullDownRefresh: vi.fn(), showToast: vi.fn() };
    await stopHomeRefresh(api);
    expect(api.stopPullDownRefresh).toHaveBeenCalledOnce();
  });
});

interface TestElementProps {
  className?: string;
  ariaLabel?: string;
  role?: string;
  onClick?: (event: { stopPropagation: () => void }) => void;
  onRefresherRefresh?: () => void;
  onScrollToLower?: () => void;
  src?: string;
  scrollY?: boolean;
  enhanced?: boolean;
  usingSticky?: boolean;
  refresherEnabled?: boolean;
  children?: ReactNode;
}

type TestElement = ReactElement<TestElementProps>;

function findAllByClass(root: ReactNode, className: string): TestElement[] {
  return flattenElements(root).filter((element) =>
    element.props.className?.split(/\s+/).includes(className)
  );
}

function findByAriaLabel(root: ReactNode, ariaLabel: string): TestElement {
  const element = flattenElements(root).find((item) => item.props.ariaLabel === ariaLabel);
  if (!element) throw new Error(`missing element: ${ariaLabel}`);
  return element;
}

function flattenElements(node: ReactNode): TestElement[] {
  if (!isValidElement<TestElementProps>(node)) return [];
  if (typeof node.type === 'function') {
    const component = node.type as (props: TestElementProps) => ReactNode;
    return flattenElements(component(node.props));
  }
  return [node, ...Children.toArray(node.props.children).flatMap(flattenElements)];
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

function hasSvgSource(element: TestElement): boolean {
  return typeof element.props.src === 'string' && element.props.src.endsWith('.svg');
}
