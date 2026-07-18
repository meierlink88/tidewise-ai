import type { EventHighlight } from '@/models/event';

export const mockEvents: EventHighlight[] = [
  {
    id: 'event-001',
    title: '美联储官员释放利率路径观察信号',
    region: '美国',
    impact: '影响美元流动性与全球风险偏好',
    tags: ['宏观', '利率', '美元']
  },
  {
    id: 'event-002',
    title: '新能源产业链出现区域政策催化',
    region: '中国',
    impact: '关注上游材料与设备板块传导',
    tags: ['产业', '新能源', '政策']
  }
];
