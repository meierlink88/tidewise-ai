import { createElement } from 'react';
import { renderToStaticMarkup } from 'react-dom/server';
import { describe, expect, it, vi } from 'vitest';
import type { ReportPort } from '../../../features/reports/contract';
import { ReportError } from '../../../features/reports/contract';
import { loadReportEvidences, ReportEvidencesView } from './index';

vi.mock('@tarojs/taro', () => ({
  default: {
    getCurrentInstance: vi.fn(() => ({ router: { params: {} } })),
    pageScrollTo: vi.fn(),
    setNavigationBarTitle: vi.fn(),
    showToast: vi.fn(),
    stopPullDownRefresh: vi.fn()
  },
  usePullDownRefresh: vi.fn()
}));

vi.mock('@tarojs/components', () => ({
  Button: (props: Record<string, unknown>) => props.children ?? null,
  Image: (props: Record<string, unknown>) =>
    createElement('img', { className: props.className, src: props.src }),
  Text: (props: Record<string, unknown>) => props.children ?? null,
  View: (props: Record<string, unknown>) => props.children ?? null
}));

const reportId = 'RPT11111111-1111-4111-8111-111111111111';

describe('Report Evidence page', () => {
  it('rejects invalid route state before issuing a scoped request', async () => {
    const port = {
      getHome: vi.fn(),
      getLayer: vi.fn(),
      getIndustryChain: vi.fn(),
      getEvidences: vi.fn()
    } as unknown as ReportPort;

    await expect(loadReportEvidences(port, null)).rejects.toMatchObject({
      kind: 'invalidRequest'
    });
    expect(port.getEvidences).not.toHaveBeenCalled();
  });

  it('renders persisted Evidence projection in BFF order, including duplicate display rows', () => {
    const duplicate = {
      publishedAt: '2026-09-01T03:00:00Z',
      summary: '同一展示内容仍可能来自两条不同的持久化证据。',
      keywords: ['运输', '通道']
    };
    const html = renderToStaticMarkup(
      createElement(ReportEvidencesView, {
        state: {
          status: 'ready',
          data: {
            reportId,
            scope: { type: 'industry_chain', key: 'chn-21' },
            items: [
              { publishedAt: '2026-09-01T04:00:00Z', summary: '第一条', keywords: [] },
              duplicate,
              duplicate,
              { publishedAt: null, summary: '时间待确认的最后一条', keywords: ['待确认'] }
            ]
          },
          refreshing: false,
          refreshFailed: false
        },
        onRetry: vi.fn()
      })
    );

    expect(html.indexOf('第一条')).toBeLessThan(html.indexOf(duplicate.summary));
    expect(html.match(new RegExp(duplicate.summary, 'g'))).toHaveLength(2);
    expect(html).toContain('时间待确认');
    expect(html).toContain('运输');
    expect(html).toContain('report-evidence-item__clock');
    expect(html).toMatch(/report-clock[^"']*\.svg/);
    expect(html).not.toContain(reportId);
    expect(html).not.toContain('Evidence ID');
    expect(html).not.toContain('Event');
    expect(html).not.toContain('来源');
    expect(html).not.toContain('总数');
  });

  it('renders loading, empty and retryable error states', () => {
    expect(
      renderToStaticMarkup(
        createElement(ReportEvidencesView, {
          state: { status: 'loading' },
          onRetry: vi.fn()
        })
      )
    ).toContain('正在读取相关证据');
    expect(
      renderToStaticMarkup(
        createElement(ReportEvidencesView, {
          state: { status: 'empty', refreshing: false, refreshFailed: false },
          onRetry: vi.fn()
        })
      )
    ).toContain('暂无相关证据');
    expect(
      renderToStaticMarkup(
        createElement(ReportEvidencesView, {
          state: { status: 'error', error: new ReportError('serviceUnavailable') },
          onRetry: vi.fn()
        })
      )
    ).toContain('重新加载');
  });
});
