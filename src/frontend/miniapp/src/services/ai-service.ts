import type { AiMessage } from '@/models/ai-message';
import { request } from './request';

export function getAssistantGreeting(): Promise<AiMessage> {
  return request({
    mock: () => ({
      id: 'ai-message-001',
      role: 'assistant',
      content: '可以从事件、板块或资产角度提问，我会按决策辅助定位给出结构化分析。',
      safetyLevel: 'decision-support'
    })
  });
}
