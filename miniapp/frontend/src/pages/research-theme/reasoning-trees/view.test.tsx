import { renderToStaticMarkup } from 'react-dom/server';
import { Children, isValidElement, type ReactElement, type ReactNode } from 'react';
import { describe, expect, it, vi } from 'vitest';
import listFixture from '../../../../../../testdata/reasoning-tree-v1/01-reasoning-tree-list-result.json';
import detailFixture from '../../../../../../testdata/reasoning-tree-v1/02-reasoning-tree-with-contradiction-result.json';
import {
  parseResearchReasoningTreeDetail,
  parseResearchReasoningTreeIndex
} from '../../../features/research-reasoning-trees/wire-contract';
import { ReasoningThemeHero, ReasoningTreeTransmission, ReasoningTreeView } from './view';

vi.mock('@tarojs/components', () => ({
  ScrollView: 'scroll-view',
  Text: 'text',
  View: 'view'
}));

const themeId = listFixture.result.theme.id;
const treeId = detailFixture.result.reasoning_tree.reasoning_tree_id;

describe('ReasoningTreeView', () => {
  it('renders the Theme-level judgment separately from each tree', () => {
    const theme = parseResearchReasoningTreeIndex(listFixture.result).theme;
    const markup = renderToStaticMarkup(<ReasoningThemeHero theme={theme} />);

    expect(markup).toContain('中等影响');
    expect(markup).toContain(theme.title);
    expect(markup).toContain(theme.oneLineConclusion);
    expect(markup).toContain(theme.transmissionSummary);
    expect(markup).toContain('主题传导摘要');
    expect(markup).not.toContain('Theme 传导摘要');
    expect(markup).toContain('07-28 08:05 发布');
  });

  it('renders the compact node path and defaults the selected detail to the result node', () => {
    const detail = parseResearchReasoningTreeDetail(detailFixture.result, themeId, treeId);
    const markup = renderToStaticMarkup(<ReasoningTreeView detail={detail} />);
    const nodes = detail.reasoningTree.nodes;
    const selected = nodes.at(-1)!;

    expect(countClass(markup, 'reasoning-event')).toBe(2);
    expect(markup).toContain(detail.reasoningTree.title);
    expect(markup).toContain(`${detail.reasoningTree.eventCount} 条政经事件`);
    expect(markup).toContain(detail.reasoningTree.supportSummary);
    expect(markup).toContain(detail.reasoningTree.counterSummary);
    expect(markup).not.toContain(detail.reasoningTree.transmissionSummary);
    expect(markup).toContain(detail.reasoningTree.impactSummary);
    expect(markup).toContain(detail.reasoningTree.conclusionBoundarySummary);
    expect(countClass(markup, 'reasoning-chain-node')).toBe(3);
    expect(countClass(markup, 'reasoning-chain-edge')).toBe(2);
    expect(classTextContents(markup, 'reasoning-chain-node__index')).toEqual([
      '节点 01',
      '节点 02',
      '节点 03 · 结果'
    ]);
    expect(countClass(markup, 'reasoning-chain-node__signal-slot')).toBe(3);
    expect(countClass(markup, 'reasoning-chain-node__direction--increase')).toBe(3);
    expect(countClass(markup, 'reasoning-chain-node__strength')).toBe(3);
    expect(classTextContents(markup, 'reasoning-chain-node__strength')).toEqual([
      '中等影响',
      '中等影响',
      '中等影响'
    ]);
    expect(countClass(markup, 'reasoning-chain-node__gap')).toBe(0);
    expect(markup).not.toContain(nodes[0].evidenceGapSummary);
    expect(markup).not.toContain(selected.reasoningBasisSummary);
    expect(markup).toContain('产业链节点传导');
    expect(markup).toContain('reasoning-chain-node--selected');
    expect(markup).toContain('节点 03 · 结果节点');
    expect(markup).toContain(selected.name);
    expect(markup).toContain(selected.primarySignal.displaySummary);
    expect(markup).toContain(selected.incomingTransmissionTitle);
    expect(markup).toContain(selected.incomingTransmissionMechanism);
    expect(markup).toContain(selected.incomingConditionSummary);
    expect(markup).toContain('节点 02 → 节点 03');
    expect(markup).not.toContain(nodes[1].incomingTransmissionTitle);
    expect(markup).not.toContain('影响状态');
    expect(markup).not.toContain('变量状态');
    expect(markup).not.toContain('变量信号');
    expect(markup).not.toContain('推导依据');
    expect(markup).not.toContain('数据缺口');
    expect(markup).not.toContain('主题影响');
    expect(markup).toContain('判断边界');
    expect(markup).not.toContain('结论边界与失效条件');
    expect(markup).toContain('后续验证');
    expect(markup).not.toContain('下一检查点');
    expect(markup).not.toContain(detail.reasoningTree.invalidationConditions[0]);
    expect(markup).not.toContain(detail.reasoningTree.invalidationConditions[1]);
    for (const checkpoint of detail.reasoningTree.checkpoints) {
      expect(markup).toContain(checkpoint.summary);
    }
  });

  it('switches the selected detail and incoming transmission when a node card is clicked', () => {
    const detail = parseResearchReasoningTreeDetail(detailFixture.result, themeId, treeId);
    const nodes = detail.reasoningTree.nodes;
    let selectedNodeId = nodes.at(-1)!.id;
    const transmission = ReasoningTreeTransmission({
      nodes,
      selectedNodeId,
      onSelect: (id) => {
        selectedNodeId = id;
      }
    });
    const nodeCards = findAllByClass(transmission, 'reasoning-chain-node');

    nodeCards[0].props.onClick?.();
    expect(selectedNodeId).toBe(nodes[0].id);
    const entryMarkup = renderToStaticMarkup(
      <ReasoningTreeTransmission
        nodes={nodes}
        selectedNodeId={selectedNodeId}
        onSelect={() => undefined}
      />
    );
    expect(entryMarkup).toContain('节点 01 · 信号入口');
    expect(entryMarkup).toContain(nodes[0].primarySignal.displaySummary);
    expect(countClass(entryMarkup, 'reasoning-node-detail__mechanism')).toBe(0);

    nodeCards[1].props.onClick?.();
    expect(selectedNodeId).toBe(nodes[1].id);
    const downstreamMarkup = renderToStaticMarkup(
      <ReasoningTreeTransmission
        nodes={nodes}
        selectedNodeId={selectedNodeId}
        onSelect={() => undefined}
      />
    );
    expect(downstreamMarkup).toContain('节点 02 · 路径节点');
    expect(downstreamMarkup).toContain(nodes[1].incomingTransmissionTitle);
    expect(downstreamMarkup).toContain(nodes[1].incomingTransmissionMechanism);
    expect(downstreamMarkup).toContain(nodes[1].incomingConditionSummary);
    expect(downstreamMarkup).toContain('节点 01 → 节点 02');
    expect(downstreamMarkup).not.toContain(nodes[2].incomingTransmissionTitle);
    expect(downstreamMarkup).not.toContain('component_of');
    expect(downstreamMarkup).not.toContain('approved');
  });

  it('retains neutral boundary and follow-up verification sections when content is empty', () => {
    const detail = parseResearchReasoningTreeDetail(detailFixture.result, themeId, treeId);
    const empty = {
      ...detail,
      reasoningTree: {
        ...detail.reasoningTree,
        conclusionBoundarySummary: null,
        counterSummary: '',
        invalidationConditions: [],
        checkpoints: []
      }
    };

    const markup = renderToStaticMarkup(<ReasoningTreeView detail={empty} />);

    expect(markup).toContain('当前暂无明确反证');
    expect(markup).toContain('判断边界');
    expect(markup).toContain('后续验证');
    expect(markup).toContain('暂无');
  });

  it('shows a one-node Tree as both signal entry and result without a transmission mechanism', () => {
    const detail = parseResearchReasoningTreeDetail(detailFixture.result, themeId, treeId);
    const oneNode = {
      ...detail,
      reasoningTree: {
        ...detail.reasoningTree,
        nodes: detail.reasoningTree.nodes.slice(0, 1)
      }
    };

    const markup = renderToStaticMarkup(<ReasoningTreeView detail={oneNode} />);

    expect(classTextContents(markup, 'reasoning-chain-node__index')).toEqual(['节点 01 · 结果']);
    expect(markup).toContain('信号入口 · 结果节点');
    expect(countClass(markup, 'reasoning-node-detail__mechanism')).toBe(0);
    expect(markup).not.toContain('传导机制');
  });

  it('omits event time when the BFF publishes null', () => {
    const detail = parseResearchReasoningTreeDetail(detailFixture.result, themeId, treeId);
    const withoutTimes = {
      ...detail,
      reasoningTree: {
        ...detail.reasoningTree,
        events: detail.reasoningTree.events.map((event) => ({ ...event, eventTime: null }))
      }
    };
    const markup = renderToStaticMarkup(<ReasoningTreeView detail={withoutTimes} />);

    expect(countClass(markup, 'reasoning-event__time')).toBe(0);
    expect(markup).not.toContain('时间未知');
  });
});

function countClass(markup: string, className: string): number {
  return [...markup.matchAll(/class="([^"]*)"/g)].filter((match) =>
    match[1].split(/\s+/).includes(className)
  ).length;
}

function classTextContents(markup: string, className: string): string[] {
  const pattern = new RegExp(`<text class="${className}">([\\s\\S]*?)</text>`, 'g');
  return [...markup.matchAll(pattern)].map((match) =>
    match[1].replace(/<!--.*?-->/g, '').replace(/<[^>]+>/g, '')
  );
}

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
