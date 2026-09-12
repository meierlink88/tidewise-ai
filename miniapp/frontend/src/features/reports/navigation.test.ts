import { describe, expect, it, vi } from 'vitest';
import { reportDetailShare } from './share';
import { buildReportDetailURL, navigateToReportDetail, parseReportDetailRoute } from './navigation';

const reportId = 'RPT11111111-1111-4111-8111-111111111111';

describe('Report navigation', () => {
  it.each([
    ['industry_chain_analyses', 'industry-unit-ICHfcd316fe-9235-5e3e-862b-90f39a6fe5ba'],
    ['concept_analyses', 'industry-unit-CON62cfd327-6ea3-5cf7-af94-a5cade025030'],
    ['geopolitical_stories', 'geopolitics'],
    ['macroeconomic_stories', 'macro']
  ] as const)('preserves %s keys through navigation and sharing', (targetType, targetKey) => {
    const route = { reportId, targetType, targetKey };
    const navigateTo = vi.fn();
    navigateToReportDetail({ navigateTo }, route);
    const expectedURL = `/pages/report/detail/index?reportId=${reportId}&targetType=${targetType}&targetKey=${targetKey}`;
    expect(navigateTo).toHaveBeenCalledOnce();
    expect(navigateTo).toHaveBeenCalledWith({ url: expectedURL });
    expect(parseReportDetailRoute(route)).toEqual(route);
    expect(
      parseReportDetailRoute({
        ...route,
        targetKey: targetKey.replaceAll('I', '%49').replaceAll('C', '%43')
      })
    ).toEqual(route);
    const share = reportDetailShare(route);
    expect(share.path).toBe(expectedURL);
    expect(parseReportDetailRoute(Object.fromEntries(new URLSearchParams(share.query)))).toEqual(
      route
    );
  });

  it('accepts a case-sensitive local key at the 128-character boundary', () => {
    const route = {
      reportId,
      targetType: 'concept_analyses' as const,
      targetKey: `A${'b'.repeat(127)}`
    };
    expect(parseReportDetailRoute(route)).toEqual(route);
    expect(buildReportDetailURL(route)).toContain(`targetKey=${route.targetKey}`);
  });

  it.each(['', '_key', '.key', '-key', 'a/b', 'a b', 'a?b', 'a#b', 'a&b', 'A'.repeat(129)])(
    'rejects invalid local key %j before navigation',
    (targetKey) => {
      const route = { reportId, targetType: 'industry_chain_analyses' as const, targetKey };
      const navigateTo = vi.fn();
      expect(() => navigateToReportDetail({ navigateTo }, route)).toThrow('invalid Report route');
      expect(navigateTo).not.toHaveBeenCalled();
      expect(() => parseReportDetailRoute(route)).toThrow('invalid Report route');
    }
  );

  it.each(['concept_analyses', 'industry_chain_analyses'] as const)(
    'round-trips the %s detail route',
    (targetType) => {
      expect(
        buildReportDetailURL({
          reportId,
          targetType,
          targetKey: 'chn-21'
        })
      ).toBe(
        `/pages/report/detail/index?reportId=${reportId}&targetType=${targetType}&targetKey=chn-21`
      );
      expect(parseReportDetailRoute({ reportId, targetType, targetKey: 'chn-21' })).toEqual({
        reportId,
        targetType,
        targetKey: 'chn-21'
      });
    }
  );

  it('uses navigateTo and never derives a target from display copy', () => {
    const navigateTo = vi.fn();
    navigateToReportDetail(
      { navigateTo },
      { reportId, targetType: 'geopolitical_stories', targetKey: 'geopolitics' }
    );
    expect(navigateTo).toHaveBeenCalledWith({
      url: `/pages/report/detail/index?reportId=${reportId}&targetType=geopolitical_stories&targetKey=geopolitics`
    });
  });

  it('accepts only the validated internal parameters injected by the Taro router', () => {
    expect(
      parseReportDetailRoute({
        reportId,
        targetType: 'geopolitical_stories',
        targetKey: 'geopolitics',
        stamp: 'AA',
        $taroTimestamp: 1788265499968
      })
    ).toEqual({ reportId, targetType: 'geopolitical_stories', targetKey: 'geopolitics' });
    expect(() =>
      parseReportDetailRoute({
        reportId,
        targetType: 'geopolitical_stories',
        targetKey: 'geopolitics',
        stamp: '../AA'
      })
    ).toThrow('invalid Report route');
    expect(() =>
      parseReportDetailRoute({
        reportId,
        targetType: 'geopolitical_stories',
        targetKey: 'geopolitics',
        $taroTimestamp: '1788265499968'
      })
    ).toThrow('invalid Report route');
  });

  it('fails before a request when route parameters are missing, duplicated or illegal', () => {
    expect(() =>
      parseReportDetailRoute({
        reportId,
        targetType: 'geopolitical_stories',
        targetKey: '地缘政治'
      })
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
        targetType: 'geopolitical_stories',
        targetKey: 'geopolitics',
        title: '不允许从标题推导'
      })
    ).toThrow('invalid Report route');
    expect(() =>
      parseReportDetailRoute({
        reportId,
        targetType: 'concept_analyses',
        targetKey: `a${'b'.repeat(128)}`
      })
    ).toThrow('invalid Report route');
  });
});
