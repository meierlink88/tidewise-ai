import type { DailyBriefV1 } from '../contracts/daily-brief-v1';
import type { DailyBriefHomeView } from '../models/daily-brief-view';

const confidenceLabels = { high: '高可信', medium: '中可信', low: '待验证' } as const;

export function mapDailyBriefToHome(brief: DailyBriefV1): DailyBriefHomeView {
  return {
    id: brief.id,
    displayDate: brief.displayDate,
    updatedAt: brief.updatedAt,
    summary: brief.summary,
    market: brief.market,
    sentiment: brief.sentiment,
    themes: [...brief.themes],
    conclusions: brief.conclusions.map((conclusion) => ({
      ...conclusion,
      confidenceLabel: confidenceLabels[conclusion.confidence],
      impacts: [...conclusion.impacts],
      evidence: [...conclusion.evidence]
    })),
    disclaimer: brief.disclaimer
  };
}
