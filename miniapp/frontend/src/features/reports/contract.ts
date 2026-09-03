export type ReportLayerKey = 'geopolitics' | 'macroeconomics';
export type ReportDetailTargetType = 'layer' | 'industry_chain';
export type ReportCardKind = ReportLayerKey | 'industry_chain';

export interface ReportReference<T extends string = string> {
  type: T;
  localKey: string;
}

/** Codes are server-owned and intentionally open for forward-compatible rendering. */
export interface ReportCodedLabel {
  code: string;
  label: string;
}

export type ReportResult = ReportCodedLabel;
export type ReportResultCode = string;
export type ReportNature = ReportCodedLabel;
export type ReportNatureCode = string;

export interface ReportConfidence extends ReportCodedLabel {
  score: number | null;
}

export interface ReportTimeWindow extends ReportCodedLabel {}

export interface ReportSummary {
  id: string;
  generatedAt: string;
  publishedAt: string;
  industryChainCount: number;
}

export type ReportHomeSelectionMode = 'today' | 'latest_fallback';

export interface ReportHomeSelection {
  mode: ReportHomeSelectionMode;
  date: string;
  timezone: 'Asia/Shanghai';
}

export interface ReportImpactItem {
  ref: ReportReference;
  name: string;
  result: ReportResult;
  conclusionBasis: ReportCodedLabel | null;
  validationStatus: ReportCodedLabel | null;
  confidence: ReportConfidence;
  timeWindow: ReportTimeWindow;
  evidenceScopeToken: string | null;
}

export interface ReportCard {
  key: string;
  kind: ReportCardKind;
  detailRef: ReportReference<ReportDetailTargetType>;
  title: string;
  subtitle: string;
  conclusion: string;
  result: ReportResult;
  confidence: ReportConfidence;
  timeWindow: ReportTimeWindow;
  impactItems: ReportImpactItem[];
  evidenceScopeToken: string | null;
}

export interface ReportCardPage {
  items: ReportCard[];
  nextCursor: string | null;
}

export interface ReportHomeGroup {
  report: ReportSummary;
  cards: ReportCard[];
  nextCursor: string | null;
}

export interface ReportHome {
  selection: ReportHomeSelection;
  reports: ReportHomeGroup[];
}

export interface ReportAnchor {
  key: string;
  name: string;
  currentState: string;
  result: ReportResult;
  conclusionBasis: ReportCodedLabel | null;
  validationStatus: ReportCodedLabel | null;
  transmissionLogic: string;
  timeWindow: ReportTimeWindow;
  confidence: ReportConfidence;
  evidenceScopeToken: string | null;
}

export interface ReportReasoningStep {
  input: string;
  mechanism: string;
  output: string;
  reasoningType: ReportCodedLabel;
  confidence: ReportConfidence;
  evidenceScopeToken: string | null;
}

export interface ReportTransmissionTarget {
  ref: ReportReference;
  name: string;
  result: ReportResult;
}

export interface ReportTransmissionPath {
  key: string;
  sourceConclusion: string;
  targets: ReportTransmissionTarget[];
  logic: string;
  kind: ReportCodedLabel;
  confidence: ReportConfidence;
  status: ReportCodedLabel;
}

export interface ReportLayerUncertainty {
  counterevidence: string | null;
  evidenceGap: string | null;
  boundary: string | null;
  reversalCondition: string | null;
}

export interface ReportLayerDetailContent {
  key: ReportLayerKey;
  title: string;
  conclusion: string;
  result: ReportResult;
  confidence: ReportConfidence;
  timeWindow: ReportTimeWindow;
  anchors: ReportAnchor[];
  reasoningSteps: ReportReasoningStep[];
  transmissions: ReportTransmissionPath[];
  uncertainty: ReportLayerUncertainty;
  evidenceScopeToken: string | null;
}

export interface ReportRelatedIndustryChain {
  key: string;
  name: string;
  result: ReportResult;
}

export interface ReportLayerDetail {
  report: ReportSummary;
  layer: ReportLayerDetailContent;
  relatedIndustryChains: ReportRelatedIndustryChain[];
}

export interface ReportIndustryChainNode {
  key: string;
  nodeLocalKey: string;
  name: string;
  impact: string;
  result: ReportResult;
  conclusionBasis: ReportCodedLabel | null;
  validationStatus: ReportCodedLabel | null;
  transmissionLogic: string;
  timeWindow: ReportTimeWindow;
  confidence: ReportConfidence;
  evidenceScopeToken: string | null;
}

export interface ReportGraphEdge {
  fromNodeKey: string;
  toNodeKey: string;
  relation: ReportCodedLabel;
}

export interface ReportGraphNode {
  key: string;
  name: string;
}

export interface ReportIndustryChainDetailContent {
  key: string;
  name: string;
  conclusion: string;
  status: string;
  result: ReportResult;
  confidence: ReportConfidence;
  timeWindow: ReportTimeWindow;
  path: string;
  topologyNodes: ReportGraphNode[];
  nodes: ReportIndustryChainNode[];
  edges: ReportGraphEdge[];
  counterevidenceAndGap: string;
  stopCondition: string;
  evidenceScopeToken: string | null;
}

export interface ReportIndustryChainDetail {
  report: ReportSummary;
  industryChain: ReportIndustryChainDetailContent;
}

export interface ReportEvidence {
  publishedAt: string | null;
  summary: string;
  keywords: string[];
}

export interface ReportEvidenceList {
  reportId: string;
  scopeToken: string;
  items: ReportEvidence[];
}

export interface ReportPort {
  getHome(): Promise<ReportHome>;
  getIndustryChains(reportId: string, cursor?: string, limit?: number): Promise<ReportCardPage>;
  getLayer(reportId: string, layerKey: ReportLayerKey): Promise<ReportLayerDetail>;
  getIndustryChain(reportId: string, chainKey: string): Promise<ReportIndustryChainDetail>;
  getEvidences(reportId: string, scopeToken: string): Promise<ReportEvidenceList>;
}

export type ReportErrorKind =
  | 'invalidRequest'
  | 'reportUnavailable'
  | 'layerUnavailable'
  | 'chainUnavailable'
  | 'evidenceScopeUnavailable'
  | 'serviceUnavailable'
  | 'invalidResponse';

export class ReportError extends Error {
  constructor(public readonly kind: ReportErrorKind) {
    super(kind);
    this.name = 'ReportError';
  }
}
