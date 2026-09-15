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
            kind: 'macroeconomic_stories',
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
          kind: 'macroeconomic_stories',
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
      title: `宏观经济 · ${detail.summary.title}`
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
          kind: 'macroeconomic_stories',
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

describe('geopolitical prototype with report-owned data', () => {
  it('preserves the hero and keeps other report kinds on the established body', () => {
    const detail = parseAnalysisDetail(unified.details['geopolitical_stories/g1'], 'g1');
    const render = (
      kind: 'geopolitical_stories' | 'macroeconomic_stories' | 'industry_chain_analyses'
    ) =>
      act(() =>
        root.render(
          createElement(NormalizedDetailView, { detail, reportId, kind, onEvidence: vi.fn() })
        )
      );
    render('macroeconomic_stories');
    const hero = host.querySelector('.normalized-hero')!.outerHTML;
    for (const kind of ['geopolitical_stories', 'industry_chain_analyses'] as const) {
      render(kind);
      expect(host.querySelector('.normalized-hero')!.outerHTML).toBe(hero);
      expect(!!host.querySelector('.geo-detail-main')).toBe(kind === 'geopolitical_stories');
      expect(!!host.querySelector('.normalized-main')).toBe(kind !== 'geopolitical_stories');
    }
  });

  it('renders report metrics, allocation and intact timing, switches assets and reasoning locally', () => {
    const detail = parseAnalysisDetail(unified.details['geopolitical_stories/g1'], 'g1');
    const first = detail.reasonings![0];
    const asset = first.affected_assets[0];
    first.reasoning_summary.logic = '这是报告提供的陈述句。';
    first.reasoning_blocks = [
      {
        local_key: 'block',
        title: '报告区块标题',
        explanation: '报告区块解释',
        relation_type: 'comparison',
        nodes: [
          {
            local_key: 'metric',
            name: '市场指标',
            description: '指标说明',
            metrics: [
              {
                name: '市场指标',
                display_value: '+9.5%',
                unit: '%',
                measure_type: 'change_rate',
                value_nature: 'observed',
                period_label: '过去一周',
                evidence_scope_token: asset.assessment.evidence_scope_token,
                evidence_count: 1
              }
            ]
          }
        ]
      }
    ];
    const metric = first.reasoning_blocks[0].nodes[0].metrics[0];
    first.reasoning_blocks[0].nodes.push(
      {
        local_key: 'range',
        name: '成本',
        metrics: [{ ...metric, display_value: '1万至2万元', unit: '元' }]
      },
      {
        local_key: 'unit',
        name: '运价',
        metrics: [{ ...metric, display_value: '80万', unit: '美元/日' }]
      }
    );
    first.affected_assets = [4, 0, undefined, -3].map((delta, i) => ({
      ...asset,
      local_key: `asset-${i}`,
      name: `报告资产${i}`,
      assessment: {
        ...asset.assessment,
        weight_delta_pp: delta,
        direction: 'warming',
        conclusion: `资产结论${i}`,
        forecast_window: {
          ...asset.assessment.forecast_window,
          description: i === 0 ? '价格反应 · 小时级' : ''
        }
      }
    }));
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
    expect(host.querySelector('.geo-detail-question')!.textContent).toBe(
      '关键机制问题这是报告提供的陈述句。'
    );
    expect(host.querySelector('.geo-detail-metric-value')!.textContent).toBe('+9.5%');
    expect(host.querySelector('.geo-detail-metric-note')!.textContent).toBe('过去一周');
    expect(host.querySelectorAll('.geo-detail-metric-value')[1].textContent).toBe('1万至2万元');
    expect(host.querySelectorAll('.geo-detail-metric-value')[2].textContent).toBe('80万美元/日');
    expect(host.querySelectorAll('.geo-detail-asset-tile')).toHaveLength(4);
    expect(host.querySelector('.geo-detail-timing-text')!.textContent).toBe('价格反应 · 小时级');
    expect(
      host.querySelector('.geo-detail-asset-analysis .geo-detail-allocation')!.textContent
    ).toBe('↑4%');
    click(host.querySelector('.geo-detail-asset-analysis .geo-detail-evidence'));
    expect(evidence).toHaveBeenLastCalledWith({
      reportId,
      scopeToken: asset.assessment.evidence_scope_token,
      title: `地缘政治 · ${detail.summary.title}`
    });
    click(host.querySelectorAll('.geo-detail-asset-tile')[1]);
    expect(
      host.querySelector('.geo-detail-asset-analysis .geo-detail-allocation')!.textContent
    ).toBe('—');
    expect(host.querySelector('.geo-detail-timing')).toBeNull();
    click(host.querySelectorAll('.geo-detail-asset-tile')[2]);
    expect(
      host.querySelector('.geo-detail-asset-analysis .geo-detail-allocation')!.textContent
    ).toBe('升温');
    click(host.querySelectorAll('.geo-detail-asset-tile')[3]);
    expect(
      host.querySelector('.geo-detail-asset-analysis .geo-detail-allocation')!.textContent
    ).toBe('↓3%');
    click(host.querySelectorAll('.geo-detail-tab')[1]);
    click(host.querySelectorAll('.geo-detail-tab')[0]);
    expect(host.querySelector('.geo-detail-timing-text')!.textContent).toBe('价格反应 · 小时级');
    expect(read).not.toHaveBeenCalled();
  });

  it('handles a single reasoning with no metrics or assets without adding sample content', () => {
    const detail = parseAnalysisDetail(unified.details['geopolitical_stories/g1'], 'g1');
    detail.reasonings = [{ ...detail.reasonings![0], reasoning_blocks: [], affected_assets: [] }];
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
    expect(host.querySelector('.geo-detail-reasoning-tabs')).toBeNull();
    expect(host.querySelector('.geo-detail-metric-row')).toBeNull();
    expect(host.querySelector('.geo-detail-assets-surface')).toBeNull();
    expect(host.textContent).toContain(detail.reasonings[0].assessment.conclusion);
  });
});

it('loads geopolitical null metadata and keeps story Evidence accessible', () => {
  const raw = JSON.parse(JSON.stringify(unified.details['geopolitical_stories/g1']));
  const clear = (assessment: Record<string, unknown>) => {
    assessment.confidence = null;
    assessment.forecast_window = null;
    assessment.follow_up = null;
    assessment.evidence_scope_token = null;
    assessment.evidence_count = 0;
  };
  raw.summary.affected_anchors.forEach((a: { assessment: Record<string, unknown> }) =>
    clear(a.assessment)
  );
  raw.reasonings.forEach(
    (r: {
      assessment: Record<string, unknown>;
      affected_assets: { assessment: Record<string, unknown> }[];
    }) => {
      clear(r.assessment);
      r.affected_assets.forEach((a) => clear(a.assessment));
    }
  );
  const detail = parseAnalysisDetail(raw, 'g1', 'geopolitical_stories');
  expect(() => parseAnalysisDetail(raw, 'g1', 'macroeconomic_stories')).toThrow();
  const evidence = vi.fn();
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
  expect(host.textContent).toContain(detail.summary.summary.conclusion);
  expect(host.textContent).not.toContain('传导时间');
  const token = detail.summary.summary.evidence_scope_token;
  expect(token).toBeTruthy();
  click(host.querySelector('.normalized-evidence'));
  expect(evidence).toHaveBeenCalledWith(expect.objectContaining({ scopeToken: token, reportId }));
});
