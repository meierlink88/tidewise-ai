// @vitest-environment jsdom
import { act, createElement, type ReactNode, type CSSProperties } from 'react';
import { createRoot, type Root } from 'react-dom/client';
import { afterEach, beforeEach, expect, it, vi } from 'vitest';
import sample from '../../../../features/design-preview/report-data/standard-report.json';
import StandardReport from './StandardReport';
import type { CompanyReport } from './contract';

vi.mock('@tarojs/components', () => {
  function element(tag: string) {
    return ({
      children,
      className,
      style,
      ariaLabel,
      onClick
    }: {
      children?: ReactNode;
      className?: string;
      style?: CSSProperties;
      ariaLabel?: string;
      onClick?: () => void;
    }) => createElement(tag, { className, style, 'aria-label': ariaLabel, onClick }, children);
  }
  return {
    View: element('div'),
    Image: element('img'),
    Text: element('span'),
    Button: element('button'),
    ScrollView: element('section')
  };
});
vi.mock('../../../../features/design-preview/shell', () => ({
  PreviewSheet: ({
    title,
    close,
    children
  }: {
    title: string;
    close: () => void;
    children: ReactNode;
  }) =>
    createElement(
      'section',
      { role: 'dialog', 'aria-label': title },
      createElement('button', { 'aria-label': '关闭详情', onClick: close }, '关闭'),
      children
    )
}));
let root: Root, host: HTMLDivElement;
beforeEach(() => {
  Object.assign(globalThis, { IS_REACT_ACT_ENVIRONMENT: true });
  host = document.createElement('div');
  root = createRoot(host);
});
afterEach(() => {
  act(() => root.unmount());
});
function click(label: string) {
  const button = host.querySelector<HTMLButtonElement>(`[aria-label="${label}"]`);
  expect(button).not.toBeNull();
  act(() => button?.click());
}
it('renders supplied identity and scores instead of retaining Xinquan constants', () => {
  const report: CompanyReport = {
    ...sample,
    report_info: { ...sample.report_info, company_name: '数据驱动样例' },
    debate: {
      ...sample.debate,
      summary: {
        ...sample.debate.summary,
        bull_score: { score: 10, max_score: 70 },
        bear_score: { score: 60, max_score: 70 }
      }
    },
    four_dimensions: {
      ...sample.four_dimensions,
      detail: {
        ...sample.four_dimensions.detail,
        fundamental: {
          ...sample.four_dimensions.detail.fundamental,
          six_dimension_scores: {
            ...sample.four_dimensions.detail.fundamental.six_dimension_scores,
            quality_profile:
              sample.four_dimensions.detail.fundamental.six_dimension_scores.quality_profile.map(
                (row) => ({ ...row, score: 2 })
              )
          }
        }
      }
    }
  };
  act(() => root.render(<StandardReport report={report} />));
  expect(host.textContent).toContain('数据驱动样例');
  expect(host.textContent).not.toContain('新泉股份');
  expect(host.querySelector<HTMLElement>('.bull-segment')?.style.flexGrow).toBe('10');
  click('基本面详情与逻辑');
  expect(host.querySelector('[role="dialog"]')?.textContent).toContain('盈利能力');
  expect(host.querySelector<HTMLElement>('.standard-quality-row .standard-fill')?.style.width).toBe(
    '20%'
  );
});
it('keeps missing scores distinct from zero and shows zero fund flow without invalid geometry', () => {
  const report: CompanyReport = {
    ...sample,
    four_dimensions: {
      ...sample.four_dimensions,
      detail: {
        ...sample.four_dimensions.detail,
        fundamental: {
          ...sample.four_dimensions.detail.fundamental,
          six_dimension_scores: {
            ...sample.four_dimensions.detail.fundamental.six_dimension_scores,
            quality_profile:
              sample.four_dimensions.detail.fundamental.six_dimension_scores.quality_profile.map(
                (row, i) => ({ ...row, score: i === 0 ? null : 0 })
              )
          }
        },
        sentiment: {
          ...sample.four_dimensions.detail.sentiment,
          daily_main_fund_flow: {
            ...sample.four_dimensions.detail.sentiment.daily_main_fund_flow,
            daily_values: [{ trade_date: '2026-09-23', net_flow: 0 }]
          }
        }
      }
    }
  };
  act(() => root.render(<StandardReport report={report} />));
  click('基本面详情与逻辑');
  expect(host.querySelector('.standard-quality-row')?.textContent).toContain('未评分');
  expect(host.querySelector('.standard-quality-row')?.querySelector('.standard-fill')).toBeNull();
  expect(host.querySelectorAll('.standard-quality-row')[1].textContent).toContain('0.0');
  click('关闭详情');
  click('情绪面详情与逻辑');
  expect(host.querySelector('.sentiment-selected')?.textContent).toContain('零值');
  expect(host.querySelector<HTMLElement>('.standard-flow-bar')?.style.height).toBe('0%');
  expect(host.innerHTML).not.toMatch(/NaN|Infinity/);
});
it('filters news and switches between the same bull/bear argument structure', () => {
  act(() => root.render(<StandardReport report={sample} />));
  click('新闻面详情与逻辑');
  click('评级');
  expect(host.querySelector('[role="dialog"]')?.textContent).toContain('该类别暂无事件');
  click('公告');
  expect(host.querySelector('[role="dialog"]')?.textContent).toContain('报告检索窗口内无新增公告');
  click('关闭详情');
  click('多空决议详情');
  click('多头论点 · 1');
  expect(host.querySelector('[role="dialog"]')?.textContent).toContain('Q2 单季盈利拐点已现');
  expect(host.querySelector('[role="dialog"]')?.textContent).toContain('成立理由');
  expect(host.querySelector('[role="dialog"]')?.textContent).not.toContain('反证与理由');
});
