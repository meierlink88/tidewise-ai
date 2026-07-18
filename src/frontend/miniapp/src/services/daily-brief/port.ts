import type { DailyBriefV1 } from '../../contracts/daily-brief-v1';

export interface DailyBriefPort {
  getDailyBrief(): Promise<DailyBriefV1 | null>;
}
