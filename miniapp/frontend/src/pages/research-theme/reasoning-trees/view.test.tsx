import { renderToStaticMarkup } from 'react-dom/server';
import { describe, expect, it, vi } from 'vitest';
import listFixture from '../../../../../../testdata/reasoning-tree-v1/01-reasoning-tree-list-result.json';
import detailFixture from '../../../../../../testdata/reasoning-tree-v1/02-reasoning-tree-with-contradiction-result.json';
import {
  parseResearchReasoningTreeDetail,
  parseResearchReasoningTreeIndex
} from '../../../features/research-reasoning-trees/wire-contract';
import { ReasoningThemeHero, ReasoningTreeView } from './view';

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
    expect(markup).toContain('07-28 08:05 发布');
  });

  it('renders selectable industry-chain nodes and the selected-node detail contract', () => {
    const detail = parseResearchReasoningTreeDetail(detailFixture.result, themeId, treeId);
    const markup = renderToStaticMarkup(<ReasoningTreeView detail={detail} />);

    expect(countClass(markup, 'reasoning-event')).toBe(2);
    expect(markup).toContain(detail.reasoningTree.title);
    expect(markup).toContain(`${detail.reasoningTree.eventCount} 条 Event`);
    expect(markup).toContain(detail.reasoningTree.supportSummary);
    expect(markup).toContain(detail.reasoningTree.counterSummary);
    expect(markup).toContain(detail.reasoningTree.transmissionSummary);
    expect(markup).toContain(detail.reasoningTree.impactSummary);
    expect(markup).toContain(detail.reasoningTree.conclusionBoundarySummary);
    expect(countClass(markup, 'reasoning-chain-node')).toBe(3);
    expect(countClass(markup, 'reasoning-chain-edge')).toBe(2);
    expect(classTextContents(markup, 'reasoning-chain-node__index')).toEqual([
      '节点 01',
      '节点 02',
      '节点 03'
    ]);
    expect(countClass(markup, 'reasoning-chain-node__signal-slot')).toBe(3);
    expect(countClass(markup, 'reasoning-chain-node__direction--increase')).toBe(3);
    expect(countClass(markup, 'reasoning-chain-node__gap')).toBe(3);
    for (const node of detail.reasoningTree.nodes) {
      expect(markup).toContain(node.evidenceGapSummary);
    }
    expect(markup).toContain('产业链节点传导');
    expect(markup).toContain('reasoning-chain-node--selected');
    expect(markup).toContain('Theme Impact');
    expect(markup).toContain('变量信号');
    expect(markup).toContain('数据缺口');
    expect(markup).toContain('结论边界与失效条件');
    expect(markup).toContain('下一检查点');
  });

  it('labels a non-first node without a formal graph edge as analyst inference', () => {
    const detail = parseResearchReasoningTreeDetail(detailFixture.result, themeId, treeId);
    const inferred = {
      ...detail,
      reasoningTree: {
        ...detail.reasoningTree,
        nodes: detail.reasoningTree.nodes
          .slice(0, 2)
          .map((node, index) => (index === 1 ? { ...node, incomingGraphEdge: null } : node))
      }
    };

    const markup = renderToStaticMarkup(<ReasoningTreeView detail={inferred} />);

    expect(markup).toContain('分析推断');
  });

  it('retains neutral boundary and checkpoint sections when arrays are empty', () => {
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
    expect(markup).toContain('结论边界与失效条件');
    expect(markup).toContain('下一检查点');
    expect(markup).toContain('暂无');
  });

  it('shows a one-node Tree as both signal entry and result', () => {
    const detail = parseResearchReasoningTreeDetail(detailFixture.result, themeId, treeId);
    const oneNode = {
      ...detail,
      reasoningTree: {
        ...detail.reasoningTree,
        nodes: detail.reasoningTree.nodes.slice(0, 1)
      }
    };

    const markup = renderToStaticMarkup(<ReasoningTreeView detail={oneNode} />);

    expect(classTextContents(markup, 'reasoning-chain-node__index')).toEqual(['节点 01']);
    expect(markup).toContain('信号入口 · 结果节点');
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
