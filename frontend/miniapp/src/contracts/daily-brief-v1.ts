export const DAILY_BRIEF_SCHEMA_VERSION = 'mock.daily-brief.v1' as const;

export type ImpactEntityType = 'market' | 'sector' | 'benchmark' | 'commodity' | 'economy' | 'industry_chain';
export type Direction = 'up' | 'down' | 'neutral' | 'divergent';

export interface EvidenceItemV1 {
  id: string;
  source: string;
  title: string;
  summary: string;
  publishedAt: string;
  observedAt: string;
  confidence: 'high' | 'medium' | 'low';
}

export interface ImpactAssessmentV1 {
  id: string;
  entityId: string;
  entityType: ImpactEntityType;
  entityName: string;
  direction: Direction;
  horizon: 'short' | 'medium' | 'long';
  strength: 'high' | 'medium' | 'low';
  rationale: string;
  uncertainty: string;
}

export interface ReasoningConclusionV1 {
  id: string;
  badge: string;
  title: string;
  summary: string;
  direction: Direction;
  confidence: 'high' | 'medium' | 'low';
  graphAvailability: 'coming_soon' | 'unavailable';
  impacts: ImpactAssessmentV1[];
  evidence: EvidenceItemV1[];
}

export interface DailyBriefV1 {
  schemaVersion: typeof DAILY_BRIEF_SCHEMA_VERSION;
  id: string;
  asOf: string;
  displayDate: string;
  updatedAt: string;
  title: string;
  summary: string;
  market: { label: string; direction: Direction; hint: string };
  sentiment: { label: string; direction: Direction; hint: string };
  themes: string[];
  conclusions: ReasoningConclusionV1[];
  disclaimer: string;
}
