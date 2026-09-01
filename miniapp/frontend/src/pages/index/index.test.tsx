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
  Text: 'text',
  View: 'view'
}));

const chrome = { statusBarHeight: 44, navigationBarHeight: 44, rightReservedWidth: 102 };

describe('Report homepage', () => {
  it('keeps the application shell and renders every persisted card in one Report group', async () => {
    const home = await mockReportPort.getHome();
    const page = IndexView({
      chrome,
      query: '',
      onQueryChange: vi.fn(),
      state: { status: 'ready', data: home, refreshing: false, refreshFailed: false },
      onRetry: vi.fn(),
      onOpenDetail: vi.fn(),
      onOpenEvidence: vi.fn()
    });
    const copy = textContent(page);

    expect(copy).toContain('观潮');
    expect(copy).toContain('今日推理');
    expect(copy).toContain('当前事件如何从地缘政治与宏观经济传导至产业链');
    expect(copy).toContain('地缘政治');
    expect(copy).toContain('宏观经济');
    expect(copy).toContain('人形机器人产业链');
    expect(copy).toContain('AI数据中心液冷服务器产业链');
    expect(copy).toContain('AI算力基础设施服务产业链');
    expect(copy).toContain('油品石化贸易服务产业链');
    expect(copy).not.toContain('RPT11111111');
    expect(copy).not.toContain('EVT');
    expect(findAllByClass(page, 'report-evidence-button')).toHaveLength(22);
    expect(findAllByClass(page, 'home-report-card__kind-icon')).toHaveLength(6);
    expect(findAllByClass(page, 'home-report-card__arrow')).toHaveLength(6);
    expect(findAllByClass(page, 'home-company-boundary__icon')).toHaveLength(1);
    expect(findAllByClass(page, 'home-report-card__kind-icon').every(hasSvgSource)).toBe(true);
    expect(findAllByClass(page, 'home-report-card__arrow').every(hasSvgSource)).toBe(true);
    expect(findAllByClass(page, 'home-company-boundary__icon').every(hasSvgSource)).toBe(true);
  });

  it('opens direct-impact Evidence with the object scope without opening the card', async () => {
    const home = await mockReportPort.getHome();
    const onOpenEvidence = vi.fn();
    const page = IndexView({
      chrome,
      query: '',
      onQueryChange: vi.fn(),
      state: { status: 'ready', data: home, refreshing: false, refreshFailed: false },
      onRetry: vi.fn(),
      onOpenDetail: vi.fn(),
      onOpenEvidence
    });
    const button = findByAriaLabel(page, '查看伊朗—美以及海湾安全对抗证据');
    const stopPropagation = vi.fn();

    button.props.onClick?.({ stopPropagation });

    expect(stopPropagation).toHaveBeenCalledOnce();
    expect(onOpenEvidence).toHaveBeenCalledWith({
      reportId: 'RPT11111111-1111-4111-8111-111111111111',
      scopeType: 'anchor',
      scopeKey: 'geo-a01',
      title: '伊朗—美以及海湾安全对抗证据'
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
      onOpenDetail: vi.fn(),
      onOpenEvidence: vi.fn()
    });
    expect(textContent(page)).toContain('今日暂无 · 展示最近发布');
    expect(textContent(page)).toContain('最近发布');
    expect(textContent(page)).toContain('2026.09.01 12:45');
  });

  it('renders explicit loading, empty and retryable error states', () => {
    const base = {
      chrome,
      query: '',
      onQueryChange: vi.fn(),
      onRetry: vi.fn(),
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
  onClick?: (event: { stopPropagation: () => void }) => void;
  src?: string;
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
