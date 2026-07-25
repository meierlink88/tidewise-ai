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

describe('ResearchThemeCard navigation', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    vi.mocked(Taro.navigateTo).mockResolvedValue({ errMsg: 'navigateTo:ok' });
    vi.mocked(Taro.showToast).mockResolvedValue({ errMsg: 'showToast:ok' });
  });

  it('binds reasoning-tree navigation only to the detail button', () => {
    const card = ResearchThemeCard({ theme });
    const root = card as TestElement;
    const detailButton = findByClass(card, 'theme-card__detail-button');
    const eventButton = findByClass(card, 'theme-card__event-button');
    const nodeButton = findByClass(card, 'theme-card__node');

    expect(root.props.onClick).toBeUndefined();
    eventButton.props.onClick?.(tapEvent());
    nodeButton.props.onClick?.(tapEvent());
    expect(Taro.navigateTo).not.toHaveBeenCalled();

    const event = tapEvent();
    detailButton.props.onClick?.(event);

    expect(event.stopPropagation).toHaveBeenCalledOnce();
    expect(Taro.navigateTo).toHaveBeenCalledOnce();
    expect(Taro.navigateTo).toHaveBeenCalledWith({
      url: `/pages/research-theme/reasoning-trees/index?theme_id=${theme.id}`
    });
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
});

interface TestElementProps {
  className?: string;
  children?: ReactNode;
  onClick?: (event: ReturnType<typeof tapEvent>) => void;
}

type TestElement = ReactElement<TestElementProps>;

function findByClass(root: ReactNode, className: string): TestElement {
  const match = flattenElements(root).find((element) =>
    element.props.className?.split(/\s+/).includes(className)
  );
  if (!match) throw new Error(`missing element .${className}`);
  return match;
}

function flattenElements(node: ReactNode): TestElement[] {
  if (!isValidElement<TestElementProps>(node)) return [];
  return [
    node,
    ...Children.toArray(node.props.children).flatMap((child) => flattenElements(child))
  ];
}

function tapEvent() {
  return { stopPropagation: vi.fn() };
}
