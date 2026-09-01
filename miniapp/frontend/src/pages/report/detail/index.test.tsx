import { createElement } from 'react';
import { renderToStaticMarkup } from 'react-dom/server';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import type { ReportPort } from '../../../features/reports/contract';
import { mockReportPort } from '../../../mocks/reports/mock-port';
import { loadReportDetail, ReportDetailView } from './index';

const captured = vi.hoisted(() => ({
  buttons: [] as Array<Record<string, unknown>>,
  images: [] as Array<Record<string, unknown>>,
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
  Text: (props: Record<string, unknown>) => props.children ?? null,
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

  it('renders layer reasoning and wires every published Evidence scope', async () => {
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
    expect(copy).toContain('油品石化贸易服务产业链');
    for (const className of [
      'report-step-card__arrow',
      'report-transmission-target__arrow',
      'report-related-chain-item__arrow'
    ]) {
      const icon = captured.images.find((props) => props.className === className);
      expect(icon?.src).toEqual(expect.stringMatching(/\.svg$/));
    }
    for (const label of [
      '查看地缘政治证据',
      '查看伊朗—美以及海湾安全对抗证据',
      '查看推理步骤 1 证据',
      '查看向下传导证据'
    ]) {
      clickCapturedButton(label);
    }
    expect(onOpenEvidence.mock.calls.map(([route]) => route.scopeType)).toEqual([
      'layer',
      'anchor',
      'reasoning_step',
      'transmission_path'
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

  it('keeps anchor/node transmission targets static while layer/chain targets remain clickable', async () => {
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

    const targets = captured.views.filter(
      (props) => props.className === 'report-transmission-target'
    );
    expect(targets.filter((props) => props.role === 'button')).toHaveLength(2);
    expect(targets.filter((props) => props.onClick === undefined)).toHaveLength(1);
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
    expect(copy).toContain('交付周期与销售价格上升');
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
    clickCapturedButton('查看油品运输服务证据');
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

  it('exposes candidate-mechanism Evidence only when that scope is published', async () => {
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

    clickCapturedButton('查看待验证机制证据');
    expect(onOpenEvidence).toHaveBeenCalledWith({
      reportId,
      scopeType: 'candidate_mechanism',
      scopeKey: 'macro-candidate-01',
      title: '待验证机制证据'
    });
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
