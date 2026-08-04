import { describe, expect, it } from 'vitest';
import { createMockResearchThemeHomepagePort } from '../../mocks/research-themes/mock-port';
import { filterHomeResearchThemes } from './feed';

describe('research theme homepage feed', () => {
  it('provides the V1 Theme card content', async () => {
    const feed = await createMockResearchThemeHomepagePort().list({ period: 'today', limit: 20 });

    expect(feed).toMatchObject({ themeCount: 1, eventCount: 2 });
    expect(feed).not.toHaveProperty('trackingCount');
    expect(feed.items[0]).toMatchObject({
      title: '高速光模块需求验证',
      impactStrength: 'medium',
      transmissionStage: 'validation',
      evidenceEventCount: 2,
      reasoningTreeCount: 2
    });
    expect(feed.items[0]).not.toHaveProperty('categories');
    expect(feed.items[0].impacts.map((impact) => impact.name)).toEqual([
      '交换机',
      '高速光模块',
      'DSP 芯片'
    ]);
  });

  it('searches Theme, summary, guidance, and impact nodes without static categories', async () => {
    const { items } = await createMockResearchThemeHomepagePort().list({
      period: 'today',
      limit: 20
    });

    expect(filterHomeResearchThemes(items, 'DSP 芯片')).toHaveLength(1);
    expect(filterHomeResearchThemes(items, '采购订单')).toHaveLength(1);
    expect(filterHomeResearchThemes(items, '不存在')).toEqual([]);
  });

  it('keeps today themes out of history search results', async () => {
    const port = createMockResearchThemeHomepagePort();
    const today = await port.list({ period: 'today', limit: 20 });
    const history = await port.list({ period: 'history', limit: 5 });
    const todayStart = Date.parse(today.windowStart);
    const todayEnd = Date.parse(today.windowEnd);
    const historyStart = Date.parse(history.windowStart);
    const historyEnd = Date.parse(history.windowEnd);

    expect(todayEnd - todayStart).toBe(24 * 60 * 60 * 1000);
    expect(historyEnd).toBe(todayStart);
    expect(historyEnd - historyStart).toBe(30 * 24 * 60 * 60 * 1000);
    expect(
      today.items.every(
        (theme) =>
          Date.parse(theme.publishedAt) >= todayStart && Date.parse(theme.publishedAt) < todayEnd
      )
    ).toBe(true);
    expect(
      history.items.every(
        (theme) =>
          Date.parse(theme.publishedAt) >= historyStart &&
          Date.parse(theme.publishedAt) < historyEnd
      )
    ).toBe(true);
    expect(history.items.map((theme) => theme.id)).not.toContain(today.items[0].id);
    expect(filterHomeResearchThemes(history.items, today.items[0].title)).toEqual([]);
  });
});
