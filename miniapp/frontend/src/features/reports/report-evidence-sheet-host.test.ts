import { createElement, type ReactNode } from 'react';
import { renderToStaticMarkup } from 'react-dom/server';
import { describe, expect, it, vi } from 'vitest';
import { mockReportPort } from '../../mocks/reports/mock-port';
import type { ReportEvidenceRoute } from './navigation';
import {
  ReportEvidenceSheetHost,
  ReportEvidenceSheetHostController
} from './report-evidence-sheet';

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
    createElement(
      'div',
      { className: props.className, 'aria-label': props.ariaLabel },
      props.children as ReactNode
    )
}));

const firstRoute: ReportEvidenceRoute = {
  reportId: 'RPT11111111-1111-4111-8111-111111111111',
  scopeToken: 'RPE11111111-1111-4111-8111-111111111111',
  title: '地缘政治证据'
};

describe('ReportEvidenceSheetHostController', () => {
  it('opens and closes the selected Evidence scope for its host subscriber', () => {
    const controller = new ReportEvidenceSheetHostController();
    const observedRoutes: Array<ReportEvidenceRoute | null> = [];
    const unsubscribe = controller.subscribe(() => observedRoutes.push(controller.snapshot()));

    controller.open(firstRoute);
    controller.close();
    unsubscribe();

    expect(observedRoutes).toEqual([firstRoute, null]);
  });

  it('replaces an open scope without retaining the earlier selection', () => {
    const controller = new ReportEvidenceSheetHostController();
    const listener = vi.fn();
    controller.subscribe(listener);
    const latestRoute: ReportEvidenceRoute = {
      ...firstRoute,
      scopeToken: 'RPE22222222-2222-4222-8222-222222222222',
      title: '宏观经济证据'
    };

    controller.open(firstRoute);
    controller.open(latestRoute);

    expect(controller.snapshot()).toBe(latestRoute);
    expect(listener).toHaveBeenCalledTimes(2);
  });

  it('renders and dismisses the bottom sheet from the controller state', () => {
    const controller = new ReportEvidenceSheetHostController();

    controller.open(firstRoute);
    const openHTML = renderToStaticMarkup(
      createElement(ReportEvidenceSheetHost, { controller, port: mockReportPort })
    );
    controller.close();
    const closedHTML = renderToStaticMarkup(
      createElement(ReportEvidenceSheetHost, { controller, port: mockReportPort })
    );

    expect(openHTML).toContain('report-evidence-sheet');
    expect(openHTML).toContain('正在读取相关证据');
    expect(closedHTML).toBe('');
  });
});
