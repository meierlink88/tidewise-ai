import { describe, expect, it } from 'vitest';
import { createMockResearchThemeHomepagePort } from '../../mocks/research-themes/mock-port';
import { filterHomeResearchThemes } from './feed';

describe('research theme homepage feed', () => {
  it('provides the V1 Theme card content', async () => {
    const feed = await createMockResearchThemeHomepagePort().list();

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
    const { items } = await createMockResearchThemeHomepagePort().list();

    expect(filterHomeResearchThemes(items, 'DSP 芯片')).toHaveLength(1);
    expect(filterHomeResearchThemes(items, '采购订单')).toHaveLength(1);
    expect(filterHomeResearchThemes(items, '不存在')).toEqual([]);
  });
});
