import { describe, expect, it } from 'vitest';
import { createMockResearchThemeFeedPort } from '../../mocks/research-themes/mock-port';
import { filterHomeResearchThemes, getHomeThemeCategories } from './feed';

describe('research theme homepage feed', () => {
  it('provides the V1 Theme card content', async () => {
    const feed = await createMockResearchThemeFeedPort().list();

    expect(feed).toMatchObject({ themeCount: 1, eventCount: 2, trackingCount: 3 });
    expect(feed.items[0]).toMatchObject({
      title: '高速光模块需求验证',
      impactStrength: 'medium',
      transmissionStage: 'validation',
      evidenceEventCount: 2,
      reasoningTreeCount: 2
    });
    expect(feed.items[0].impacts.map((impact) => impact.name)).toEqual([
      '交换机',
      '高速光模块',
      'DSP 芯片'
    ]);
  });

  it('derives category tabs and searches Theme, summary, guidance, and impact nodes', async () => {
    const { items } = await createMockResearchThemeFeedPort().list();

    expect(getHomeThemeCategories(items)).toEqual(['全部']);
    expect(filterHomeResearchThemes(items, { category: '全部', query: 'DSP 芯片' })).toHaveLength(
      1
    );
    expect(filterHomeResearchThemes(items, { category: '全部', query: '采购订单' })).toHaveLength(
      1
    );
    expect(filterHomeResearchThemes(items, { category: '全部', query: '不存在' })).toEqual([]);
    expect(filterHomeResearchThemes(items, { category: '不存在的分类', query: '' })).toEqual([]);
  });
});
