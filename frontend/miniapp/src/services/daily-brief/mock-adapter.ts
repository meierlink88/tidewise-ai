import { mockDailyBrief } from '../../data/daily-brief/mock-daily-brief';
import type { DailyBriefPort } from './port';

export type MockDailyBriefScenario = 'ready' | 'empty' | 'error';

export class MockDailyBriefAdapter implements DailyBriefPort {
  constructor(private readonly scenario: MockDailyBriefScenario = 'ready') {}

  async getDailyBrief() {
    if (this.scenario === 'error') {
      throw new Error('今日观潮加载失败');
    }
    return this.scenario === 'empty' ? null : mockDailyBrief;
  }
}
