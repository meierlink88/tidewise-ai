// @vitest-environment jsdom
import { act, createElement, type ReactNode } from 'react';
import { createRoot, type Root } from 'react-dom/client';
import { beforeEach, afterEach, describe, expect, it, vi } from 'vitest';
import fixture from '../mocks/reports/normalized.json';
import v5 from '../mocks/reports/normalized-v5.json';
import { normalizedMockReportPort } from '../mocks/reports/mock-port';
import { parseAnalysisChain, parseAnalysisDetail } from '../features/reports/normalized-contract';
import { NormalizedHome } from './index/normalized-home';
import { NormalizedDetailView, ChainContent } from './report/detail/normalized-detail';

vi.mock('@tarojs/taro', () => ({ default: { pxTransform: (n: number) => `${n}px` } }));
function element(tag: string) {
  return ({
    children,
    className,
    ariaLabel,
    onClick,
    disabled
  }: {
    children?: ReactNode;
    className?: string;
    ariaLabel?: string;
    onClick?: () => void;
    disabled?: boolean;
  }) => createElement(tag, { className, 'aria-label': ariaLabel, onClick, disabled }, children);
}
vi.mock('@tarojs/components', () => ({
  View: element('div'),
  Text: element('span'),
  Button: element('button'),
  ScrollView: element('section'),
  Image: element('img')
}));
vi.mock('../features/reports/port', () => ({ getReportPort: () => normalizedMockReportPort }));
let root: Root, host: HTMLDivElement;
const reportId = 'RPT11111111-1111-4111-8111-111111111111';
beforeEach(() => {
  (globalThis as { IS_REACT_ACT_ENVIRONMENT?: boolean }).IS_REACT_ACT_ENVIRONMENT = true;
  host = document.createElement('div');
  document.body.append(host);
  root = createRoot(host);
});
afterEach(() => {
  act(() => root.unmount());
  host.remove();
  vi.restoreAllMocks();
  delete (globalThis as { IS_REACT_ACT_ENVIRONMENT?: boolean }).IS_REACT_ACT_ENVIRONMENT;
});
const click = (el: Element | null) => {
  expect(el).not.toBeNull();
  act(() => el?.dispatchEvent(new MouseEvent('click', { bubbles: true })));
};
describe('normalized report interaction', () => {
  it('combines industry and legacy cards with independent cursors and real detail identities', async () => {
    const home = await normalizedMockReportPort.getHome();
    const group = structuredClone(home.reports[0]);
    const legacy = group.analysisGroups![2];
    const unit = legacy.items[0];
    legacy.next_cursor = 'legacy-next';
    const industry = Array.from({ length: 23 }, (_, i) => ({
      ...unit,
      local_key: i === 0 ? unit.local_key : `industry-${i}`
    }));
    group.analysisGroups!.push({
      kind: 'industry_chain_analyses',
      items: industry.slice(0, 20),
      next_cursor: 'industry-next'
    });
    const detail = vi.fn();
    const read = vi
      .spyOn(normalizedMockReportPort, 'getAnalyses')
      .mockRejectedValueOnce(new Error('temporary'))
      .mockResolvedValueOnce({ items: industry.slice(20), next_cursor: null })
      .mockResolvedValueOnce({
        items: [{ ...unit, local_key: 'legacy-second' }],
        next_cursor: null
      });
    act(() =>
      root.render(<NormalizedHome group={group} query='' onDetail={detail} onEvidence={vi.fn()} />)
    );
    expect(host.querySelectorAll('.normalized-home-tab')).toHaveLength(3);
    expect(host.querySelectorAll('.normalized-home-card')).toHaveLength(23);
    expect(host.querySelector('.normalized-home-total')?.textContent).toBe('已加载 23 条主线');
    click(host.querySelectorAll('.normalized-card-path')[2]);
    expect(detail).toHaveBeenLastCalledWith({
      reportId,
      targetType: 'industry_chain_analyses',
      targetKey: unit.local_key
    });
    await act(async () => host.querySelector<HTMLButtonElement>('.normalized-home-more')!.click());
    expect(host.textContent).toContain('加载失败，点击重试');
    await act(async () => host.querySelector<HTMLButtonElement>('.normalized-home-more')!.click());
    expect(read).toHaveBeenNthCalledWith(2, reportId, 'industry_chain_analyses', 'industry-next');
    expect(host.querySelectorAll('.normalized-home-card')).toHaveLength(26);
    await act(async () => host.querySelector<HTMLButtonElement>('.normalized-home-more')!.click());
    expect(read).toHaveBeenLastCalledWith(reportId, 'concept_analyses', 'legacy-next');
    expect(host.querySelectorAll('.normalized-home-card')).toHaveLength(27);
    expect(host.querySelector('.normalized-home-total')?.textContent).toBe('27 条主线');
    expect(host.querySelector('.normalized-home-more')).toBeNull();
    click(host.querySelectorAll('.normalized-card-path')[25]);
    expect(detail).toHaveBeenLastCalledWith({
      reportId,
      targetType: 'concept_analyses',
      targetKey: unit.local_key
    });
  });
  it('uses v5 judgment origin for direct and inferred node labels', () => {
    const c = parseAnalysisChain(v5.chains['geopolitical_stories/g1/g1-2-chain'], 'g1-2-chain');
    act(() => root.render(<ChainContent c={c} reportId={reportId} onEvidence={vi.fn()} />));
    expect(host.querySelector('.normalized-node-basis')?.textContent).toBe('直接');
    click(host.querySelectorAll('.normalized-graph-node')[1]);
    expect(host.querySelector('.normalized-node-basis')?.textContent).toBe('推理');
  });
  it('keeps independent mechanism paths and conditional text intact', () => {
    const detail = parseAnalysisDetail(fixture.details['geopolitical_stories/g1'], 'g1');
    const paths = [
      '通道受扰 → 替代不足时成本可能上升',
      '运费下降 → 收益影响仍不确定',
      '没有箭头的完整段落。'
    ];
    detail.macro_impacts[0].assessment.transmission_logic = paths.join('\n');
    act(() =>
      root.render(
        <NormalizedDetailView
          detail={detail}
          reportId={reportId}
          kind='geopolitical_stories'
          onEvidence={vi.fn()}
        />
      )
    );
    const rendered = [...host.querySelectorAll('.normalized-mechanism-path')].map((path) =>
      [...path.querySelectorAll('.normalized-prose')].map((step) => step.textContent).join(' → ')
    );
    expect(rendered).toEqual(paths);
  });
  it('filters all three card groups and preserves report identity in evidence and detail navigation', async () => {
    const home = await normalizedMockReportPort.getHome(),
      evidence = vi.fn(),
      detail = vi.fn();
    act(() =>
      root.render(
        <NormalizedHome group={home.reports[0]} query='' onDetail={detail} onEvidence={evidence} />
      )
    );
    expect(host.querySelectorAll('.normalized-home-card')).toHaveLength(3);
    expect(host.querySelector('.normalized-home-total')?.textContent).toBe('3 条主线');
    for (let i = 0; i < 3; i++) {
      click(host.querySelectorAll('.normalized-home-tab')[i < 2 ? i + 1 : 0]);
      const group = fixture.groups[i],
        unit = group.items[0];
      const card = host.querySelectorAll('.normalized-home-card')[i < 2 ? 0 : 2];
      expect(card.querySelector('.normalized-card-conclusion')?.textContent).toBe(
        unit.summary.conclusion
      );
      click(card.querySelector('.normalized-card-evidence'));
      expect(evidence).toHaveBeenLastCalledWith({
        reportId,
        scopeToken: unit.summary.evidence_scope_token,
        title: unit.title
      });
      click(card.querySelector('.normalized-card-path'));
      expect(detail).toHaveBeenLastCalledWith({
        reportId,
        targetType: group.kind,
        targetKey: unit.local_key
      });
    }
  });
  it('filters the displayed cards and count without discarding the report content', async () => {
    const home = await normalizedMockReportPort.getHome();
    const group = home.reports[0];
    const render = (query: string) =>
      act(() =>
        root.render(
          <NormalizedHome group={group} query={query} onDetail={vi.fn()} onEvidence={vi.fn()} />
        )
      );
    const title = group.analysisGroups![1].items[0].title;
    render(title);
    expect(host.querySelectorAll('.normalized-home-card')).toHaveLength(1);
    expect(host.querySelector('.normalized-home-total')?.textContent).toBe('1 条主线');
    click(host.querySelectorAll('.normalized-home-tab')[1]);
    expect(host.textContent).toContain('暂无匹配的结论');
    expect(host.querySelector('.normalized-home-total')?.textContent).toBe('0 条主线');
    render('');
    expect(host.querySelectorAll('.normalized-home-card')).toHaveLength(1);
    click(host.querySelectorAll('.normalized-home-tab')[0]);
    expect(host.querySelectorAll('.normalized-home-card')).toHaveLength(3);
  });
  it('switches a macro tab to its scoped industry chain, then renders selected node prose', async () => {
    const detail = parseAnalysisDetail(fixture.details['geopolitical_stories/g1'], 'g1');
    act(() =>
      root.render(
        <NormalizedDetailView
          detail={detail}
          reportId={reportId}
          kind='geopolitical_stories'
          onEvidence={vi.fn()}
        />
      )
    );
    expect(host.querySelector('.normalized-hero-logic')?.textContent).toBe(
      detail.summary.summary.transmission_logic
    );
    expect(host.querySelector('.normalized-mechanism')?.textContent).toContain(
      detail.macro_impacts[0].assessment.transmission_logic
    );
    expect(host.querySelector('.normalized-graph-section')).toBeNull();
    await act(async () => host.querySelectorAll<HTMLButtonElement>('.normalized-tab')[1].click());
    expect(host.querySelector('.normalized-graph-section')).not.toBeNull();
    const c = fixture.chains['geopolitical_stories/g1/g1-2-chain'];
    click(host.querySelector(`[aria-label="查看${c.affected_nodes[1].name}节点详情"]`));
    const node = host.querySelector('.normalized-node-detail');
    expect(node?.textContent).toContain(c.affected_nodes[1].assessment.conclusion);
    expect(node?.textContent).toContain(c.affected_nodes[1].assessment.conditions[0]);
    expect(node?.textContent).toContain(c.affected_nodes[1].objections.summary);
    expect(node?.querySelector('.normalized-followup .normalized-prose')?.textContent).toBe(
      c.affected_nodes[1].assessment.follow_up[0]
    );
  });
  it('keeps pagination inside its group and ignores a late page after report refresh', async () => {
    const home = await normalizedMockReportPort.getHome();
    const group = structuredClone(home.reports[0]);
    group.analysisGroups![0].next_cursor = 'geo-next';
    let resolve!: (page: {
      items: import('../features/reports/normalized-contract').AnalysisSummary[];
      next_cursor: null;
    }) => void;
    const pending = new Promise<import('../features/reports/normalized-contract').AnalysisPage>(
      (r) => {
        resolve = r;
      }
    );
    const read = vi.spyOn(normalizedMockReportPort, 'getAnalyses').mockReturnValueOnce(pending);
    act(() =>
      root.render(<NormalizedHome group={group} query='' onDetail={vi.fn()} onEvidence={vi.fn()} />)
    );
    click(host.querySelector('.normalized-home-more'));
    expect(read).toHaveBeenCalledWith(reportId, 'geopolitical_stories', 'geo-next');
    click(host.querySelectorAll('.normalized-home-tab')[2]);
    expect(host.querySelector('.normalized-card-conclusion')?.textContent).toBe(
      fixture.groups[1].items[0].summary.conclusion
    );
    act(() =>
      root.render(
        <NormalizedHome
          group={structuredClone(home.reports[0])}
          query=''
          onDetail={vi.fn()}
          onEvidence={vi.fn()}
        />
      )
    );
    await act(async () =>
      resolve({
        items: [{ ...group.analysisGroups![0].items[0], local_key: 'late-page' }],
        next_cursor: null
      })
    );
    click(host.querySelectorAll('.normalized-home-tab')[1]);
    expect(host.querySelectorAll('.normalized-home-card')).toHaveLength(1);
  });
  it('allows retrying a failed chain without losing its selected tab', async () => {
    const read = vi
      .spyOn(normalizedMockReportPort, 'getAnalysisChain')
      .mockRejectedValueOnce(new Error('temporary'));
    const detail = parseAnalysisDetail(fixture.details['geopolitical_stories/g1'], 'g1');
    act(() =>
      root.render(
        <NormalizedDetailView
          detail={detail}
          reportId={reportId}
          kind='geopolitical_stories'
          onEvidence={vi.fn()}
        />
      )
    );
    await act(async () => host.querySelectorAll<HTMLButtonElement>('.normalized-tab')[1].click());
    expect(host.textContent).toContain('因果链加载失败');
    const retry = Array.from(host.querySelectorAll('button')).find(
      (b) => b.textContent === '重新加载'
    );
    expect(retry).toBeDefined();
    await act(async () => retry!.click());
    expect(read).toHaveBeenCalledTimes(2);
    expect(host.querySelector('.normalized-graph-section')).not.toBeNull();
  });
  it('shows observation follow-up without inventing confidence or assessed nodes', () => {
    const c = parseAnalysisChain(fixture.chains['concept_analyses/c1/c5-ai1'], 'c5-ai1');
    act(() => root.render(<ChainContent c={c} reportId={reportId} onEvidence={vi.fn()} />));
    expect(host.textContent).not.toContain('置信度');
    expect(host.querySelector('.normalized-node-detail')).toBeNull();
    expect(host.textContent).toContain(c.empty_state!.reason);
    expect(host.textContent).toContain(c.empty_state!.follow_up[0]);
  });
});
