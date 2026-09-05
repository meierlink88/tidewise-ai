import { describe, expect, it, vi } from 'vitest';
import type { ReportEvidenceRoute } from './navigation';
import { ReportEvidenceSheetHostController } from './report-evidence-sheet';

vi.mock('@tarojs/components', () => ({
  Button: 'button',
  Image: 'image',
  ScrollView: 'scroll-view',
  Text: 'text',
  View: 'view'
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
});
