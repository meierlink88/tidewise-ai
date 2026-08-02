import { Children, isValidElement, type ReactElement, type ReactNode } from 'react';
import { describe, expect, it, vi } from 'vitest';
import {
  mockResearchThemeDetail,
  mockResearchThemeFeed
} from '../../../mocks/research-themes/mock-port';
import { ResearchThemeEventsSheet } from './research-theme-events-sheet';

vi.mock('@tarojs/components', () => ({
  Button: 'button',
  ScrollView: 'scroll-view',
  Text: 'text',
  View: 'view'
}));

describe('ResearchThemeEventsSheet', () => {
  it('renders the approved A timeline with only time, title, and summary', () => {
    const sheet = ResearchThemeEventsSheet({
      theme: mockResearchThemeFeed.items[0],
      detailState: { status: 'ready', value: mockResearchThemeDetail },
      onClose: vi.fn(),
      onRetry: vi.fn()
    });

    expect(findByClass(sheet, 'theme-events-overlay').props.catchMove).toBe(true);
    expect(findAllByClass(sheet, 'theme-events-item')).toHaveLength(2);
    expect(textContent(sheet)).toContain('高速光模块需求验证');
    expect(textContent(sheet)).toContain('端口计划上调');
    expect(textContent(sheet)).toContain('云厂商端口计划上调 80%。');
    expect(textContent(sheet)).toContain('时间待确认');
    expect(textContent(sheet)).not.toContain('关联判断');
    expect(textContent(sheet)).not.toContain('来源');
    expect(textContent(sheet)).not.toContain('事件详情');
  });

  it('keeps unavailable and retry behavior inside the sheet', () => {
    const onClose = vi.fn();
    const onRetry = vi.fn();
    const sheet = ResearchThemeEventsSheet({
      theme: mockResearchThemeFeed.items[0],
      detailState: { status: 'error', errorKind: 'themeUnavailable' },
      onClose,
      onRetry
    });

    expect(textContent(sheet)).toContain('该主题事件暂不可用');
    findByClass(sheet, 'theme-events-state__retry').props.onClick?.(tapEvent());
    expect(onRetry).toHaveBeenCalledOnce();

    findByClass(sheet, 'theme-events-overlay').props.onClick?.(tapEvent());
    expect(onClose).toHaveBeenCalledOnce();
    const sheetTap = tapEvent();
    findByClass(sheet, 'theme-events-sheet').props.onClick?.(sheetTap);
    expect(sheetTap.stopPropagation).toHaveBeenCalledOnce();
    expect(onClose).toHaveBeenCalledOnce();
  });
});

interface TestElementProps {
  className?: string;
  children?: ReactNode;
  catchMove?: boolean;
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
