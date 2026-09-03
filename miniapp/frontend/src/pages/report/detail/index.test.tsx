import { createElement, type ReactNode } from 'react';
import { renderToStaticMarkup } from 'react-dom/server';
import { describe, expect, it, vi } from 'vitest';
import { ReportError, type ReportPort } from '../../../features/reports/contract';
import { mockReportPort } from '../../../mocks/reports/mock-port';
import { loadReportDetail, ReportDetailView } from './index';

vi.mock('@tarojs/taro', () => ({
  default: {
    pxTransform: (value: number) => `${value}px`,
    pageScrollTo: vi.fn(),
    getCurrentInstance: () => ({ router: { params: {} } }),
    setNavigationBarTitle: vi.fn(),
    stopPullDownRefresh: vi.fn(),
    showToast: vi.fn()
  },
  usePullDownRefresh: vi.fn()
}));

vi.mock('@tarojs/components', () => ({
  Button: (props: Record<string, unknown>) =>
    createElement(
      'button',
      { className: props.className, 'aria-label': props.ariaLabel, disabled: props.disabled },
      props.children as ReactNode
    ),
  Image: (props: Record<string, unknown>) =>
    createElement('img', { className: props.className, src: props.src }),
  ScrollView: (props: Record<string, unknown>) =>
    createElement('section', { className: props.className }, props.children as ReactNode),
  Text: (props: Record<string, unknown>) =>
    createElement('span', { className: props.className }, props.children as ReactNode),
  View: (props: Record<string, unknown>) =>
    createElement('div', { className: props.className }, props.children as ReactNode)
}));

const reportId = 'RPT11111111-1111-4111-8111-111111111111';

describe('Report detail page', () => {
  it('loads optional layer continuation without making macroeconomics mandatory', async () => {
    await expect(
      loadReportDetail(mockReportPort, {
        reportId,
        targetType: 'layer',
        targetKey: 'geopolitics'
      })
    ).resolves.toMatchObject({
      targetType: 'layer',
      continuationDetail: { layer: { key: 'macroeconomics' } }
    });

    const port = {
      ...mockReportPort,
      getLayer: vi.fn(async (_id: string, key: 'geopolitics' | 'macroeconomics') => {
        if (key === 'macroeconomics') throw new ReportError('layerUnavailable');
        return mockReportPort.getLayer(reportId, key);
      })
    } as unknown as ReportPort;
    await expect(
      loadReportDetail(port, { reportId, targetType: 'layer', targetKey: 'geopolitics' })
    ).resolves.toMatchObject({ targetType: 'layer', detail: { layer: { key: 'geopolitics' } } });
  });

  it('exposes all 54 related industry chains from the report snapshot', async () => {
    const layer = await mockReportPort.getLayer(reportId, 'macroeconomics');
    expect(layer.relatedIndustryChains).toHaveLength(54);
    await expect(mockReportPort.getIndustryChain(reportId, 'chn-54')).resolves.toMatchObject({
      industryChain: { key: 'chn-54' }
    });
  });

  it('renders conclusion basis and validation status separately', async () => {
    const detail = await mockReportPort.getIndustryChain(reportId, 'chn-01');
    const html = renderToStaticMarkup(
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
    expect(html).toContain('直接证据');
    expect(html).toContain('推理假设');
    expect(html).toContain('待验证');
    expect(html).toContain('report-nature-chip--direct_evidence');
    expect(html).toContain('report-nature-chip--reasoning_hypothesis');
    expect(html).toContain('report-nature-chip--pending_validation');
  });

  it('shows Evidence actions only for direct-evidence nodes with a server token', async () => {
    const detail = await mockReportPort.getIndustryChain(reportId, 'chn-01');
    const directNode = detail.industryChain.nodes.find(
      (node) => node.conclusionBasis?.code === 'direct_evidence'
    );
    const hypothesisNode = detail.industryChain.nodes.find(
      (node) => node.conclusionBasis?.code === 'reasoning_hypothesis'
    );
    if (!directNode || !hypothesisNode) throw new Error('expected direct and hypothesis nodes');
    const directHTML = renderToStaticMarkup(
      createElement(ReportDetailView, {
        state: {
          status: 'ready',
          data: {
            targetType: 'industry_chain',
            detail: {
              ...detail,
              industryChain: { ...detail.industryChain, nodes: [directNode] }
            }
          },
          refreshing: false,
          refreshFailed: false
        },
        onRetry: vi.fn(),
        onOpenDetail: vi.fn(),
        onOpenEvidence: vi.fn()
      })
    );
    const hypothesisHTML = renderToStaticMarkup(
      createElement(ReportDetailView, {
        state: {
          status: 'ready',
          data: {
            targetType: 'industry_chain',
            detail: {
              ...detail,
              industryChain: { ...detail.industryChain, nodes: [hypothesisNode] }
            }
          },
          refreshing: false,
          refreshFailed: false
        },
        onRetry: vi.fn(),
        onOpenDetail: vi.fn(),
        onOpenEvidence: vi.fn()
      })
    );
    expect(directHTML).toContain(`查看${directNode.name}证据：依据`);
    expect(hypothesisHTML).not.toContain(`查看${hypothesisNode.name}证据：依据`);
  });

  it('renders graph relation labels and boundary icons', async () => {
    const detail = await mockReportPort.getIndustryChain(reportId, 'chn-01');
    detail.industryChain.topologyNodes.push({ key: 'unassessed-node', name: '结构上下文节点' });
    const html = renderToStaticMarkup(
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
    expect(html).toContain('组成');
    expect(html).toContain('反证与缺口');
    expect(html).toContain('停止条件');
    expect(html).toContain('report-info');
    expect(html).toContain('report-warning');
    expect(html).toContain('结构上下文节点');
    expect(html).toContain('暂无本期评估');
  });
});
