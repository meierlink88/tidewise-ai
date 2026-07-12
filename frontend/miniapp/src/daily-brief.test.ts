import { describe, expect, it } from 'vitest';
import { mockDailyBrief } from './data/daily-brief/mock-daily-brief';
import { MockDailyBriefAdapter } from './services/daily-brief/mock-adapter';
import { mapDailyBriefToHome } from './templates/daily-brief';
import { createInitialResourceState, resourceStateReducer } from './models/resource-state';
import { getVisibleHomeSections, homeSectionRegistry } from './templates/home-sections';
import { getGraphComingSoonMessage } from './utils/coming-soon';

describe('mock-only daily brief contract', () => {
  it('uses the mock schema and only approved impact entity types', () => {
    expect(mockDailyBrief.schemaVersion).toBe('mock.daily-brief.v1');
    expect(mockDailyBrief.conclusions.flatMap((item) => item.impacts).map((item) => item.entityType)).toEqual(
      expect.arrayContaining(['sector', 'benchmark', 'commodity', 'economy', 'industry_chain'])
    );
  });

  it('contains no graph payload or stock recommendation fields', () => {
    const serialized = JSON.stringify(mockDailyBrief);

    expect(serialized).not.toMatch(/ReasoningGraph|ReasoningPathStep|graphId|nodes|edges/);
    expect(serialized).not.toMatch(/个股|股票|买入|卖出|目标价|北方稀土|英伟达/);
  });
});

describe('MockDailyBriefAdapter', () => {
  it('returns the ready fixture', async () => {
    const adapter = new MockDailyBriefAdapter('ready');
    await expect(adapter.getDailyBrief()).resolves.toEqual(mockDailyBrief);
  });

  it('returns null for the empty scenario', async () => {
    const adapter = new MockDailyBriefAdapter('empty');
    await expect(adapter.getDailyBrief()).resolves.toBeNull();
  });

  it('throws a displayable error for the error scenario', async () => {
    const adapter = new MockDailyBriefAdapter('error');
    await expect(adapter.getDailyBrief()).rejects.toThrow('今日观潮加载失败');
  });
});

describe('daily brief home mapper', () => {
  it('maps conclusions with their impacts, evidence and uncertainty', () => {
    const view = mapDailyBriefToHome(mockDailyBrief);
    expect(view.market.label).toBe('严重分化');
    expect(view.conclusions[0].impacts[0].uncertainty).toBeTruthy();
    expect(view.conclusions[0].evidence[0].source).toBe('商务部');
    expect(view.disclaimer).toContain('不构成投资建议');
  });

  it('maps missing evidence to an empty list', () => {
    const brief = { ...mockDailyBrief, conclusions: [{ ...mockDailyBrief.conclusions[0], evidence: [] }] };
    expect(mapDailyBriefToHome(brief).conclusions[0].evidence).toEqual([]);
  });
});

describe('resource state reducer', () => {
  it('moves through loading, ready, empty and error states', () => {
    const initial = createInitialResourceState<string>();
    const loading = resourceStateReducer(initial, { type: 'load' });
    const ready = resourceStateReducer(loading, { type: 'resolve', data: 'brief' });
    const empty = resourceStateReducer(loading, { type: 'resolve', data: null });
    const error = resourceStateReducer(loading, { type: 'reject', message: '失败' });

    expect(initial.status).toBe('idle');
    expect(loading.status).toBe('loading');
    expect(ready).toEqual({ status: 'ready', data: 'brief' });
    expect(empty).toEqual({ status: 'empty' });
    expect(error).toEqual({ status: 'error', message: '失败' });
  });
});

describe('home section registry', () => {
  it('keeps the canonical section order without fixture copy', () => {
    expect(homeSectionRegistry.map((section) => section.key)).toEqual([
      'brief-summary', 'themes', 'conclusions', 'impacts', 'evidence', 'safety-note'
    ]);
    expect(JSON.stringify(homeSectionRegistry)).not.toContain(mockDailyBrief.summary);
  });

  it('hides empty optional sections', () => {
    const view = mapDailyBriefToHome(mockDailyBrief);
    const visible = getVisibleHomeSections({ ...view, themes: [] });
    expect(visible.map((section) => section.key)).not.toContain('themes');
  });
});

describe('graph placeholder', () => {
  it('uses the approved non-navigating feedback copy', () => {
    expect(getGraphComingSoonMessage()).toBe('推导图谱即将开放');
  });
});
