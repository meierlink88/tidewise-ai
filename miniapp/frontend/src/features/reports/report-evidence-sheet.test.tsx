import { createElement, type ReactNode } from 'react';
import { renderToStaticMarkup } from 'react-dom/server';
import { describe, expect, it, vi } from 'vitest';
import { mockReportPort } from '../../mocks/reports/mock-port';
import type { ReportEvidenceList, ReportPort } from './contract';
import { ReportError } from './contract';
import type { ReportEvidenceRoute } from './navigation';
import { loadReportEvidences, ReportEvidenceSheetView } from './report-evidence-sheet';

vi.mock('@tarojs/components', () => ({
  Button: (props: Record<string, unknown>) =>
    createElement(
      'button',
      { className: props.className, 'aria-label': props.ariaLabel },
      props.children as ReactNode
    ),
  Image: (props: Record<string, unknown>) =>
    createElement('img', { className: props.className, src: props.src }),
  RootPortal: (props: Record<string, unknown>) =>
    createElement('aside', {}, props.children as ReactNode),
  ScrollView: (props: Record<string, unknown>) =>
    createElement('section', { className: props.className }, props.children as ReactNode),
  Text: (props: Record<string, unknown>) =>
    createElement('span', { className: props.className }, props.children as ReactNode),
  View: (props: Record<string, unknown>) =>
    createElement('div', { className: props.className }, props.children as ReactNode)
}));

const reportId = 'RPT11111111-1111-4111-8111-111111111111';
const scopeToken = 'RPE11111111-1111-4111-8111-111111111111';
const route: ReportEvidenceRoute = { reportId, scopeToken, title: '地缘政治证据' };

describe('ReportEvidenceSheet', () => {
  it('loads only the server-issued scope token', async () => {
    const result: ReportEvidenceList = { reportId, scopeToken, items: [] };
    const getEvidences = vi.fn().mockResolvedValue(result);
    const port = {
      getHome: vi.fn(),
      getIndustryChains: vi.fn(),
      getLayer: vi.fn(),
      getIndustryChain: vi.fn(),
      getEvidences
    } as ReportPort;

    await expect(loadReportEvidences(port, route)).resolves.toBe(result);
    expect(getEvidences).toHaveBeenCalledWith(reportId, scopeToken);
  });

  it('loads mock summary evidence including keywords', async () => {
    const home = await mockReportPort.getHome();
    const token = home.reports[0]?.cards[0]?.evidenceScopeToken;
    if (!token) throw new Error('expected a summary evidence token');
    const result = await loadReportEvidences(mockReportPort, {
      reportId,
      scopeToken: token,
      title: '地缘政治证据'
    });
    expect(result.items[0]?.keywords.length).toBeGreaterThan(0);
  });

  it('renders evidence summaries and keyword rows without technical identifiers', () => {
    const html = renderToStaticMarkup(
      createElement(ReportEvidenceSheetView, {
        title: route.title,
        state: {
          status: 'ready',
          data: {
            reportId,
            scopeToken,
            items: [
              {
                publishedAt: '2026-09-01T04:00:00Z',
                summary: '第一条证据摘要',
                keywords: ['运输', '通道']
              },
              { publishedAt: null, summary: '时间待确认的证据', keywords: [] }
            ]
          },
          refreshing: false,
          refreshFailed: false
        },
        onRetry: vi.fn(),
        onClose: vi.fn()
      })
    );
    expect(html).toContain('第一条证据摘要');
    expect(html).toContain('运输');
    expect(html).toContain('时间待确认');
    expect(html).toContain('report-evidence-sheet__close');
    expect(html).toContain('report-evidence-sheet__close-label');
    expect(html).toContain('关闭相关证据');
    expect(html).not.toContain(scopeToken);
    expect(html).not.toContain(reportId);
  });

  it('renders loading, empty and retryable error states', () => {
    const render = (state: Parameters<typeof ReportEvidenceSheetView>[0]['state']) =>
      renderToStaticMarkup(
        createElement(ReportEvidenceSheetView, {
          title: route.title,
          state,
          onRetry: vi.fn(),
          onClose: vi.fn()
        })
      );
    expect(render({ status: 'loading' })).toContain('正在读取相关证据');
    expect(render({ status: 'empty', refreshing: false, refreshFailed: false })).toContain(
      '暂无相关证据'
    );
    expect(render({ status: 'error', error: new ReportError('serviceUnavailable') })).toContain(
      '重新加载'
    );
  });
});
