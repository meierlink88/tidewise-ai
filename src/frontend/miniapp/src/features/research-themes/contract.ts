export type ResearchImpactLevel = 'high' | 'focus' | 'watch';
export type ResearchTransmissionStage = 'upstream' | 'midstream' | 'downstream' | 'infrastructure' | 'service';

export interface HomeResearchChainNode {
  id: string;
  name: string;
  relationRole: 'driver' | 'beneficiary' | 'constraint' | 'exposure';
  impactSummary: string;
}

export interface HomeResearchIndex {
  id: string;
  name: string;
  impactDirection: 'positive' | 'negative' | 'mixed' | 'neutral';
  impactSummary: string;
}

export interface HomeResearchThemeItem {
  id: string;
  name: string;
  oneLineConclusion: string;
  impactLevel: ResearchImpactLevel;
  transmissionPath: string;
  tradingDirection: string;
  transmissionStage: ResearchTransmissionStage;
  transmissionPhaseLabel: string;
  nextCheckpoint: string;
  indexImpactSummary: string;
  publishedAt: string;
  updateLabel: string;
  categories: string[];
  affectedChainNodes: HomeResearchChainNode[];
  relatedIndices: HomeResearchIndex[];
  supportingEventCount: number;
  contradictingEventCount: number;
  hasMoreDetail: boolean;
}

export interface HomeResearchThemeFeed {
  windowStart: string;
  windowEnd: string;
  asOf: string;
  themeCount: number;
  eventCount: number;
  trackingCount: number;
  items: HomeResearchThemeItem[];
  nextCursor: string | null;
}

export interface ResearchThemeFeedPort {
  list(): Promise<HomeResearchThemeFeed>;
}
