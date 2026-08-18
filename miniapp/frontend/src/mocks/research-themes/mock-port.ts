import {
  ResearchThemeDetailError,
  type HomeResearchThemeFeed,
  type ResearchThemeDetail,
  type ResearchThemeHomepagePort
} from '../../features/research-themes/contract';
import { formatResearchUpdateLabel } from '../../features/research-themes/presentation';

const shanghaiOffsetMilliseconds = 8 * 60 * 60 * 1000;
const dayMilliseconds = 24 * 60 * 60 * 1000;
const mockNow = new Date();
const shiftedNow = new Date(mockNow.getTime() + shanghaiOffsetMilliseconds);
const todayStartMilliseconds =
  Date.UTC(shiftedNow.getUTCFullYear(), shiftedNow.getUTCMonth(), shiftedNow.getUTCDate()) -
  shanghaiOffsetMilliseconds;
const todayEndMilliseconds = todayStartMilliseconds + dayMilliseconds;
const historyStartMilliseconds = todayStartMilliseconds - 30 * dayMilliseconds;
const mockAsOf = mockNow.toISOString();
const todayPublishedAt = new Date(
  Math.max(todayStartMilliseconds, mockNow.getTime() - 30 * 1000)
).toISOString();
const historyPublishedAt = new Date(todayStartMilliseconds - 22 * 60 * 60 * 1000).toISOString();

export const mockResearchThemeFeed: HomeResearchThemeFeed = {
  windowStart: new Date(todayStartMilliseconds).toISOString(),
  windowEnd: new Date(todayEndMilliseconds).toISOString(),
  asOf: mockAsOf,
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
      analysisAsOf: mockAsOf,
      windowStart: new Date(todayStartMilliseconds).toISOString(),
      windowEnd: new Date(todayEndMilliseconds).toISOString(),
      publishedAt: todayPublishedAt,
      updateLabel: formatResearchUpdateLabel(todayPublishedAt, mockAsOf),
      impacts: [
        {
          nodeKey: '22222222-2222-4222-8222-222222222222',
          displayName: '交换机',
          chainNodeId: '22222222-2222-4222-8222-222222222222',
          name: '交换机',
          relationRole: 'beneficiary',
          impactDirection: 'positive',
          impactSummary: '端口计划增加可能提高交换机需求。',
          displayOrder: 1
        },
        {
          nodeKey: '33333333-3333-4333-8333-333333333333',
          displayName: '高速光模块',
          chainNodeId: '33333333-3333-4333-8333-333333333333',
          name: '高速光模块',
          relationRole: 'beneficiary',
          impactDirection: 'positive',
          impactSummary: '端口配置增加可能提高模块需求。',
          displayOrder: 2
        },
        {
          nodeKey: '44444444-4444-4444-8444-444444444444',
          displayName: 'DSP 芯片',
          chainNodeId: '44444444-4444-4444-8444-444444444444',
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

export const mockHistoricalResearchThemeFeed: HomeResearchThemeFeed = {
  windowStart: new Date(historyStartMilliseconds).toISOString(),
  windowEnd: new Date(todayStartMilliseconds).toISOString(),
  asOf: mockAsOf,
  themeCount: 1,
  eventCount: 1,
  nextCursor: null,
  items: [
    {
      id: '55555555-5555-4555-8555-555555555555',
      analysisBatchId: 'history-theme-memory-v1',
      title: '存储芯片价格修复观察',
      oneLineConclusion: '渠道报价回升，但库存去化与终端补库仍需继续验证',
      conclusionDirection: 'mixed',
      impactStrength: 'medium',
      attentionLevel: 'medium',
      conclusionStatus: 'partial',
      transmissionStage: 'validation',
      investmentGuidanceAction: 'observe',
      investmentGuidanceSummary: '观察存储原厂报价、渠道库存和终端补库节奏。',
      timeHorizonCategory: 'short_term',
      timeHorizonSummary: '未来一个季度',
      transmissionSummary: '报价回升 → 渠道库存去化 → 原厂稼动率修复',
      checkpointSummary: '合约价、渠道库存周转天数与终端订单。',
      riskSummary: '终端需求不足可能令价格修复持续性低于预期。',
      analysisAsOf: new Date(todayStartMilliseconds - 23 * 60 * 60 * 1000).toISOString(),
      windowStart: new Date(historyStartMilliseconds).toISOString(),
      windowEnd: new Date(todayStartMilliseconds).toISOString(),
      publishedAt: historyPublishedAt,
      updateLabel: formatResearchUpdateLabel(historyPublishedAt, mockAsOf),
      impacts: [
        {
          nodeKey: '66666666-6666-4666-8666-666666666666',
          displayName: '存储芯片',
          chainNodeId: '66666666-6666-4666-8666-666666666666',
          name: '存储芯片',
          relationRole: 'beneficiary',
          impactDirection: 'mixed',
          impactSummary: '报价改善利好盈利修复，但补库强度尚不确定。',
          displayOrder: 1
        }
      ],
      evidenceEventCount: 1,
      reasoningTreeCount: 1
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

export const mockHistoricalResearchThemeDetail: ResearchThemeDetail = {
  id: mockHistoricalResearchThemeFeed.items[0].id,
  title: mockHistoricalResearchThemeFeed.items[0].title,
  events: [
    {
      eventId: '77777777-7777-4777-8777-777777777777',
      title: '渠道报价回升',
      summary: '部分存储产品渠道报价较前期低点回升。',
      eventTime: { status: 'confirmed', date: '08-03', time: '10:00' }
    }
  ]
};

export function createMockResearchThemeHomepagePort(): ResearchThemeHomepagePort {
  return {
    async list(request) {
      return request.period === 'history' ? mockHistoricalResearchThemeFeed : mockResearchThemeFeed;
    },
    async getDetail(themeId) {
      if (themeId === mockResearchThemeDetail.id) return mockResearchThemeDetail;
      if (themeId === mockHistoricalResearchThemeDetail.id)
        return mockHistoricalResearchThemeDetail;
      throw new ResearchThemeDetailError('themeUnavailable');
    }
  };
}
