import {
  ResearchThemeDetailError,
  type HomeResearchThemeFeed,
  type ResearchThemeDetail,
  type ResearchThemeHomepagePort
} from '../../features/research-themes/contract';

export const mockResearchThemeFeed: HomeResearchThemeFeed = {
  windowStart: '2026-07-27T08:00:00Z',
  windowEnd: '2026-07-28T08:00:00Z',
  asOf: '2026-07-28T08:00:00Z',
  themeCount: 1,
  eventCount: 2,
  nextCursor: null,
  items: [
    {
      id: '11111111-1111-4111-8111-111111111111',
      analysisBatchId: '20260728T-theme-reason-tree-v1',
      title: '高速光模块需求验证',
      oneLineConclusion: '云厂商端口计划上调，可能依次增强高速光模块与 DSP 芯片需求预期',
      conclusionDirection: 'positive',
      impactStrength: 'medium',
      attentionLevel: 'high',
      conclusionStatus: 'partial',
      transmissionStage: 'validation',
      investmentGuidanceAction: 'focus',
      investmentGuidanceSummary:
        '关注高速互联产业链，优先验证采购订单、可插拔技术路线及光模块排产。',
      timeHorizonCategory: 'short_term',
      timeHorizonSummary: '未来一个季度',
      transmissionSummary: '端口计划 +80% → 数据中心交换机 → 高速光模块 → DSP 芯片',
      checkpointSummary: '采购数量、单端口模块用量、光模块排产与 DSP 渗透率。',
      riskSummary: '采购未落地或替代技术路线可能削弱传导。',
      analysisAsOf: '2026-07-28T08:00:00Z',
      windowStart: '2026-07-27T08:00:00Z',
      windowEnd: '2026-07-28T08:00:00Z',
      publishedAt: '2026-07-28T08:05:00Z',
      updateLabel: '刚刚更新',
      impacts: [
        {
          chainNodeEntityId: '22222222-2222-4222-8222-222222222222',
          name: '交换机',
          relationRole: 'beneficiary',
          impactDirection: 'positive',
          impactSummary: '端口计划增加可能提高交换机需求。',
          displayOrder: 1
        },
        {
          chainNodeEntityId: '33333333-3333-4333-8333-333333333333',
          name: '高速光模块',
          relationRole: 'beneficiary',
          impactDirection: 'positive',
          impactSummary: '端口配置增加可能提高模块需求。',
          displayOrder: 2
        },
        {
          chainNodeEntityId: '44444444-4444-4444-8444-444444444444',
          name: 'DSP 芯片',
          relationRole: 'beneficiary',
          impactDirection: 'positive',
          impactSummary: '模块排产增加可能提高备料需求。',
          displayOrder: 3
        }
      ],
      evidenceEventCount: 2,
      reasoningTreeCount: 2
    }
  ]
};

export const mockResearchThemeDetail: ResearchThemeDetail = {
  id: mockResearchThemeFeed.items[0].id,
  title: mockResearchThemeFeed.items[0].title,
  events: [
    {
      eventId: '99999999-9999-4999-8999-999999999999',
      title: '端口计划上调',
      summary: '云厂商端口计划上调 80%。',
      eventTime: { status: 'confirmed', date: '07-28', time: '14:00' }
    },
    {
      eventId: 'aaaaaaaa-1111-4111-8111-111111111111',
      title: '采购尚未发生',
      summary: '当前尚未观察到正式采购。',
      eventTime: { status: 'pending' }
    }
  ]
};

export function createMockResearchThemeHomepagePort(): ResearchThemeHomepagePort {
  return {
    async list() {
      return mockResearchThemeFeed;
    },
    async getDetail(themeId) {
      if (themeId !== mockResearchThemeDetail.id) {
        throw new ResearchThemeDetailError('themeUnavailable');
      }
      return mockResearchThemeDetail;
    }
  };
}
