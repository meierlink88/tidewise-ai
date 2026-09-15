// @vitest-environment jsdom
import { act, createElement, type ReactNode } from 'react';
import { createRoot, type Root } from 'react-dom/client';
import { beforeEach, afterEach, describe, expect, it, vi } from 'vitest';
import { normalizedMockReportPort } from '../mocks/reports/mock-port';
import { parseAnalysisChain, parseAnalysisDetail } from '../features/reports/normalized-contract';
import { NormalizedDetailView } from './report/detail/normalized-detail';
import unified from '../mocks/reports/unified-v6.json';
import original from '../mocks/reports/normalized-v5.json';

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

describe('v6 data with the established report UI', () => {
  it('renders the same visible structure and text as the original report before and after field migration', async () => {
    const legacy = parseAnalysisDetail(original.details['geopolitical_stories/g1'], 'g1');
    legacy.industry_chains = legacy.industry_chains.slice(0, 1);
    const converted = {
      ...legacy,
      macro_impacts: [],
      industry_chains: [],
      reasonings: [
        ...legacy.macro_impacts.map((m) => ({
          ...m,
          title: m.name,
          reasoning_summary: { logic: m.assessment.transmission_logic, objections: m.objections },
          reasoning_blocks: [],
          affected_assets: [{ ...m, node_local_key: '' }]
        })),
        ...legacy.industry_chains.map((h) => {
          const c = parseAnalysisChain(
            original.chains['geopolitical_stories/g1/g1-2-chain'],
            h.local_key
          );
          return { ...c, title: c.name, reasoning_blocks: [], affected_assets: c.affected_nodes };
        })
      ]
    };
    // Restrict this parity case to the first original chain, supplied by the fixed fixture.
    const render = (detail: typeof legacy) =>
      act(() =>
        root.render(
          createElement(NormalizedDetailView, {
            detail,
            reportId,
            kind: 'geopolitical_stories',
            onEvidence: vi.fn()
          })
        )
      );
    render(legacy);
    const before = host.innerHTML;
    render(converted);
    expect(host.innerHTML).toBe(before);
    const chain = parseAnalysisChain(
      original.chains['geopolitical_stories/g1/g1-2-chain'],
      legacy.industry_chains[0].local_key
    );
    const read = vi.spyOn(normalizedMockReportPort, 'getAnalysisChain').mockResolvedValue(chain);
    render(legacy);
    await act(async () =>
      host
        .querySelectorAll<HTMLButtonElement>('.normalized-tab')
        [legacy.macro_impacts.length].click()
    );
    const originalChain = host.innerHTML;
    render(converted);
    expect(host.innerHTML).toBe(originalChain);
    expect(read).toHaveBeenCalledTimes(1);
  });

  it('switches reasoning and graph nodes locally with scoped evidence', () => {
    const detail = parseAnalysisDetail(unified.details['geopolitical_stories/g1'], 'g1');
    const evidence = vi.fn();
    const read = vi.spyOn(normalizedMockReportPort, 'getAnalysisChain');
    act(() =>
      root.render(
        createElement(NormalizedDetailView, {
          detail,
          reportId,
          kind: 'geopolitical_stories',
          onEvidence: evidence
        })
      )
    );
    const first = detail.reasonings![0];
    expect(host.querySelector('.normalized-conclusion-text')?.textContent).toBe(
      first.assessment.conclusion
    );
    expect(host.querySelector('.normalized-mechanism-text')?.textContent).toContain(
      first.reasoning_summary.logic
    );
    expect(host.querySelector('.normalized-support')?.textContent).toContain(
      first.assessment.conditions[0]
    );
    click(host.querySelector('.normalized-evidence'));
    expect(evidence).toHaveBeenLastCalledWith({
      reportId,
      scopeToken: first.assessment.evidence_scope_token,
      title: `地缘政治 · ${detail.summary.title}`
    });
    click(host.querySelectorAll('.normalized-tab')[1]);
    const second = detail.reasonings![1];
    expect(host.querySelector('.normalized-conclusion-text')?.textContent).toBe(
      second.assessment.conclusion
    );
    const nodes = second.affected_assets.filter((a) => a.node_local_key);
    click(host.querySelectorAll('.normalized-graph-node')[1]);
    expect(host.querySelector('.normalized-node-conclusion')?.textContent).toBe(
      nodes[1].assessment.conclusion
    );
    click(host.querySelectorAll('.normalized-tab')[0]);
    click(host.querySelectorAll('.normalized-tab')[1]);
    expect(host.querySelector('.normalized-node-conclusion')?.textContent).toBe(
      nodes[0].assessment.conclusion
    );
    expect(read).not.toHaveBeenCalled();
    expect(host.querySelector('.unified-detail')).toBeNull();
  });
  it('keeps the original macro panel without creating a graph from assets', () => {
    const detail = parseAnalysisDetail(unified.details['geopolitical_stories/g1'], 'g1');
    detail.reasonings = [detail.reasonings![0]];
    delete detail.reasonings[0].graph;
    act(() =>
      root.render(
        createElement(NormalizedDetailView, {
          detail,
          reportId,
          kind: 'geopolitical_stories',
          onEvidence: vi.fn()
        })
      )
    );
    expect(host.querySelectorAll('.normalized-tab')).toHaveLength(1);
    expect(host.querySelector('.normalized-conclusion-text')?.textContent).toBe(
      detail.reasonings[0].assessment.conclusion
    );
    expect(host.querySelector('.normalized-counter')?.textContent).toContain(
      detail.reasonings[0].reasoning_summary.objections.summary
    );
    expect(host.querySelector('.normalized-graph-section')).toBeNull();
    expect(host.querySelector('.normalized-publication')).toBeNull();
  });
});
