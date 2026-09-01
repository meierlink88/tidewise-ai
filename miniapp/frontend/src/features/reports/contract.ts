export type ReportLayerKey = 'geopolitics' | 'macroeconomics';
export type ReportDetailTargetType = 'layer' | 'industry_chain';
export type ReportResultCode = 'warming' | 'cooling' | 'diverging' | 'pending';
export type ReportNatureCode = 'direct_evidence' | 'reasoning_hypothesis' | 'pending_validation';

export type ReportEvidenceScopeType =
  | 'report_card'
  | 'layer'
  | 'anchor'
  | 'reasoning_step'
  | 'transmission_path'
  | 'candidate_mechanism'
  | 'industry_chain'
  | 'industry_chain_node';

export type ReportReferenceType = 'layer' | 'anchor' | 'industry_chain' | 'industry_chain_node';

export interface ReportReference<T extends string = ReportReferenceType> {
  type: T;
  key: string;
}

export interface ReportResult {
  code: ReportResultCode;
  label: '升温' | '降温' | '分化' | '待验证';
}

export interface ReportNature {
  code: ReportNatureCode;
  label: '直接证据' | '推理假设' | '待验证';
}

export interface ReportConfidence {
  label: string;
  score: number | null;
}

export interface ReportSummary {
  id: string;
  title: string;
  generatedAt: string;
  publishedAt: string;
}

export type ReportHomeSelectionMode = 'today' | 'latest_fallback';

export interface ReportHomeSelection {
  mode: ReportHomeSelectionMode;
  date: string;
  timezone: 'Asia/Shanghai';
}

export type ReportCardKind = 'geopolitics' | 'macroeconomics' | 'industry_chain';

export interface ReportImpactItem {
  ref: ReportReference;
  name: string;
  result: ReportResult;
  confidence: ReportConfidence;
  timeWindow: string;
  hasEvidence: boolean;
}

export interface ReportCard {
  key: string;
  kind: ReportCardKind;
  displayOrder: number;
  detailRef: ReportReference<ReportDetailTargetType>;
  title: string;
  subtitle: string;
  conclusion: string;
  result: ReportResult;
  confidence: ReportConfidence;
  timeWindow: string;
  impactItems: ReportImpactItem[];
  hasEvidence: boolean;
}

export interface ReportCompanyBoundary {
  key: 'company';
  displayOrder: 4;
  title: string;
  published: false;
  boundary: string;
}

export interface ReportHomeGroup {
  report: ReportSummary;
  industryChainCount: number;
  cards: ReportCard[];
  company: ReportCompanyBoundary;
}

export interface ReportHome {
  selection: ReportHomeSelection;
  reports: ReportHomeGroup[];
}

export interface ReportAnchor {
  key: string;
  displayOrder: number;
  scope: ReportReference<ReportEvidenceScopeType>;
  name: string;
  currentState: string;
  result: ReportResult;
  nature: ReportNature;
  reasoning: string;
  timeWindow: string;
  confidence: ReportConfidence;
  hasEvidence: boolean;
}

export interface ReportReasoningStep {
  key: string;
  displayOrder: number;
  scope: ReportReference<ReportEvidenceScopeType>;
  input: string;
  mechanism: string;
  output: string;
  type: string;
  confidence: ReportConfidence;
  hasEvidence: boolean;
}

export interface ReportTransmissionTarget {
  ref: ReportReference;
  label: string;
  result: ReportResult;
}

export interface ReportTransmissionPath {
  key: string;
  displayOrder: number;
  scope: ReportReference<ReportEvidenceScopeType>;
  sourceConclusion: string;
  targetRefs: ReportTransmissionTarget[];
  logic: string;
  relationNature: string;
  evidenceRole: string;
  confidence: ReportConfidence;
  status: string;
  hasEvidence: boolean;
}

export interface ReportCandidateMechanism {
  key: string;
  displayOrder: number;
  scope: ReportReference<ReportEvidenceScopeType>;
  mechanism: string;
  evidenceGap: string | null;
  confidence: ReportConfidence;
  hasEvidence: boolean;
}

export interface ReportDownwardTransmission {
  summary: string;
  publishedPaths: ReportTransmissionPath[];
  candidateMechanisms: ReportCandidateMechanism[];
  boundaryNotes: string[];
}

export interface ReportCheckpoint {
  key: string;
  displayOrder: number;
  summary: string;
}

export interface ReportLayerUncertainty {
  counterevidence: string | null;
  evidenceGap: string | null;
  boundary: string | null;
  reversalCondition: string | null;
  checkpoints: ReportCheckpoint[];
}

export interface ReportLayerDetailContent {
  key: ReportLayerKey;
  displayOrder: number;
  scope: ReportReference<ReportEvidenceScopeType>;
  title: string;
  conclusion: string;
  result: ReportResult;
  confidence: ReportConfidence;
  timeWindow: string;
  anchors: ReportAnchor[];
  reasoningSteps: ReportReasoningStep[];
  downwardTransmission: ReportDownwardTransmission;
  uncertainty: ReportLayerUncertainty;
  hasEvidence: boolean;
}

export interface ReportRelatedIndustryChain {
  key: string;
  displayOrder: number;
  name: string;
  result: ReportResult;
  detailRef: ReportReference<'industry_chain'>;
}

export interface ReportLayerDetail {
  report: ReportSummary;
  layer: ReportLayerDetailContent;
  relatedIndustryChains: ReportRelatedIndustryChain[];
}

export interface ReportIndustryChainNode {
  key: string;
  displayOrder: number;
  scope: ReportReference<ReportEvidenceScopeType>;
  name: string;
  impact: string;
  result: ReportResult;
  nature: ReportNature;
  reasoning: string;
  timeWindow: string;
  confidence: ReportConfidence;
  hasEvidence: boolean;
}

export interface ReportGraphEdge {
  key: string;
  displayOrder: number;
  fromNodeKey: string;
  toNodeKey: string;
  relationLabel: string;
}

export interface ReportIndustryChainUncertainty {
  counterevidenceAndGap: string | null;
  stopCondition: string | null;
  checkpoints: ReportCheckpoint[];
}

export interface ReportIndustryChainDetailContent {
  key: string;
  claimKey: string;
  displayOrder: number;
  scope: ReportReference<ReportEvidenceScopeType>;
  name: string;
  conclusion: string;
  status: string;
  result: ReportResult;
  confidence: ReportConfidence;
  timeWindow: string;
  pathSummary: string | null;
  acceptedHypothesisSummary: string | null;
  nodes: ReportIndustryChainNode[];
  edges: ReportGraphEdge[];
  uncertainty: ReportIndustryChainUncertainty;
  hasEvidence: boolean;
}

export interface ReportIndustryChainDetail {
  report: ReportSummary;
  industryChain: ReportIndustryChainDetailContent;
}

export interface ReportEvidenceScope {
  type: ReportEvidenceScopeType;
  key: string;
}

export interface ReportEvidence {
  publishedAt: string | null;
  summary: string;
  keywords: string[];
}

export interface ReportEvidenceList {
  reportId: string;
  scope: ReportEvidenceScope;
  items: ReportEvidence[];
}

export interface ReportPort {
  getHome(): Promise<ReportHome>;
  getLayer(reportId: string, layerKey: ReportLayerKey): Promise<ReportLayerDetail>;
  getIndustryChain(reportId: string, chainKey: string): Promise<ReportIndustryChainDetail>;
  getEvidences(reportId: string, scope: ReportEvidenceScope): Promise<ReportEvidenceList>;
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
