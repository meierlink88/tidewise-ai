import type { HomeResearchThemeFeed, ResearchThemeFeedPort } from '../../features/research-themes/contract';

export const mockResearchThemeFeed: HomeResearchThemeFeed = {
  windowStart: '2026-07-06T09:00:00+08:00',
  windowEnd: '2026-07-07T09:00:00+08:00',
  asOf: '2026-07-07T09:00:00+08:00',
  themeCount: 3,
  eventCount: 13,
  trackingCount: 17,
  nextCursor: null,
  items: [
    {
      id: '11111111-1111-4111-8111-111111111111',
      name: '算力基建',
      oneLineConclusion: '算力资本开支推升，AI供应链涨价周期开启',
      impactLevel: 'high',
      transmissionPath: '北美云厂商资本开支维持高位 → 订单与交期同步扩张',
      tradingDirection: '关注高速互联与结构材料的订单和交期变化',
      transmissionStage: 'diffusion',
      nextCheckpoint: '尚未显现',
      marketConfirmationSummary: '算力产业链订单、交期与价格信号同步走强',
      publishedAt: '2026-07-07T08:00:00+08:00',
      updateLabel: '1小时前更新',
      categories: ['算力基建'],
      affectedChainNodes: [
        {
          id: '21111111-1111-4111-8111-111111111111',
          name: '光模块',
          relationRole: 'beneficiary',
          impactSummary: '订单增加，交期拉长'
        },
        {
          id: '21111111-1111-4111-8111-222222222222',
          name: 'PCB/载板',
          relationRole: 'beneficiary',
          impactSummary: '缺口扩大，价格上移'
        },
        {
          id: '21111111-1111-4111-8111-333333333333',
          name: '结构材料',
          relationRole: 'exposure',
          impactSummary: '等待供给弹性和成本传导验证'
        }
      ],
      relatedIndices: [],
      supportingEventCount: 7,
      contradictingEventCount: 0,
    },
    {
      id: '22222222-2222-4222-8222-222222222222',
      name: '贸易管制',
      oneLineConclusion: '出口管制升级，稀土定价中枢加速上移',
      impactLevel: 'high',
      transmissionPath: '重稀土出口审批暂停 → 审批收紧且开工率低位',
      tradingDirection: '关注供给收紧形成的定价权变化',
      transmissionStage: 'validation',
      nextCheckpoint: '尚未显现',
      marketConfirmationSummary: '稀土价格中枢上移，下游成本压力增加',
      publishedAt: '2026-07-06T22:00:00+08:00',
      updateLabel: '昨日 22:00更新',
      categories: ['地缘政治', '贸易管制'],
      affectedChainNodes: [
        {
          id: '22222222-2222-4222-8222-111111111111',
          name: '供给收紧',
          relationRole: 'constraint',
          impactSummary: '库存消耗，议价权上移'
        },
        {
          id: '22222222-2222-4222-8222-222222222222',
          name: '定价上移',
          relationRole: 'beneficiary',
          impactSummary: '价格发现，利润重分配'
        },
        {
          id: '22222222-2222-4222-8222-333333333333',
          name: '下游替代',
          relationRole: 'exposure',
          impactSummary: '观察替代供给与库存策略'
        }
      ],
      relatedIndices: [],
      supportingEventCount: 4,
      contradictingEventCount: 0,
    },
    {
      id: '33333333-3333-4333-8333-333333333333',
      name: '货币政策',
      oneLineConclusion: '美元维持高位，大宗与无息资产承压',
      impactLevel: 'focus',
      transmissionPath: '美元指数站上 101 → 实际利率高位且美元走强',
      tradingDirection: '关注美元高位下的大宗定价与资产分化',
      transmissionStage: 'validation',
      nextCheckpoint: '尚未显现',
      marketConfirmationSummary: '美元与实际利率高位压制大宗及无息资产估值',
      publishedAt: '2026-07-06T21:00:00+08:00',
      updateLabel: '昨日 21:00更新',
      categories: ['货币政策'],
      affectedChainNodes: [
        {
          id: '23333333-3333-4333-8333-111111111111',
          name: '实际利率',
          relationRole: 'driver',
          impactSummary: '融资成本与持有成本维持高位'
        },
        {
          id: '23333333-3333-4333-8333-222222222222',
          name: '大宗定价',
          relationRole: 'constraint',
          impactSummary: '计价压制与需求分化'
        },
        {
          id: '23333333-3333-4333-8333-333333333333',
          name: '资产分化',
          relationRole: 'exposure',
          impactSummary: '等待区域需求对冲验证'
        }
      ],
      relatedIndices: [],
      supportingEventCount: 2,
      contradictingEventCount: 0,
    }
  ]
};

export function createMockResearchThemeFeedPort(): ResearchThemeFeedPort {
  return {
    async list() {
      return mockResearchThemeFeed;
    }
  };
}
