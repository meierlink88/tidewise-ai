import type { MarketAnchor } from '@/models/market';

export const mockMarkets: MarketAnchor[] = [
  {
    id: 'market-001',
    name: '美元指数',
    value: '104.20',
    trend: '震荡偏强'
  },
  {
    id: 'market-002',
    name: '十年美债',
    value: '4.18%',
    trend: '利率中枢抬升'
  },
  {
    id: 'market-003',
    name: '恒生科技',
    value: '阶段修复',
    trend: '风险偏好改善'
  }
];
