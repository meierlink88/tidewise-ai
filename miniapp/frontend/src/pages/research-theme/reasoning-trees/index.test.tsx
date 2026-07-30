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
  it('loads a newly selected Reason Tree and returns the page to the tree content top', () => {
    const index = parseResearchReasoningTreeIndex(listFixture.result);
    const session = new ResearchReasoningTreeSession(index.theme.id, inertPort);
    const selectReasoningTree = vi.spyOn(session, 'selectReasoningTree');
    const selectedReasoningTreeId = index.reasoningTrees[0].reasoningTreeId;
    const nextReasoningTreeId = index.reasoningTrees[1].reasoningTreeId;
    const content = IndexContent({
      state: {
        routeStatus: 'valid',
        index: { status: 'ready', value: index },
        selectedReasoningTreeId,
        detailsByReasoningTreeId: {}
      },
      session
    });

    const tabs = findAllByClass(content, 'reasoning-tab');
    expect(tabs.map((tab) => tab.props.children)).toEqual(['高速光模块产业链', 'DSP 芯片产业链']);
    tabs[1].props.onClick?.();

    expect(selectReasoningTree).toHaveBeenCalledWith(nextReasoningTreeId);
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
