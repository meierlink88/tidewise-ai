import { describe, expect, it, vi } from 'vitest';
import {
  buildReportDetailURL,
  buildReportEvidenceURL,
  navigateToReportDetail,
  parseReportDetailRoute,
  parseReportEvidenceRoute
} from './navigation';

const reportId = 'RPT11111111-1111-4111-8111-111111111111';

describe('Report navigation', () => {
  it('builds encoded registered-page URLs from stable references', () => {
    expect(
      buildReportDetailURL({
        reportId,
        targetType: 'industry_chain',
        targetKey: 'chn-21'
      })
    ).toBe(
      `/pages/report/detail/index?reportId=${reportId}&targetType=industry_chain&targetKey=chn-21`
    );
    expect(
      buildReportEvidenceURL({
        reportId,
        scopeType: 'industry_chain_node',
        scopeKey: 'chn-21-n01',
        title: '油品运输服务'
      })
    ).toContain('scopeKey=chn-21-n01&title=%E6%B2%B9%E5%93%81');
  });

  it('safely truncates a full business name only for the Evidence route title', () => {
    const longTitle = `${'证'.repeat(78)}😀完整业务名称证据`;
    const url = buildReportEvidenceURL({
      reportId,
      scopeType: 'industry_chain_node',
      scopeKey: 'chn-21-n01',
      title: longTitle
    });
    const routedTitle = new URLSearchParams(url.split('?')[1]).get('title');

    expect(routedTitle).not.toBeNull();
    expect(Array.from(routedTitle ?? '')).toHaveLength(80);
    expect(routedTitle).toBe(`${'证'.repeat(78)}😀…`);
    expect(routedTitle).not.toContain('�');

    const maximumBusinessNameURL = buildReportEvidenceURL({
      reportId,
      scopeType: 'anchor',
      scopeKey: 'geo-a01',
      title: `${'名'.repeat(10_000)}证据`
    });
    expect(
      Array.from(new URLSearchParams(maximumBusinessNameURL.split('?')[1]).get('title') ?? '')
    ).toHaveLength(80);
  });

  it('normalizes one H5 percent-encoded Evidence title before validation', () => {
    expect(
      parseReportEvidenceRoute({
        reportId,
        scopeType: 'industry_chain_node',
        scopeKey: 'chn-21-n01',
        title: encodeURIComponent('油品运输服务')
      })
    ).toEqual({
      reportId,
      scopeType: 'industry_chain_node',
      scopeKey: 'chn-21-n01',
      title: '油品运输服务'
    });
  });

  it('preserves an already-decoded Weapp or TT title with a literal percent sign', () => {
    expect(
      parseReportEvidenceRoute({
        reportId,
        scopeType: 'anchor',
        scopeKey: 'macro-a01',
        title: '增长50%证据'
      }, false).title
    ).toBe('增长50%证据');
    expect(
      parseReportEvidenceRoute({
        reportId,
        scopeType: 'anchor',
        scopeKey: 'macro-a01',
        title: '增长%20证据'
      }, false).title
    ).toBe('增长%20证据');
  });

  it('decodes one H5 layer while preserving a literal percent escape in the title', () => {
    expect(
      parseReportEvidenceRoute({
        reportId,
        scopeType: 'anchor',
        scopeKey: 'macro-a01',
        title: encodeURIComponent('增长%20证据')
      }).title
    ).toBe('增长%20证据');
  });

  it('rejects malformed and overlong encoded route values', () => {
    const route = {
      reportId,
      scopeType: 'industry_chain_node',
      scopeKey: 'chn-21-n01'
    } as const;

    expect(() =>
      parseReportEvidenceRoute({ ...route, title: '%E6%B2%ZZ' })
    ).toThrow('invalid Report route');
    expect(() =>
      parseReportEvidenceRoute({
        ...route,
        title: encodeURIComponent('证'.repeat(81))
      })
    ).toThrow('invalid Report route');
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
    expect(
      parseReportEvidenceRoute({
        reportId,
        scopeType: 'anchor',
        scopeKey: 'geo-a1',
        title: '直接影响',
        stamp: 'AbZ',
        $taroTimestamp: 1788265499968
      })
    ).toEqual({
      reportId,
      scopeType: 'anchor',
      scopeKey: 'geo-a1',
      title: '直接影响'
    });
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
      parseReportEvidenceRoute({
        reportId,
        scopeType: 'event',
        scopeKey: 'chn-21',
        title: '相关事件'
      })
    ).toThrow('invalid Report route');
    expect(() =>
      parseReportEvidenceRoute({
        reportId,
        scopeType: 'anchor',
        scopeKey: 'geo-a01',
        title: '证'.repeat(81)
      })
    ).toThrow('invalid Report route');
    expect(() =>
      parseReportEvidenceRoute({
        reportId,
        scopeType: 'industry_chain_node',
        scopeKey: 'chn-21/chn-21-n01',
        title: '相关证据'
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
