import { describe, expect, it, vi } from 'vitest';
import { buildReportDetailURL, navigateToReportDetail, parseReportDetailRoute } from './navigation';

const reportId = 'RPT11111111-1111-4111-8111-111111111111';

describe('Report navigation', () => {
  it('builds one encoded registered detail-page URL from stable references', () => {
    expect(
      buildReportDetailURL({
        reportId,
        targetType: 'industry_chain',
        targetKey: 'chn-21'
      })
    ).toBe(
      `/pages/report/detail/index?reportId=${reportId}&targetType=industry_chain&targetKey=chn-21`
    );
  });

  it('uses navigateTo and never derives a target from display copy', () => {
    const navigateTo = vi.fn();
    navigateToReportDetail(
      { navigateTo },
      { reportId, targetType: 'layer', targetKey: 'geopolitics' }
    );
    expect(navigateTo).toHaveBeenCalledWith({
      url: `/pages/report/detail/index?reportId=${reportId}&targetType=layer&targetKey=geopolitics`
    });
  });

  it('accepts only the validated internal parameters injected by the Taro router', () => {
    expect(
      parseReportDetailRoute({
        reportId,
        targetType: 'layer',
        targetKey: 'geopolitics',
        stamp: 'AA',
        $taroTimestamp: 1788265499968
      })
    ).toEqual({ reportId, targetType: 'layer', targetKey: 'geopolitics' });
    expect(() =>
      parseReportDetailRoute({
        reportId,
        targetType: 'layer',
        targetKey: 'geopolitics',
        stamp: '../AA'
      })
    ).toThrow('invalid Report route');
    expect(() =>
      parseReportDetailRoute({
        reportId,
        targetType: 'layer',
        targetKey: 'geopolitics',
        $taroTimestamp: '1788265499968'
      })
    ).toThrow('invalid Report route');
  });

  it('fails before a request when route parameters are missing, duplicated or illegal', () => {
    expect(() =>
      parseReportDetailRoute({ reportId, targetType: 'layer', targetKey: '地缘政治' })
    ).toThrow('invalid Report route');
    expect(() =>
      parseReportDetailRoute({
        reportId,
        targetType: ['layer', 'industry_chain'],
        targetKey: 'geopolitics'
      })
    ).toThrow('invalid Report route');
    expect(() =>
      parseReportDetailRoute({
        reportId,
        targetType: 'layer',
        targetKey: 'geopolitics',
        title: '不允许从标题推导'
      })
    ).toThrow('invalid Report route');
    expect(() =>
      parseReportDetailRoute({
        reportId,
        targetType: 'industry_chain',
        targetKey: `a${'b'.repeat(128)}`
      })
    ).toThrow('invalid Report route');
  });
});
