import type { SubscriptionTopic } from '@/models/subscription';

export const mockSubscriptions: SubscriptionTopic[] = [
  {
    id: 'sub-001',
    name: '全球央行政策',
    description: '跟踪主要央行利率、流动性和监管事件'
  },
  {
    id: 'sub-002',
    name: 'AI 算力链',
    description: '跟踪芯片、服务器、光模块和云资本开支变化'
  }
];
