// @vitest-environment jsdom
import { act, createElement, type ReactNode } from 'react';
import { createRoot, type Root } from 'react-dom/client';
import { beforeEach, afterEach, describe, expect, it, vi } from 'vitest';
import { normalizedMockReportPort } from '../mocks/reports/mock-port';
import { parseAnalysisDetail } from '../features/reports/normalized-contract';
import { UnifiedDetailView } from './report/detail/unified-detail';
import unified from '../mocks/reports/unified-v6.json';

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

describe('unified report interactions', () => {
  it('switches whole reasonings and assets locally while keeping evidence scoped', () => {
    const detail = parseAnalysisDetail(unified.details['geopolitical_stories/g1'], 'g1');
    const evidence = vi.fn();
    act(() =>
      root.render(createElement(UnifiedDetailView, { detail, reportId, onEvidence: evidence }))
    );
    expect(host.querySelector('.unified-mechanism')?.textContent).toContain(
      detail.reasonings![0].reasoning_summary.logic
    );
    expect(host.querySelector('.unified-metric-value')?.textContent).toBe('+9.5%');
    expect(host.querySelector('.unified-period')?.textContent).toContain('近一周');
    expect(host.querySelector('.unified-asset-detail')?.textContent).toContain('↑4%');
    click(host.querySelector('.unified-evidence'));
    expect(evidence).toHaveBeenCalledWith(
      expect.objectContaining({ reportId, scopeToken: 'RPE11111111-1111-4111-8111-111111111111' })
    );
    const second = detail.reasonings![1];
    click(
      Array.from(host.querySelectorAll('button')).find((b) => b.textContent === second.title) ??
        null
    );
    expect(host.querySelector('.unified-mechanism')?.textContent).toContain(
      second.reasoning_summary.logic
    );
    expect(host.querySelector('.unified-asset-detail')?.textContent).toContain(
      second.affected_assets[0].name
    );
    if (second.affected_assets.length > 1) {
      click(host.querySelectorAll('.unified-asset')[1]);
      expect(host.querySelector('.unified-asset-header')?.textContent).toContain(
        second.affected_assets[1].name
      );
    }
  });
  it('does not invent a zero allocation and hides tabs for a single reasoning', () => {
    const detail = parseAnalysisDetail(unified.details['geopolitical_stories/g1'], 'g1');
    detail.reasonings = [detail.reasonings![0]];
    delete detail.reasonings[0].affected_assets[0].assessment.weight_delta_pp;
    act(() =>
      root.render(createElement(UnifiedDetailView, { detail, reportId, onEvidence: vi.fn() }))
    );
    expect(host.querySelector('.unified-tab')).toBeNull();
    expect(host.querySelector('.unified-delta')?.textContent).not.toContain('0%');
    expect(host.querySelector('.normalized-publication')).toBeNull();
  });
});
