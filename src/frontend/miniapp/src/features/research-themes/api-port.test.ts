import { describe, expect, it, vi } from 'vitest';
import { createResearchThemeApiPort } from './api-port';

describe('research theme BFF adapter', () => {
  it('maps the Miniapp BFF contract into homepage card data', async () => {
    const request = vi.fn().mockResolvedValue({
      statusCode: 200,
      data: {
        window_start: '2026-07-18T00:00:00Z',
        window_end: '2026-07-18T10:00:00Z',
        as_of: '2026-07-18T10:00:00Z',
        theme_count: 1,
        event_count: 2,
        next_cursor: null,
        items: [
          {
            id: '11111111-1111-5111-8111-111111111111',
            name: 'AI算力扩产与半导体',
            one_line_conclusion: '晶圆扩产增强但卡点与价格背离',
            impact_level: 'high',
            transmission_path: '资本开支 → 设备材料需求',
            trading_direction: '重点验证订单、交期和关键材料价格',
            transmission_stage: 'diffusion',
            next_checkpoint: '卡点尚未证明',
            index_impact_summary: '市场混合偏背离',
            published_at: '2026-07-18T09:00:00Z',
            affected_chain_nodes: [
              {
                id: '22222222-2222-5222-8222-222222222222',
                name: '半导体设备',
                relation_role: 'beneficiary',
                impact_summary: '订单仍待验证'
              }
            ],
            related_indices: [],
            supporting_event_count: 2,
            contradicting_event_count: 1
          }
        ]
      }
    });

    const feed = await createResearchThemeApiPort({ baseUrl: 'https://miniapp.example.test', request }).list();

    expect(request).toHaveBeenCalledWith({
      url: 'https://miniapp.example.test/api/v1/miniapp/research/themes',
      method: 'GET',
      data: { window_hours: 24, limit: 20 },
      dataType: 'json'
    });
    expect(feed).toMatchObject({ themeCount: 1, eventCount: 2, trackingCount: 17, nextCursor: null });
    expect(feed.items[0]).toMatchObject({
      name: 'AI算力扩产与半导体',
      tradingDirection: '重点验证订单、交期和关键材料价格',
      transmissionStage: 'diffusion',
      nextCheckpoint: '卡点尚未证明',
      updateLabel: '1小时前更新',
      categories: ['算力基建'],
      supportingEventCount: 2,
      affectedChainNodes: [{ name: '半导体设备', relationRole: 'beneficiary', impactSummary: '订单仍待验证' }]
    });
    expect(feed.items[0]).not.toHaveProperty('transmissionPhaseLabel');
    expect(feed.items[0]).not.toHaveProperty('hasMoreDetail');
  });

  it('fails closed on a BFF error instead of returning mock data', async () => {
    const request = vi.fn().mockResolvedValue({ statusCode: 503, data: { error: 'unavailable' } });
    const port = createResearchThemeApiPort({ baseUrl: 'https://miniapp.example.test/', request });

    await expect(port.list()).rejects.toThrow('503');
  });
});
