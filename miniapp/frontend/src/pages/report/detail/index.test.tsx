import { afterEach, describe, expect, it, vi } from 'vitest';
import { normalizedMockReportPort } from '../../../mocks/reports/mock-port';
import { loadReportDetail } from './index';

vi.mock('@tarojs/taro', () => ({ default: {}, usePullDownRefresh: vi.fn() }));
vi.mock('@tarojs/components', () => ({
  View: 'view',
  Text: 'text',
  Button: 'button',
  Image: 'image',
  ScrollView: 'scroll-view'
}));
afterEach(() => vi.restoreAllMocks());
const reportId = 'RPT11111111-1111-4111-8111-111111111111';
describe('report detail loading', () => {
  it('reads the selected analysis using its exact report and unit key', async () => {
    const getAnalysis = vi.spyOn(normalizedMockReportPort, 'getAnalysis');
    const port = normalizedMockReportPort;
    const result = await loadReportDetail(port, {
      reportId,
      targetType: 'geopolitical_stories',
      targetKey: 'g1'
    });
    expect(getAnalysis).toHaveBeenCalledWith(reportId, 'geopolitical_stories', 'g1');
    expect(result.targetType).toBe('analysis');
    expect(result.detail.macro_impacts.length).toBeGreaterThan(0);
  });
  it('rejects missing and retired routes before calling an adapter', async () => {
    const getAnalysis = vi.spyOn(normalizedMockReportPort, 'getAnalysis');
    const port = normalizedMockReportPort;
    await expect(loadReportDetail(port, null)).rejects.toMatchObject({ kind: 'invalidRequest' });
    for (const targetType of ['layer', 'industry_chain'] as const)
      await expect(
        loadReportDetail(port, { reportId, targetType, targetKey: 'g1' })
      ).rejects.toMatchObject({ kind: 'invalidRequest' });
    expect(getAnalysis).not.toHaveBeenCalled();
  });
});
