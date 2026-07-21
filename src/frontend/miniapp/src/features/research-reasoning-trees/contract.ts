import type {
  HomeResearchChainNode,
  HomeResearchIndex,
  ResearchImpactLevel,
  ResearchTransmissionStage
} from '../research-themes/contract';

export type ResearchEvidenceRole = 'driver' | 'supporting' | 'contradicting' | 'context';
export type ResearchChangeDirection = 'increase' | 'decrease' | 'mixed' | 'unchanged' | 'uncertain';
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

export interface ResearchReasoningTreeTheme {
  id: string;
  name: string;
  oneLineConclusion: string;
  impactLevel: ResearchImpactLevel;
  transmissionPath: string;
  tradingDirection: string;
  transmissionStage: ResearchTransmissionStage;
  nextCheckpoint: string;
  marketConfirmationSummary: string;
  publishedAt: string;
  affectedChainNodes: HomeResearchChainNode[];
  relatedIndices: HomeResearchIndex[];
  supportingEventCount: number;
  contradictingEventCount: number;
}

export interface ResearchReasoningTreeChainNode {
  id: string;
  name: string;
}

export interface ResearchReasoningTreeSummary {
  anchorId: string;
  centerChainNode: ResearchReasoningTreeChainNode;
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
  evidenceSummary: string;
}

export interface ResearchReasoningTreePathNode {
  chainNodeId: string;
  name: string;
  changeDirection: ResearchChangeDirection;
  changeSummary: string;
  impactSummary: string;
  incomingTransmissionMechanism: string | null;
}

export interface ResearchReasoningTree {
  anchorId: string;
  centerChainNode: ResearchReasoningTreeChainNode;
  oneLineConclusion: string;
  factSummary: string;
  netDirectionSummary: string;
  supportSummary: string;
  counterSummary: string | null;
  tradingDirection: string;
  nextCheckpoint: string;
  eventCount: number;
  events: ResearchReasoningTreeEvent[];
  pathNodes: ResearchReasoningTreePathNode[];
}

export interface ResearchReasoningTreeDetail {
  themeId: string;
  reasoningTree: ResearchReasoningTree;
}

export interface ResearchReasoningTreePort {
  list(themeId: string): Promise<ResearchReasoningTreeIndex>;
  get(themeId: string, anchorId: string): Promise<ResearchReasoningTreeDetail>;
}
