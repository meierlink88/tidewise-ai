import { describe, expect, it, vi } from 'vitest';
import reasoningTreeListFixture from '../../mocks/research-reasoning-trees/list.json';
import reasoningTreeDetailFixture from '../../mocks/research-reasoning-trees/detail.json';
import { createResearchThemeApiPort } from './api-port';

describe('research theme BFF adapter', () => {
  it('maps the V1 Miniapp BFF contract into Theme card data', async () => {
    const request = vi.fn().mockResolvedValue({
      statusCode: 200,
      data: {
        request_id: 'miniapp-theme-test',
        result: {
          window_start: '2026-07-27T08:00:00Z',
          window_end: '2026-07-28T08:00:00Z',
          as_of: '2026-07-28T09:05:00Z',
          theme_count: 1,
          event_count: 2,
          next_cursor: null,
          items: [
            {
              id: '11111111-1111-4111-8111-111111111111',
              analysis_batch_id: 'batch-1',
              title: '高速光模块需求验证',
              one_line_conclusion: '端口计划上调可能增强高速光模块需求预期',
              conclusion_direction: 'positive',
              impact_strength: 'medium',
              attention_level: 'high',
              conclusion_status: 'partial',
              transmission_stage: 'validation',
              investment_guidance_action: 'focus',
              investment_guidance_summary: '优先验证采购订单与光模块排产。',
              time_horizon_category: 'short_term',
              time_horizon_summary: null,
              transmission_summary: '交换机 → 高速光模块',
              checkpoint_summary: null,
              risk_summary: null,
              analysis_as_of: '2026-07-28T08:00:00Z',
              window_start: '2026-07-27T08:00:00Z',
              window_end: '2026-07-28T08:00:00Z',
              published_at: '2026-07-28T08:05:00Z',
              impacts: [
                {
                  chain_node_entity_id: '33333333-3333-4333-8333-333333333333',
                  name: '高速光模块',
                  relation_role: 'beneficiary',
                  impact_direction: 'positive',
                  impact_summary: '需求预期增强',
                  display_order: 1
                }
              ],
              evidence_event_count: 2,
              reasoning_tree_count: 1
            }
          ]
        }
      }
    });

    const feed = await createResearchThemeApiPort({
      baseUrl: 'https://miniapp.example.test',
      request
    }).list({ period: 'today', limit: 20 });

    expect(request).toHaveBeenCalledWith({
      url: 'https://miniapp.example.test/api/miniapp/v1/research/themes',
      method: 'GET',
      data: { period: 'today', limit: 20 },
      dataType: 'json',
      timeout: 10_000
    });
    expect(feed).toMatchObject({ themeCount: 1, eventCount: 2, nextCursor: null });
    expect(feed.items[0]).toMatchObject({
      title: '高速光模块需求验证',
      impactStrength: 'medium',
      transmissionSummary: '交换机 → 高速光模块',
      updateLabel: '1 小时前',
      impacts: [{ name: '高速光模块', displayOrder: 1 }],
      evidenceEventCount: 2,
      reasoningTreeCount: 1
    });
    expect(feed.items[0]).not.toHaveProperty('marketConfirmationSummary');
    expect(feed.items[0]).not.toHaveProperty('subjectEntityId');
  });

  it('fails closed on a BFF error', async () => {
    const request = vi.fn().mockResolvedValue({ statusCode: 503, data: { error: 'unavailable' } });
    await expect(
      createResearchThemeApiPort({
        baseUrl: 'https://miniapp.example.test/',
        request
      }).list({ period: 'today', limit: 20 })
    ).rejects.toThrow('503');
  });

  it('requests a history cursor page without the legacy rolling window', async () => {
    const request = vi.fn().mockResolvedValue({
      statusCode: 200,
      data: {
        request_id: 'history-page',
        result: {
          window_start: '2026-07-04T16:00:00Z',
          window_end: '2026-08-03T16:00:00Z',
          as_of: '2026-08-04T03:00:00Z',
          theme_count: 0,
          event_count: 0,
          items: [],
          next_cursor: null
        }
      }
    });

    await createResearchThemeApiPort({
      baseUrl: 'https://miniapp.example.test',
      request
    }).list({ period: 'history', limit: 5, cursor: 'opaque-cursor' });

    expect(request).toHaveBeenCalledWith(
      expect.objectContaining({
        data: { period: 'history', limit: 5, cursor: 'opaque-cursor' }
      })
    );
    expect(request.mock.calls[0][0].data).not.toHaveProperty('window_hours');
  });

  it('loads one Theme event timeline through the published detail contract', async () => {
    const request = vi.fn().mockResolvedValue({
      statusCode: 200,
      data: {
        request_id: 'miniapp-theme-detail-test',
        result: {
          ...reasoningTreeListFixture.result.theme,
          events: reasoningTreeDetailFixture.result.reasoning_tree.events
        }
      }
    });
    const themeId = reasoningTreeListFixture.result.theme.id;

    const detail = await createResearchThemeApiPort({
      baseUrl: 'https://miniapp.example.test',
      request
    }).getDetail(themeId);

    expect(request).toHaveBeenCalledWith({
      url: `https://miniapp.example.test/api/miniapp/v1/research/themes/${themeId}`,
      method: 'GET',
      data: {},
      dataType: 'json',
      timeout: 10_000
    });
    expect(detail).toEqual({
      id: themeId,
      title: '高速光模块需求验证',
      events: [
        {
          eventId: '99999999-9999-4999-8999-999999999999',
          title: '端口计划上调',
          summary: '云厂商端口计划上调 80%。',
          eventTime: expect.objectContaining({ status: 'confirmed' })
        },
        {
          eventId: 'aaaaaaaa-1111-4111-8111-111111111111',
          title: '采购尚未发生',
          summary: '当前尚未观察到正式采购。',
          eventTime: { status: 'pending' }
        }
      ]
    });
  });

  it('maps an expired Theme detail to a stable unavailable error', async () => {
    const request = vi.fn().mockResolvedValue({
      statusCode: 404,
      data: {
        request_id: 'miniapp-theme-detail-missing',
        error: {
          code: 'RESEARCH_THEME_NOT_FOUND',
          message: 'not found',
          details: {}
        }
      }
    });

    await expect(
      createResearchThemeApiPort({
        baseUrl: 'https://miniapp.example.test',
        request
      }).getDetail(reasoningTreeListFixture.result.theme.id)
    ).rejects.toMatchObject({ kind: 'themeUnavailable' });
  });

  it('cleans a detail transport failure into a stable service error', async () => {
    const request = vi.fn().mockRejectedValue(new Error('private upstream URL'));

    await expect(
      createResearchThemeApiPort({
        baseUrl: 'https://miniapp.example.test',
        request
      }).getDetail(reasoningTreeListFixture.result.theme.id)
    ).rejects.toMatchObject({ kind: 'serviceUnavailable', message: 'serviceUnavailable' });
  });
});
