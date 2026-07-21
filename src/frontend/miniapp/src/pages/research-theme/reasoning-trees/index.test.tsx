import Taro from '@tarojs/taro';
import { Children, isValidElement, type ReactElement, type ReactNode } from 'react';
import { describe, expect, it, vi } from 'vitest';
import listFixture from '../../../../../../testdata/reasoning-tree-v1/01-reasoning-tree-list-result.json';
import type { ResearchReasoningTreePort } from '../../../features/research-reasoning-trees/contract';
import { ResearchReasoningTreeSession } from '../../../features/research-reasoning-trees/session';
import { parseResearchReasoningTreeIndex } from '../../../features/research-reasoning-trees/wire-contract';
import { IndexContent } from './index';

vi.mock('@tarojs/taro', () => ({
  default: {
    nextTick: vi.fn((callback: () => void) => callback()),
    pageScrollTo: vi.fn().mockResolvedValue({ errMsg: 'pageScrollTo:ok' })
  },
  useRouter: vi.fn()
}));

vi.mock('@tarojs/components', () => ({
  Button: 'button',
  ScrollView: 'scroll-view',
  Text: 'text',
  View: 'view'
}));

vi.mock('../../../features/research-reasoning-trees/port', () => ({
  createResearchReasoningTreePort: vi.fn(() => ({
    list: vi.fn(),
    get: vi.fn()
  }))
}));

describe('reasoning tree page interactions', () => {
  it('loads a newly selected Anchor and returns the page to the tree content top', () => {
    const index = parseResearchReasoningTreeIndex(listFixture.result);
    const session = new ResearchReasoningTreeSession(index.theme.id, inertPort);
    const selectAnchor = vi.spyOn(session, 'selectAnchor');
    const selectedAnchorId = index.reasoningTrees[0].anchorId;
    const nextAnchorId = index.reasoningTrees[1].anchorId;
    const content = IndexContent({
      state: {
        routeStatus: 'valid',
        index: { status: 'ready', value: index },
        selectedAnchorId,
        detailsByAnchorId: {}
      },
      session
    });

    const tabs = findAllByClass(content, 'reasoning-tab');
    tabs[1].props.onClick?.();

    expect(selectAnchor).toHaveBeenCalledWith(nextAnchorId);
    expect(Taro.nextTick).toHaveBeenCalledOnce();
    expect(Taro.pageScrollTo).toHaveBeenCalledWith({
      selector: '#reasoning-tabs-wrap',
      duration: 180
    });
  });
});

const inertPort: ResearchReasoningTreePort = {
  async list() {
    throw new Error('not used');
  },
  async get() {
    throw new Error('not used');
  }
};

interface TestElementProps {
  className?: string;
  children?: ReactNode;
  onClick?: () => void;
}

type TestElement = ReactElement<TestElementProps>;

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
