import { createElement } from 'react';
import { renderToStaticMarkup } from 'react-dom/server';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import type { ReportPort } from '../../../features/reports/contract';
import { mockReportPort } from '../../../mocks/reports/mock-port';
import { loadReportDetail, ReportDetailView } from './index';

const captured = vi.hoisted(() => ({
  buttons: [] as Array<Record<string, unknown>>,
  images: [] as Array<Record<string, unknown>>,
  texts: [] as Array<Record<string, unknown>>,
  views: [] as Array<Record<string, unknown>>
}));

vi.mock('@tarojs/taro', () => ({
  default: {
    getCurrentInstance: vi.fn(() => ({ router: { params: {} } })),
    navigateTo: vi.fn(),
    pageScrollTo: vi.fn(),
    pxTransform: vi.fn((value: number) => `${value}rpx`),
    setNavigationBarTitle: vi.fn(),
    showToast: vi.fn(),
    stopPullDownRefresh: vi.fn()
  },
  usePullDownRefresh: vi.fn()
}));

vi.mock('@tarojs/components', () => ({
  Button: (props: Record<string, unknown>) => {
    captured.buttons.push(props);
    return props.children ?? null;
  },
  Image: (props: Record<string, unknown>) => {
    captured.images.push(props);
    return null;
  },
  ScrollView: (props: Record<string, unknown>) => props.children ?? null,
  Text: (props: Record<string, unknown>) => {
    captured.texts.push(props);
    return props.children ?? null;
  },
  View: (props: Record<string, unknown>) => {
    captured.views.push(props);
    return props.children ?? null;
  }
}));

const reportId = 'RPT11111111-1111-4111-8111-111111111111';

describe('Report detail page', () => {
  beforeEach(() => {
    captured.buttons.length = 0;
    captured.images.length = 0;
    captured.texts.length = 0;
    captured.views.length = 0;
  });

  it('rejects invalid route state before calling a Report port', async () => {
    const port = {
      getHome: vi.fn(),
      getLayer: vi.fn(),
      getIndustryChain: vi.fn(),
      getEvidences: vi.fn()
    } as unknown as ReportPort;

    await expect(loadReportDetail(port, null)).rejects.toMatchObject({
      kind: 'invalidRequest'
    });
    expect(port.getLayer).not.toHaveBeenCalled();
    expect(port.getIndustryChain).not.toHaveBeenCalled();
  });

  it('loads macroeconomics as the in-page continuation of geopolitics', async () => {
    const loaded = await loadReportDetail(mockReportPort, {
      reportId,
      targetType: 'layer',
      targetKey: 'geopolitics'
    });

    expect(loaded.targetType).toBe('layer');
    if (loaded.targetType !== 'layer') throw new Error('expected layer detail');
    expect(loaded.detail.layer.key).toBe('geopolitics');
    expect(loaded.continuationDetail?.layer.key).toBe('macroeconomics');
  });

  it('renders the approved layer hierarchy without deprecated reasoning sections', async () => {
    const detail = await mockReportPort.getLayer(reportId, 'geopolitics');
    const onOpenEvidence = vi.fn();
    const onOpenDetail = vi.fn();
    const copy = renderToStaticMarkup(
      createElement(ReportDetailView, {
        state: {
          status: 'ready',
          data: { targetType: 'layer', detail },
          refreshing: false,
          refreshFailed: false
        },
        onRetry: vi.fn(),
        onOpenDetail,
        onOpenEvidence
      })
    );

    expect(copy).toContain('一句话结论');
    expect(copy).toContain('影响锚点');
    expect(copy).toContain('向下传导');
    expect(copy).toContain('反转条件');
    expect(copy).toContain('油品石化贸易服务产业链');
    expect(copy).toContain('风力发电产业链');
    expect(detail.relatedIndustryChains).toHaveLength(54);
    expect(copy).not.toContain('推理步骤');
    expect(copy).not.toContain('不确定性与反转条件');
    expect(copy).not.toContain('当前事件如何从地缘政治与宏观经济传导至产业链（动态传导）');
    for (const className of [
      'report-layer-heading__icon',
      'report-transmission-card__link-icon',
      'report-related-chain-item__arrow'
    ]) {
      const icon = captured.images.find((props) => props.className === className);
      expect(icon?.src).toEqual(expect.stringMatching(/\.svg$/));
    }
    expect(
      captured.buttons.some(
        (props) => props.ariaLabel === '查看伊朗—美以及海湾安全对抗证据：直接证据'
      )
    ).toBe(false);
    for (const label of ['查看地缘政治证据', '查看伊朗—美以及海湾安全对抗证据：依据']) {
      clickCapturedButton(label);
    }
    expect(onOpenEvidence.mock.calls.map(([route]) => route.scopeType)).toEqual([
      'layer',
      'anchor'
    ]);

    const chainEntry = captured.views.find(
      (props) => props.ariaLabel === '查看油品石化贸易服务产业链推理详情'
    );
    expect(chainEntry?.role).toBe('button');
    (chainEntry?.onClick as (() => void) | undefined)?.();
    expect(onOpenDetail).toHaveBeenCalledWith({
      reportId,
      targetType: 'industry_chain',
      targetKey: 'chn-21'
    });
  });

  it('renders semantic nature labels for direct and inferred layer anchors', async () => {
    const original = await mockReportPort.getLayer(reportId, 'geopolitics');
    const detail = {
      ...original,
      layer: {
        ...original.layer,
        anchors: [
          original.layer.anchors[0],
          {
            ...original.layer.anchors[1],
            nature: { code: 'reasoning_hypothesis' as const, label: '推理假设' as const },
            hasEvidence: true
          }
        ]
      }
    };

    renderToStaticMarkup(
      createElement(ReportDetailView, {
        state: {
          status: 'ready',
          data: { targetType: 'layer', detail },
          refreshing: false,
          refreshFailed: false
        },
        onRetry: vi.fn(),
        onOpenDetail: vi.fn(),
        onOpenEvidence: vi.fn()
      })
    );

    expect(
      captured.texts.some(
        (props) =>
          props.children === '直接证据' &&
          props.className === 'report-nature-chip report-nature-chip--direct_evidence'
      )
    ).toBe(true);
    expect(
      captured.texts.some(
        (props) =>
          props.children === '推理假设' &&
          props.className === 'report-nature-chip report-nature-chip--reasoning_hypothesis'
      )
    ).toBe(true);
    expect(
      captured.buttons.some((props) => props.ariaLabel === '查看南海海洋权益与安全争端证据：依据')
    ).toBe(false);
  });

  it('keeps one explicit layer or chain continuation per transmission card', async () => {
    const original = await mockReportPort.getLayer(reportId, 'geopolitics');
    const firstPath = original.layer.downwardTransmission.publishedPaths[0];
    const detail = {
      ...original,
      layer: {
        ...original.layer,
        downwardTransmission: {
          ...original.layer.downwardTransmission,
          publishedPaths: [
            {
              ...firstPath,
              targetRefs: [
                ...firstPath.targetRefs,
                {
                  ref: { type: 'anchor' as const, key: 'macro-a01' },
                  label: '增长预期锚点',
                  result: { code: 'cooling' as const, label: '降温' as const }
                }
              ]
            },
            ...original.layer.downwardTransmission.publishedPaths.slice(1)
          ]
        }
      }
    };
    renderToStaticMarkup(
      createElement(ReportDetailView, {
        state: {
          status: 'ready',
          data: { targetType: 'layer', detail },
          refreshing: false,
          refreshFailed: false
        },
        onRetry: vi.fn(),
        onOpenDetail: vi.fn(),
        onOpenEvidence: vi.fn()
      })
    );

    const targets = captured.views.filter((props) =>
      String(props.className).includes('report-transmission-card__heading')
    );
    expect(targets.filter((props) => props.role === 'button')).toHaveLength(2);
    expect(targets.some((props) => props.ariaLabel === '查看增长预期锚点推理详情')).toBe(false);
  });

  it('renders explicit chain edges and wires chain/node Evidence routes', async () => {
    const detail = await mockReportPort.getIndustryChain(reportId, 'chn-21');
    const onOpenEvidence = vi.fn();
    const copy = renderToStaticMarkup(
      createElement(ReportDetailView, {
        state: {
          status: 'ready',
          data: { targetType: 'industry_chain', detail },
          refreshing: false,
          refreshFailed: false
        },
        onRetry: vi.fn(),
        onOpenDetail: vi.fn(),
        onOpenEvidence
      })
    );

    expect(copy).toContain('产业链图');
    expect(copy).toContain('依赖');
    expect(copy).toContain('交付周期 UP/HIGH');
    expect(copy).toContain('销售价格 UP/HIGH');
    for (const className of [
      'report-chain-boundary__icon report-chain-boundary__icon--gap',
      'report-chain-boundary__icon report-chain-boundary__icon--stop'
    ]) {
      const icon = captured.images.find((props) => props.className === className);
      expect(icon?.src).toEqual(expect.stringMatching(/\.svg$/));
    }
    const canvas = captured.views.find((props) => props.className === 'report-chain-canvas');
    const edge = captured.views.find(
      (props) => props.className === 'report-chain-edge report-chain-edge--adjacent'
    );
    const arrow = captured.views.find(
      (props) =>
        typeof props.className === 'string' && props.className.includes('report-chain-edge-arrow')
    );
    expect(canvas?.style).toEqual(expect.any(String));
    expect(canvas?.style).toContain('width:');
    expect(canvas?.style).toContain('padding-top:');
    expect(edge?.style).toEqual(expect.any(String));
    expect(edge?.style).toContain('top:');
    expect(edge?.style).toContain('left:');
    expect(edge?.style).toContain('width:');
    expect(arrow?.style).toEqual(expect.any(String));
    expect(arrow?.style).toContain('top:');
    expect(arrow?.style).toContain('left:');
    clickCapturedButton('查看油品石化贸易服务产业链证据');
    expect(
      captured.buttons.some((props) => props.ariaLabel === '查看油品运输服务证据：直接证据')
    ).toBe(false);
    clickCapturedButton('查看油品运输服务证据：依据');
    expect(onOpenEvidence).toHaveBeenNthCalledWith(1, {
      reportId,
      scopeType: 'industry_chain',
      scopeKey: 'chn-21',
      title: '油品石化贸易服务产业链证据'
    });
    expect(onOpenEvidence).toHaveBeenNthCalledWith(2, {
      reportId,
      scopeType: 'industry_chain_node',
      scopeKey: 'chn-21-n01',
      title: '油品运输服务证据'
    });
  });

  it('does not expose Evidence actions for a reasoning-hypothesis node', async () => {
    const detail = await mockReportPort.getIndustryChain(reportId, 'chn-01');
    const hypothesisNode = detail.industryChain.nodes.find(
      (nodeItem) => nodeItem.nature.code === 'reasoning_hypothesis'
    );
    if (!hypothesisNode) throw new Error('expected a reasoning-hypothesis node');
    const hypothesisWithSupportingEvidence = { ...hypothesisNode, hasEvidence: true };
    const hypothesisFirstDetail = {
      ...detail,
      industryChain: {
        ...detail.industryChain,
        nodes: [
          hypothesisWithSupportingEvidence,
          ...detail.industryChain.nodes.filter((nodeItem) => nodeItem.key !== hypothesisNode.key)
        ]
      }
    };

    const copy = renderToStaticMarkup(
      createElement(ReportDetailView, {
        state: {
          status: 'ready',
          data: { targetType: 'industry_chain', detail: hypothesisFirstDetail },
          refreshing: false,
          refreshFailed: false
        },
        onRetry: vi.fn(),
        onOpenDetail: vi.fn(),
        onOpenEvidence: vi.fn()
      })
    );

    expect(copy).toContain('暂无直接证据，待后续验证');
    expect(
      captured.buttons.some((props) =>
        String(props.ariaLabel).startsWith(`查看${hypothesisNode.name}证据：`)
      )
    ).toBe(false);
  });

  it('renders chain summary pills and semantic node nature labels', async () => {
    const detail = await mockReportPort.getIndustryChain(reportId, 'chn-01');
    renderToStaticMarkup(
      createElement(ReportDetailView, {
        state: {
          status: 'ready',
          data: { targetType: 'industry_chain', detail },
          refreshing: false,
          refreshFailed: false
        },
        onRetry: vi.fn(),
        onOpenDetail: vi.fn(),
        onOpenEvidence: vi.fn()
      })
    );

    for (const className of [
      'report-chain-metric report-chain-metric--result',
      'report-chain-metric report-chain-metric--window',
      'report-chain-metric report-chain-metric--confidence'
    ]) {
      expect(captured.views.some((props) => String(props.className).startsWith(className))).toBe(
        true
      );
    }
    for (const className of [
      'report-chain-metric__icon report-chain-metric__icon--result',
      'report-chain-metric__icon',
      'report-chain-metric__icon'
    ]) {
      const matchingIcons = captured.images.filter((props) => props.className === className);
      expect(matchingIcons.length).toBeGreaterThan(0);
      expect(matchingIcons[0]?.src).toEqual(expect.stringMatching(/\.svg$/));
    }
    expect(
      captured.texts.some(
        (props) =>
          props.children === '直接证据' &&
          props.className ===
            'report-nature-chip report-nature-chip--direct_evidence report-chain-node__nature'
      )
    ).toBe(true);
    expect(
      captured.texts.some(
        (props) =>
          props.children === '推理假设' &&
          props.className ===
            'report-nature-chip report-nature-chip--reasoning_hypothesis report-chain-node__nature'
      )
    ).toBe(true);
  });

  it('loads every published industry-chain detail exposed by the layer list', async () => {
    const layer = await mockReportPort.getLayer(reportId, 'geopolitics');
    const finalChain = layer.relatedIndustryChains.at(-1);

    expect(finalChain).toMatchObject({ key: 'chn-54', name: '风力发电产业链' });
    await expect(mockReportPort.getIndustryChain(reportId, 'chn-54')).resolves.toMatchObject({
      industryChain: { key: 'chn-54', name: '风力发电产业链' }
    });
  });

  it('keeps candidate mechanisms as non-evidence transmission hypotheses', async () => {
    const original = await mockReportPort.getLayer(reportId, 'macroeconomics');
    const candidate = original.layer.downwardTransmission.candidateMechanisms[0];
    const detail = {
      ...original,
      layer: {
        ...original.layer,
        downwardTransmission: {
          ...original.layer.downwardTransmission,
          candidateMechanisms: [{ ...candidate, hasEvidence: true }]
        }
      }
    };
    const onOpenEvidence = vi.fn();
    renderToStaticMarkup(
      createElement(ReportDetailView, {
        state: {
          status: 'ready',
          data: { targetType: 'layer', detail },
          refreshing: false,
          refreshFailed: false
        },
        onRetry: vi.fn(),
        onOpenDetail: vi.fn(),
        onOpenEvidence
      })
    );

    expect(captured.buttons.some((props) => props.ariaLabel === '查看待验证机制证据')).toBe(false);
    expect(onOpenEvidence).not.toHaveBeenCalled();
  });
});

function clickCapturedButton(label: string): void {
  const button = captured.buttons.find((props) => props.ariaLabel === label);
  if (!button) throw new Error(`missing button: ${label}`);
  const stopPropagation = vi.fn();
  (button.onClick as (event: { stopPropagation: () => void }) => void)({
    stopPropagation
  });
  expect(stopPropagation).toHaveBeenCalledOnce();
}
