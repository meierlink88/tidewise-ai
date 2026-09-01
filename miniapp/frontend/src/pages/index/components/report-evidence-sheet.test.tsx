import { Children, createElement, isValidElement, type ReactElement, type ReactNode } from 'react';
import { renderToStaticMarkup } from 'react-dom/server';
import { describe, expect, it, vi } from 'vitest';
import type { ReportEvidenceList, ReportPort } from '../../../features/reports/contract';
import { ReportError } from '../../../features/reports/contract';
import type { ReportEvidenceRoute } from '../../../features/reports/navigation';
import { mockReportPort } from '../../../mocks/reports/mock-port';
import { HomeReportEvidenceSheetView, loadHomeReportEvidences } from './report-evidence-sheet';

vi.mock('@tarojs/components', () => ({
  Button: (props: Record<string, unknown>) =>
    createElement('button', { className: props.className }, props.children as ReactNode),
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
const route: ReportEvidenceRoute = {
  reportId,
  scopeType: 'report_card',
  scopeKey: 'card-geopolitics',
  title: '地缘政治证据'
};

describe('HomeReportEvidenceSheet', () => {
  it('loads only the selected persisted Report Evidence scope', async () => {
    const result: ReportEvidenceList = {
      reportId,
      scope: { type: 'report_card', key: 'card-geopolitics' },
      items: []
    };
    const getEvidences = vi.fn().mockResolvedValue(result);
    const port = {
      getHome: vi.fn(),
      getLayer: vi.fn(),
      getIndustryChain: vi.fn(),
      getEvidences
    } as ReportPort;

    await expect(loadHomeReportEvidences(port, route)).resolves.toBe(result);
    expect(getEvidences).toHaveBeenCalledTimes(1);
    expect(getEvidences).toHaveBeenCalledWith(reportId, {
      type: 'report_card',
      key: 'card-geopolitics'
    });
  });

  it('matches all six ordered card Evidence lists used by the prototype', async () => {
    const expectedByScope = {
      'card-geopolitics': [
        '霍尔木兹海峡油轮交通量下降并遭袭击警告',
        '中国海警在黄岩岛开展执法巡查',
        '美国政府准备使用捕获法处置扣押的伊朗石油和船只',
        '美伊在霍尔木兹海峡附近发生军事冲突',
        '美伊冲突扰乱霍尔木兹海峡油运'
      ],
      'card-macroeconomics': [
        '世界银行警告全球经济增长可能放缓至1.3%',
        '美联储主席Warsh在杰克逊霍尔会议上暗示可能加息'
      ],
      'card-chn-01': ['中国举办第二届人形机器人比赛', 'DIGITIMES发表关于具身智能实体化的评论'],
      'card-chn-02': [
        '中国发布首个国家级液冷标准',
        'OpenAI在Hot Chips 2026披露自研AI加速器Jalapeño基准测试结果',
        'Google与Marvell扩大定制芯片合作'
      ],
      'card-chn-03': [
        '国家数据局披露全国智算总规模数据',
        'OpenAI在Hot Chips 2026披露自研AI加速器Jalapeño基准测试结果',
        'Google与Marvell扩大定制芯片合作',
        'InferenceXv3实现95%以上KVCache命中率'
      ],
      'card-chn-21': ['霍尔木兹海峡油轮交通量下降并遭袭击警告', '美伊冲突扰乱霍尔木兹海峡油运']
    } as const;

    for (const [scopeKey, expected] of Object.entries(expectedByScope)) {
      const result = await loadHomeReportEvidences(mockReportPort, { ...route, scopeKey });
      expect(result.items.map((item) => item.summary)).toEqual(expected);
    }
  });

  it('renders persisted display fields in BFF order without technical metadata', () => {
    const duplicate = {
      publishedAt: '2026-09-01T03:00:00Z',
      summary: '同一展示内容仍可能来自两条持久化证据。',
      keywords: ['运输', '通道']
    };
    const html = renderToStaticMarkup(
      createElement(HomeReportEvidenceSheetView, {
        title: route.title,
        state: {
          status: 'ready',
          data: {
            reportId,
            scope: { type: 'report_card', key: 'geo-card' },
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
        onRetry: vi.fn(),
        onClose: vi.fn()
      })
    );

    expect(html.indexOf('第一条')).toBeLessThan(html.indexOf(duplicate.summary));
    expect(html.match(new RegExp(duplicate.summary, 'g'))).toHaveLength(2);
    expect(html).toContain('09-01 12:00');
    expect(html).toContain('时间待确认');
    expect(html).toContain('运输');
    expect(html).toContain('report-evidence-sheet__clock');
    expect(html).toMatch(/report-clock[^"']*\.svg/);
    expect(html).not.toContain(reportId);
    expect(html).not.toContain('Evidence ID');
    expect(html).not.toContain('Event');
    expect(html).not.toContain('来源');
    expect(html).not.toContain('总数');
  });

  it('renders loading, empty and retryable error states', () => {
    expect(renderState({ status: 'loading' })).toContain('正在读取相关证据');
    expect(renderState({ status: 'empty', refreshing: false, refreshFailed: false })).toContain(
      '暂无相关证据'
    );
    expect(
      renderState({ status: 'error', error: new ReportError('serviceUnavailable') })
    ).toContain('重新加载');
  });

  it('closes from the mask while the sheet blocks bubbling and move-through', () => {
    const onClose = vi.fn();
    const view = HomeReportEvidenceSheetView({
      title: route.title,
      state: { status: 'loading' },
      onRetry: vi.fn(),
      onClose
    });
    const overlay = elementByClass(view, 'report-evidence-sheet__overlay');
    const panel = elementByClass(view, 'report-evidence-sheet__panel');
    const stopPropagation = vi.fn();

    expect(panel.props.catchMove).toBe(true);
    expect(panel.props.onClick).toBeTypeOf('function');
    (panel.props.onClick as (event: { stopPropagation: () => void }) => void)({
      stopPropagation
    });
    expect(stopPropagation).toHaveBeenCalledTimes(1);

    (overlay.props.onClick as () => void)();
    expect(onClose).toHaveBeenCalledTimes(1);
  });
});

function renderState(state: Parameters<typeof HomeReportEvidenceSheetView>[0]['state']): string {
  return renderToStaticMarkup(
    createElement(HomeReportEvidenceSheetView, {
      title: route.title,
      state,
      onRetry: vi.fn(),
      onClose: vi.fn()
    })
  );
}

function elementByClass(
  value: ReactNode,
  className: string
): ReactElement<Record<string, unknown>> {
  if (isValidElement<Record<string, unknown>>(value)) {
    if (value.props.className === className) return value;
    for (const child of Children.toArray(value.props.children as ReactNode)) {
      try {
        return elementByClass(child, className);
      } catch {
        // Continue through sibling branches until the requested page-local element is found.
      }
    }
  }
  throw new Error(`missing element ${className}`);
}
