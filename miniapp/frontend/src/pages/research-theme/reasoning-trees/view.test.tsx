import { renderToStaticMarkup } from 'react-dom/server';
import { describe, expect, it, vi } from 'vitest';
import listFixture from '../../../../../../testdata/reasoning-tree-v1/01-reasoning-tree-list-result.json';
import contradictionFixture from '../../../../../../testdata/reasoning-tree-v1/02-reasoning-tree-with-contradiction-result.json';
import unquantifiedFixture from '../../../../../../testdata/reasoning-tree-v1/03-reasoning-tree-without-contradiction-unquantified-result.json';
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

const themeId = '11111111-1111-4111-8111-111111111111';

describe('ReasoningTreeView', () => {
  it('renders the complete Theme judgment above the Anchor tabs', () => {
    const theme = parseResearchReasoningTreeIndex(listFixture.result).theme;
    const markup = renderToStaticMarkup(<ReasoningThemeHero theme={theme} />);

    expect(markup).toContain('高影响');
    expect(markup).toContain(theme.name);
    expect(markup).toContain(theme.oneLineConclusion);
    expect(markup).toContain(theme.transmissionPath);
    expect(markup).toContain('07-20 08:00 发布');
  });

  it('renders all atomic events, branch summaries, and the ordered transmission path', () => {
    const anchorId = contradictionFixture.result.reasoning_tree.anchor_id;
    const tree = parseResearchReasoningTreeDetail(
      contradictionFixture.result,
      themeId,
      anchorId
    ).reasoningTree;
    const markup = renderToStaticMarkup(<ReasoningTreeView tree={tree} />);

    expect(countClass(markup, 'reasoning-event')).toBe(2);
    expect(markup.indexOf('北美云厂商上调AI资本开支')).toBeLessThan(
      markup.indexOf('光模块订单交付节奏仍有分化')
    );
    expect(markup).toContain(tree.supportSummary);
    expect(markup).toContain(tree.counterSummary);
    expect(countClass(markup, 'reasoning-chain-node')).toBe(2);
    expect(countClass(markup, 'reasoning-chain-edge')).toBe(1);
    expect(countClass(markup, 'reasoning-chain-node__mechanism')).toBe(1);
    expect(markup).not.toContain('reasoning-chain-edge__mechanism');
    expect(markup).toContain('AI集群扩容提高节点间高速光互联需求');
    expect(markup).toContain('reasoning-chain-node--anchor');
    expect(markup).toContain('光模块');
  });

  it('keeps the counter branch visible when no counter conclusion was published', () => {
    const anchorId = unquantifiedFixture.result.reasoning_tree.anchor_id;
    const tree = parseResearchReasoningTreeDetail(
      unquantifiedFixture.result,
      themeId,
      anchorId
    ).reasoningTree;
    const markup = renderToStaticMarkup(<ReasoningTreeView tree={tree} />);

    expect(markup).toContain('当前暂无明确反证');
    expect(markup).toContain('待验证');
  });

  it('omits event time cleanly when the BFF did not publish one', () => {
    const anchorId = contradictionFixture.result.reasoning_tree.anchor_id;
    const tree = parseResearchReasoningTreeDetail(
      contradictionFixture.result,
      themeId,
      anchorId
    ).reasoningTree;
    const withoutTimes = {
      ...tree,
      events: tree.events.map((event) => ({ ...event, eventTime: null }))
    };
    const markup = renderToStaticMarkup(<ReasoningTreeView tree={withoutTimes} />);

    expect(countClass(markup, 'reasoning-event__time')).toBe(0);
    expect(markup).not.toContain('时间未知');
  });
});

function countClass(markup: string, className: string): number {
  return [...markup.matchAll(/class="([^"]*)"/g)].filter((match) =>
    match[1].split(/\s+/).includes(className)
  ).length;
}
