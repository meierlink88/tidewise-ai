import { DAILY_BRIEF_SCHEMA_VERSION, type DailyBriefV1 } from '../../contracts/daily-brief-v1';

export const mockDailyBrief: DailyBriefV1 = {
  schemaVersion: DAILY_BRIEF_SCHEMA_VERSION,
  id: 'daily-brief-2026-07-12',
  asOf: '2026-07-12T09:00:00+08:00',
  displayDate: '07.12 周日',
  updatedAt: '09:00 数据更新',
  title: '今日观潮',
  summary: '全球贸易约束与美元定价继续分化，资源供给、算力基建和利率路径构成今日三条主要观察线索。',
  market: { label: '严重分化', direction: 'divergent', hint: '资源链与美元链反向运行' },
  sentiment: { label: '偏谨慎', direction: 'neutral', hint: '等待通胀与贸易政策进一步确认' },
  themes: ['地缘政治', '贸易管制', '货币政策'],
  conclusions: [
    {
      id: 'rare-earth-supply',
      badge: '主线一 · 资源供给',
      title: '出口约束推动稀土定价链重新评估',
      summary: '许可审批收紧与海外库存偏低共同抬高供给溢价，短期关注现货定价，中期关注替代与库存策略。',
      direction: 'up',
      confidence: 'high',
      graphAvailability: 'coming_soon',
      impacts: [
        { id: 'impact-rare-earth-sector', entityId: 'sector-rare-earth', entityType: 'sector', entityName: '稀土产业链', direction: 'up', horizon: 'short', strength: 'high', rationale: '出口供给弹性下降，海外现货溢价抬升', uncertainty: '政策执行节奏与库存释放可能改变短期强度' },
        { id: 'impact-dysprosium', entityId: 'commodity-dysprosium', entityType: 'commodity', entityName: '氧化镝现货', direction: 'up', horizon: 'short', strength: 'medium', rationale: '进口方补库与可贸易供给收缩', uncertainty: '报价与真实成交量可能存在偏差' }
      ],
      evidence: [
        { id: 'evidence-commerce', source: '商务部', title: '出口许可审批政策更新', summary: '相关关键品种出口许可审核趋严，供给预期收紧。', publishedAt: '2026-07-11', observedAt: '2026-07-12', confidence: 'high' },
        { id: 'evidence-industry', source: '行业周报', title: '海外稀土现货报价续升', summary: '欧洲市场氧化镝报价周环比上涨，进口端补库意愿增强。', publishedAt: '2026-07-11', observedAt: '2026-07-12', confidence: 'medium' }
      ]
    },
    {
      id: 'usd-pricing',
      badge: '主线二 · 宏观定价',
      title: '美元与长端利率压制非美风险偏好',
      summary: '降息预期后移使美元与美债收益率维持高位，大宗商品与新兴市场资产的定价压力继续分化。',
      direction: 'down',
      confidence: 'medium',
      graphAvailability: 'coming_soon',
      impacts: [
        { id: 'impact-dxy', entityId: 'benchmark-dxy', entityType: 'benchmark', entityName: '美元指数', direction: 'up', horizon: 'short', strength: 'medium', rationale: '利差和避险需求支撑美元', uncertainty: '通胀数据可能快速改变降息路径' },
        { id: 'impact-em', entityId: 'market-em', entityType: 'market', entityName: '新兴市场风险偏好', direction: 'down', horizon: 'short', strength: 'medium', rationale: '融资成本和资本流动压力上升', uncertainty: '各经济体政策缓冲能力不同' }
      ],
      evidence: [
        { id: 'evidence-fed', source: '美联储', title: '官员重申通胀仍需观察', summary: '多位官员强调需要更多通胀回落证据。', publishedAt: '2026-07-11', observedAt: '2026-07-12', confidence: 'high' }
      ]
    },
    {
      id: 'ai-infrastructure',
      badge: '主线三 · 产业需求',
      title: '算力资本开支向互联与散热环节扩散',
      summary: '云端资本开支保持韧性，产业链关注点由核心芯片继续向高速互联、液冷和电源等环节扩散。',
      direction: 'up',
      confidence: 'medium',
      graphAvailability: 'coming_soon',
      impacts: [
        { id: 'impact-ai-chain', entityId: 'industry-chain-ai', entityType: 'industry_chain', entityName: '算力基础设施链', direction: 'up', horizon: 'medium', strength: 'medium', rationale: '资本开支与出口需求延续', uncertainty: '产能释放和库存周期可能错位' },
        { id: 'impact-us-economy', entityId: 'economy-us', entityType: 'economy', entityName: '美国数字经济投资', direction: 'up', horizon: 'medium', strength: 'low', rationale: '云厂商继续扩张基础设施', uncertainty: '融资成本可能限制后续增速' }
      ],
      evidence: [
        { id: 'evidence-customs', source: '海关数据', title: '高速互联产品出口保持增长', summary: '相关产品出口额同比保持较高增速。', publishedAt: '2026-07-10', observedAt: '2026-07-12', confidence: 'medium' }
      ]
    }
  ],
  disclaimer: 'AI 分析仅供市场理解与决策辅助，不构成投资建议'
};
