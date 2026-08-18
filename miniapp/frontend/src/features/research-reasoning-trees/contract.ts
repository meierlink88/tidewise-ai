import type {
  HomeResearchThemeItem,
  ResearchDirection,
  ResearchImpactStrength
} from '../research-themes/contract';

export type ResearchEvidenceRole = 'driver' | 'supporting' | 'contradicting' | 'context';
export type ResearchSignalDirection = 'increase' | 'decrease' | 'mixed' | 'unchanged' | 'uncertain';
export type ResearchReasoningTreeErrorKind =
  | 'invalidRequest'
  | 'themeUnavailable'
  | 'treesNotPublished'
  | 'treeUnavailable'
  | 'serviceUnavailable';
export class ResearchReasoningTreeError extends Error {
  constructor(public readonly kind: ResearchReasoningTreeErrorKind) {
    super(kind);
    this.name = 'ResearchReasoningTreeError';
  }
}
export type ResearchReasoningTreeTheme = HomeResearchThemeItem;
export interface ResearchReasoningTreeSummary {
  treeKey: string;
  displayName: string;
  reasoningTreeId: string;
  industryChainId: string | null;
  industryChainName: string;
  title: string;
  displayOrder: number;
  eventCount: number;
  publishedAt: string;
}
export interface ResearchReasoningTreeIndex {
  theme: ResearchReasoningTreeTheme;
  reasoningTrees: ResearchReasoningTreeSummary[];
}
export interface ResearchReasoningTreeEvent {
  eventId: string;
  title: string;
  summary: string;
  eventTime: string | null;
  evidenceRole: ResearchEvidenceRole;
  supportedClaim: string | null;
  displayOrder: number;
}
export interface ResearchReasoningTreeCheckpoint {
  type: 'event' | 'relationship' | 'metric';
  summary: string;
}
export interface ResearchReasoningTreeGraphEdge {
  id: string;
  relationType: string;
  reviewStatus: string;
  status: string;
}
export interface ResearchReasoningTreeSignal {
  signalKey: string;
  variableName: string | null;
  direction: ResearchSignalDirection | null;
  variableSignalKey: string | null;
  signalRole: 'primary' | 'supporting' | 'contradicting';
  signalDirection: ResearchSignalDirection | null;
  displaySummary: string;
  displayOrder: number;
}
export interface ResearchReasoningTreeNode {
  nodeKey: string;
  displayName: string;
  id: string;
  position: number;
  chainNodeId: string | null;
  name: string;
  stateSummary: string | null;
  impactDirection: ResearchDirection;
  impactStrength: ResearchImpactStrength;
  impactSummary: string | null;
  reasoningBasisSummary: string | null;
  evidenceGapSummary: string | null;
  incomingIndustryChainGraphEdgeId: string | null;
  incomingTransmissionTitle: string | null;
  incomingTransmissionMechanism: string | null;
  incomingConditionSummary: string | null;
  incomingGraphEdge: ResearchReasoningTreeGraphEdge | null;
  signals: ResearchReasoningTreeSignal[];
  primarySignal: ResearchReasoningTreeSignal;
  signalDisplaySummary: string;
}
export interface ResearchReasoningTree {
  treeKey: string;
  displayName: string;
  reasoningTreeId: string;
  themeId: string;
  industryChainId: string | null;
  industryChainName: string;
  title: string;
  displayOrder: number;
  oneLineConclusion: string;
  factSummary: string | null;
  transmissionSummary: string | null;
  impactDirection: ResearchDirection;
  impactStrength: ResearchImpactStrength;
  impactSummary: string | null;
  conclusionBoundarySummary: string | null;
  supportSummary: string | null;
  counterSummary: string | null;
  invalidationConditions: string[];
  checkpoints: ResearchReasoningTreeCheckpoint[];
  publishedAt: string;
  eventCount: number;
  events: ResearchReasoningTreeEvent[];
  nodes: ResearchReasoningTreeNode[];
}
export interface ResearchReasoningTreeDetail {
  themeId: string;
  impactNodeIds: string[];
  reasoningTree: ResearchReasoningTree;
}
export interface ResearchReasoningTreePort {
  list(themeId: string): Promise<ResearchReasoningTreeIndex>;
  get(themeId: string, reasoningTreeId: string): Promise<ResearchReasoningTreeDetail>;
}
