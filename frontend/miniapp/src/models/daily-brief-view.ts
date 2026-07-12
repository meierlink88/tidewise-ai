import type { Direction, EvidenceItemV1, ImpactAssessmentV1 } from '../contracts/daily-brief-v1';

export interface HomeConclusionView {
  id: string;
  badge: string;
  title: string;
  summary: string;
  direction: Direction;
  confidenceLabel: string;
  impacts: ImpactAssessmentV1[];
  evidence: EvidenceItemV1[];
  graphAvailability: 'coming_soon' | 'unavailable';
}

export interface DailyBriefHomeView {
  id: string;
  displayDate: string;
  updatedAt: string;
  summary: string;
  market: { label: string; direction: Direction; hint: string };
  sentiment: { label: string; direction: Direction; hint: string };
  themes: string[];
  conclusions: HomeConclusionView[];
  disclaimer: string;
}
