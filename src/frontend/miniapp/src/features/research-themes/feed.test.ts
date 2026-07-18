import { describe, expect, it } from 'vitest';
import { createMockResearchThemeFeedPort } from '../../mocks/research-themes/mock-port';
import { filterHomeResearchThemes, getHomeThemeCategories } from './feed';

describe('research theme homepage feed', () => {
  it('returns the approved homepage summary and three presentation-ready themes', async () => {
    const feed = await createMockResearchThemeFeedPort().list();

    expect(feed.themeCount).toBe(3);
    expect(feed.eventCount).toBe(13);
    expect(feed.trackingCount).toBe(17);
    expect(feed.items.map((item) => item.transmissionStage)).toEqual(['diffusion', 'validation', 'validation']);
    expect(feed.items.every((item) => typeof item.indexImpactSummary === 'string')).toBe(true);
  });

  it('maps the primary research theme into the approved card content', async () => {
    const feed = await createMockResearchThemeFeedPort().list();

    expect(feed.items[0]).toMatchObject({
      id: '11111111-1111-4111-8111-111111111111',
      name: '算力基建',
      oneLineConclusion: '算力资本开支推升，AI供应链涨价周期开启',
      impactLevel: 'high',
      transmissionPath: '北美云厂商资本开支维持高位 → 订单与交期同步扩张',
      nextCheckpoint: '尚未显现',
      tradingDirection: '关注高速互联与结构材料的订单和交期变化',
      supportingEventCount: 7,
      updateLabel: '1小时前更新',
      affectedChainNodes: [
        { id: '21111111-1111-4111-8111-111111111111', name: '光模块' },
        { id: '21111111-1111-4111-8111-222222222222', name: 'PCB/载板' },
        { id: '21111111-1111-4111-8111-333333333333', name: '结构材料' }
      ]
    });
  });

  it('derives stable category tabs from the feed without duplicating labels', async () => {
    const { items } = await createMockResearchThemeFeedPort().list();

    expect(getHomeThemeCategories(items)).toEqual([
      '全部',
      '算力基建',
      '地缘政治',
      '贸易管制',
      '货币政策'
    ]);
    expect(getHomeThemeCategories([...items].reverse())).toEqual([
      '全部',
      '算力基建',
      '地缘政治',
      '贸易管制',
      '货币政策'
    ]);
  });

  it('filters by category and searches card content plus affected nodes', async () => {
    const { items } = await createMockResearchThemeFeedPort().list();

    expect(filterHomeResearchThemes(items, { category: '地缘政治', query: '' }).map((item) => item.name)).toEqual([
      '贸易管制'
    ]);
    expect(filterHomeResearchThemes(items, { category: '全部', query: '结构材料' }).map((item) => item.name)).toEqual([
      '算力基建'
    ]);
    expect(filterHomeResearchThemes(items, { category: '贸易管制', query: '审批' }).map((item) => item.name)).toEqual([
      '贸易管制'
    ]);
    expect(filterHomeResearchThemes(items, { category: '全部', query: '定价权' }).map((item) => item.name)).toEqual([
      '贸易管制'
    ]);
    expect(filterHomeResearchThemes(items, { category: '全部', query: '   ' })).toHaveLength(3);
    expect(filterHomeResearchThemes(items, { category: '不存在的分类', query: '' })).toEqual([]);
    expect(filterHomeResearchThemes(items, { category: '货币政策', query: '光模块' })).toEqual([]);
  });
});
