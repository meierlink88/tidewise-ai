import Taro from '@tarojs/taro';
import { Children, isValidElement, type ReactElement, type ReactNode } from 'react';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { mockResearchThemeFeed } from '../../../mocks/research-themes/mock-port';
import { ResearchThemeCard } from './research-theme-card';

vi.mock('@tarojs/taro', () => ({
  default: {
    navigateTo: vi.fn(),
    showToast: vi.fn()
  }
}));

vi.mock('@tarojs/components', () => ({
  Button: 'button',
  Image: 'image',
  Text: 'text',
  View: 'view'
}));

const theme = mockResearchThemeFeed.items[0];

describe('ResearchThemeCard', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    vi.mocked(Taro.navigateTo).mockResolvedValue({ errMsg: 'navigateTo:ok' });
    vi.mocked(Taro.showToast).mockResolvedValue({ errMsg: 'showToast:ok' });
  });

  it('renders the approved investment-outlook content and binds only the detail button', () => {
    const card = ResearchThemeCard({ theme });
    const root = card as TestElement;
    const detailButton = findByClass(card, 'theme-card__detail-button');
    const industryLabel = findByClass(card, 'theme-card__industry-label');
    const nodes = findAllByClass(card, 'theme-card__node');

    expect(root.props.onClick).toBeUndefined();
    expect(industryLabel.props.children).toBe('个关注节点');
    expect(nodes).toHaveLength(3);
    expect(textContent(nodes[0])).toContain('交换机');
    expect(textContent(nodes[0])).toContain('机会');
    expect(textContent(nodes[0])).toContain('端口计划：增加 80%');
    expect(textContent(card)).toContain(theme.oneLineConclusion);
    expect(textContent(card)).toContain(theme.transmissionSummary);
    expect(textContent(card)).toContain(theme.investmentGuidanceSummary);
    expect(textContent(card)).toContain('2 条政经事件');
    expect(textContent(card)).toContain('2 条产业链路径');
    expect(textContent(detailButton)).toBe('推导详情');
    expect(flattenElements(card).filter((element) => element.props.onClick)).toHaveLength(1);

    const event = tapEvent();
    detailButton.props.onClick?.(event);

    expect(event.stopPropagation).toHaveBeenCalledOnce();
    expect(Taro.navigateTo).toHaveBeenCalledOnce();
    expect(Taro.navigateTo).toHaveBeenCalledWith({
      url: `/pages/research-theme/reasoning-trees/index?theme_id=${theme.id}`
    });
  });

  it('shows all five nodes in display order and maps all outlook directions', () => {
    const impacts = [
      ...theme.impacts.slice(0, 2),
      {
        ...theme.impacts[0],
        chainNodeEntityId: '44444444-4444-4444-8444-444444444444',
        name: '存储芯片封测',
        impactDirection: 'uncertain' as const,
        primarySignalDisplaySummary: '订单需求：尚待确认',
        displayOrder: 3
      },
      {
        ...theme.impacts[0],
        chainNodeEntityId: '55555555-5555-4555-8555-555555555555',
        name: '服务器存储采购',
        impactDirection: 'negative' as const,
        primarySignalDisplaySummary: '采购成本：可能上升',
        displayOrder: 4
      },
      {
        ...theme.impacts[0],
        chainNodeEntityId: '66666666-6666-4666-8666-666666666666',
        name: '手机存储采购',
        impactDirection: 'mixed' as const,
        primarySignalDisplaySummary: '采购价格：方向分化',
        displayOrder: 5
      }
    ];

    const card = ResearchThemeCard({ theme: { ...theme, impacts } });
    const nodes = findAllByClass(card, 'theme-card__node');

    expect(nodes).toHaveLength(5);
    expect(nodes.map(textContent)).toEqual([
      '交换机机会端口计划：增加 80%',
      '高速光模块机会模块需求：可能增加',
      '存储芯片封测不确定订单需求：尚待确认',
      '服务器存储采购风险采购成本：可能上升',
      '手机存储采购不确定采购价格：方向分化'
    ]);
    expect(findAllByClass(card, 'theme-card__outlook--opportunity')).toHaveLength(2);
    expect(findAllByClass(card, 'theme-card__outlook--risk')).toHaveLength(1);
    expect(findAllByClass(card, 'theme-card__outlook--uncertain')).toHaveLength(2);
  });

  it('does not invent a variable status for an immutable legacy impact', () => {
    const card = ResearchThemeCard({
      theme: {
        ...theme,
        impacts: [{ ...theme.impacts[0], primarySignalDisplaySummary: null }]
      }
    });

    expect(textContent(findByClass(card, 'theme-card__node'))).toBe('交换机机会');
    expect(textContent(card)).not.toContain('变量状态');
    expect(textContent(card)).not.toContain('暂无');
  });

  it('shows a stable message when Taro rejects navigation', async () => {
    vi.mocked(Taro.navigateTo).mockRejectedValueOnce(new Error('hidden platform error'));
    const detailButton = findByClass(ResearchThemeCard({ theme }), 'theme-card__detail-button');

    detailButton.props.onClick?.(tapEvent());

    await vi.waitFor(() => {
      expect(Taro.showToast).toHaveBeenCalledWith({
        title: '影响路径暂时无法打开',
        icon: 'none',
        duration: 1600
      });
    });
  });

  it('opens the page when no Reason Tree receipt exists so the page can show its empty state', () => {
    const detailButton = findByClass(
      ResearchThemeCard({ theme: { ...theme, reasoningTreeCount: 0 } }),
      'theme-card__detail-button'
    );

    detailButton.props.onClick?.(tapEvent());

    expect(Taro.navigateTo).toHaveBeenCalledOnce();
    expect(Taro.showToast).not.toHaveBeenCalled();
  });
});

interface TestElementProps {
  className?: string;
  children?: ReactNode;
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
  return [
    node,
    ...Children.toArray(node.props.children).flatMap((child) => flattenElements(child))
  ];
}

function textContent(node: ReactNode): string {
  if (typeof node === 'string' || typeof node === 'number') return String(node);
  if (!isValidElement<TestElementProps>(node)) return '';
  return Children.toArray(node.props.children).map(textContent).join('');
}

function tapEvent() {
  return { stopPropagation: vi.fn() };
}
