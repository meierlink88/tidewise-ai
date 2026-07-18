import { mockDailyBrief } from '../../mocks/daily-brief/mock-daily-brief';
import type { DailyBriefPort } from './port';

export type MockDailyBriefScenario = 'ready' | 'empty' | 'error' | 'loading';

export class MockDailyBriefAdapter implements DailyBriefPort {
  constructor(private readonly scenario: MockDailyBriefScenario = 'ready', private readonly loadingDelayMs = 30_000) {}

  async getDailyBrief() {
    if (this.scenario === 'error') {
      throw new Error('今日观潮加载失败');
    }
    if (this.scenario === 'loading') {
      await new Promise((resolve) => setTimeout(resolve, this.loadingDelayMs));
    }
    return this.scenario === 'empty' ? null : mockDailyBrief;
  }
}
