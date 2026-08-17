export type ResearchDirection = 'positive' | 'negative' | 'mixed' | 'neutral' | 'uncertain';
export type ResearchImpactStrength = 'strong' | 'medium' | 'weak' | 'unknown';
export type ResearchTransmissionStage = 'identification' | 'validation' | 'diffusion' | 'dampening';
export type ResearchAttentionLevel = 'high' | 'medium' | 'low';
export type ResearchConclusionStatus = 'supported' | 'partial' | 'conflicted';
export type ResearchInvestmentGuidanceAction = 'focus' | 'avoid' | 'observe' | 'differentiate';
export type ResearchThemePeriod = 'today' | 'history';

export interface ResearchThemeListRequest {
  period: ResearchThemePeriod;
  limit: number;
  cursor?: string;
}

export interface HomeResearchThemeImpact {
  nodeKey: string;
  displayName: string;
  chainNodeId: string | null;
  name: string;
  relationRole: 'driver' | 'beneficiary' | 'constraint' | 'exposure';
  impactDirection: ResearchDirection;
  impactSummary: string | null;
  displayOrder: number;
}

export interface HomeResearchThemeItem {
  id: string;
  analysisBatchId: string;
  title: string;
  oneLineConclusion: string;
  conclusionDirection: ResearchDirection;
  impactStrength: ResearchImpactStrength;
  attentionLevel: ResearchAttentionLevel | null;
  conclusionStatus: ResearchConclusionStatus | null;
  transmissionStage: ResearchTransmissionStage;
  investmentGuidanceAction: ResearchInvestmentGuidanceAction;
  investmentGuidanceSummary: string;
  timeHorizonCategory: 'short_term' | 'medium_term' | 'long_term' | 'custom';
  timeHorizonSummary: string | null;
  transmissionSummary: string | null;
  checkpointSummary: string | null;
  riskSummary: string | null;
  analysisAsOf: string;
  windowStart: string;
  windowEnd: string;
  publishedAt: string;
  updateLabel: string;
  impacts: HomeResearchThemeImpact[];
  evidenceEventCount: number;
  reasoningTreeCount: number;
}

export interface HomeResearchThemeFeed {
  windowStart: string;
  windowEnd: string;
  asOf: string;
  themeCount: number;
  eventCount: number;
  items: HomeResearchThemeItem[];
  nextCursor: string | null;
}

export interface ResearchThemeEvent {
  eventId: string;
  title: string;
  summary: string;
  eventTime: { status: 'pending' } | { status: 'confirmed'; date: string; time: string };
}

export interface ResearchThemeDetail {
  id: string;
  title: string;
  events: ResearchThemeEvent[];
}

export type ResearchThemeDetailErrorKind = 'themeUnavailable' | 'serviceUnavailable';

export class ResearchThemeDetailError extends Error {
  constructor(public readonly kind: ResearchThemeDetailErrorKind) {
    super(kind);
    this.name = 'ResearchThemeDetailError';
  }
}

export interface ResearchThemeHomepagePort {
  list(request: ResearchThemeListRequest): Promise<HomeResearchThemeFeed>;
  getDetail(themeId: string): Promise<ResearchThemeDetail>;
}
