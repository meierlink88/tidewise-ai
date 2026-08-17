import {
  ResearchReasoningTreeError,
  type ResearchEvidenceRole,
  type ResearchReasoningTree,
  type ResearchReasoningTreeDetail,
  type ResearchReasoningTreeEvent,
  type ResearchReasoningTreeIndex,
  type ResearchReasoningTreeNode,
  type ResearchReasoningTreeSignal,
  type ResearchReasoningTreeSummary,
  type ResearchSignalDirection
} from './contract';
import type { ResearchDirection, ResearchImpactStrength } from '../research-themes/contract';
import { parseResearchThemeWire } from '../research-themes/wire-contract';

const uuidPattern = /^[0-9a-f]{8}-[0-9a-f]{4}-[1-5][0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$/;
const entityIDPattern =
  /^ENT[0-9a-f]{8}-[0-9a-f]{4}-[1-5][0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$/;
const directionValues = ['positive', 'negative', 'mixed', 'neutral', 'uncertain'] as const;
const strengthValues = ['strong', 'medium', 'weak', 'unknown'] as const;
const evidenceRoleValues = ['driver', 'supporting', 'contradicting', 'context'] as const;
const signalRoleValues = ['primary', 'supporting', 'contradicting'] as const;
const signalDirectionValues = ['increase', 'decrease', 'mixed', 'unchanged', 'uncertain'] as const;

type RecordValue = Record<string, unknown>;

export function parseResearchReasoningTreeIndex(value: unknown): ResearchReasoningTreeIndex {
  return failClosed(() => {
    const root = record(value);
    onlyKeys(root, ['theme', 'reasoning_trees']);
    const theme = parseResearchThemeWire(root.theme);
    const reasoningTrees = array(root.reasoning_trees).map((item, index) => {
      const summary = mapSummary(record(item));
      if (summary.displayOrder !== index + 1) invalid();
      return summary;
    });
    return { theme, reasoningTrees };
  });
}

export function parseResearchReasoningTreeDetail(
  value: unknown,
  themeId: string,
  reasoningTreeId: string
): ResearchReasoningTreeDetail {
  return failClosed(() => {
    const root = record(value);
    onlyKeys(root, ['theme_id', 'impact_node_ids', 'reasoning_tree']);
    if (uuid(root.theme_id) !== themeId) invalid();
    const reasoningTree = mapTree(record(root.reasoning_tree));
    if (reasoningTree.themeId !== themeId || reasoningTree.reasoningTreeId !== reasoningTreeId) {
      invalid();
    }
    return {
      themeId,
      impactNodeIds: array(root.impact_node_ids).map(localKey),
      reasoningTree
    };
  });
}

function mapSummary(value: RecordValue): ResearchReasoningTreeSummary {
  const snapshot = 'tree_key' in value;
  onlyKeys(value, [
    ...(snapshot ? ['tree_key', 'display_name'] : []),
    'reasoning_tree_id',
    'industry_chain_id',
    'industry_chain_name',
    'title',
    'display_order',
    'event_count',
    'published_at'
  ]);
  return {
    treeKey: snapshot ? localKey(value.tree_key) : entityID(value.industry_chain_id),
    displayName: text(snapshot ? value.display_name : value.industry_chain_name),
    reasoningTreeId: uuid(value.reasoning_tree_id),
    industryChainEntityId: snapshot
      ? nullableEntityIDString(value.industry_chain_id)
      : entityID(value.industry_chain_id),
    industryChainName: text(snapshot ? value.display_name : value.industry_chain_name),
    title: text(value.title),
    displayOrder: positiveInteger(value.display_order),
    eventCount: nonNegativeInteger(value.event_count),
    publishedAt: timestamp(value.published_at)
  };
}

function mapTree(value: RecordValue): ResearchReasoningTree {
  const snapshot = 'tree_key' in value;
  onlyKeys(value, [
    ...(snapshot ? ['tree_key', 'display_name'] : []),
    'reasoning_tree_id',
    'theme_id',
    'industry_chain_id',
    'industry_chain_name',
    'title',
    'display_order',
    'one_line_conclusion',
    'fact_summary',
    'transmission_summary',
    'impact_direction',
    'impact_strength',
    'impact_summary',
    'conclusion_boundary_summary',
    'support_summary',
    'counter_summary',
    'invalidation_conditions',
    'checkpoints',
    'published_at',
    'event_count',
    'events',
    'nodes'
  ]);
  const events = array(value.events).map((item, index) => mapEvent(record(item), index));
  const nodes = array(value.nodes).map((item, index) => mapNode(record(item), index));
  if (nodes.length === 0 || nonNegativeInteger(value.event_count) !== events.length) invalid();
  return {
    treeKey: snapshot ? localKey(value.tree_key) : entityID(value.industry_chain_id),
    displayName: text(snapshot ? value.display_name : value.industry_chain_name),
    reasoningTreeId: uuid(value.reasoning_tree_id),
    themeId: uuid(value.theme_id),
    industryChainEntityId: snapshot
      ? nullableEntityIDString(value.industry_chain_id)
      : entityID(value.industry_chain_id),
    industryChainName: text(snapshot ? value.display_name : value.industry_chain_name),
    title: text(value.title),
    displayOrder: positiveInteger(value.display_order),
    oneLineConclusion: text(value.one_line_conclusion),
    factSummary: nullableText(value.fact_summary),
    transmissionSummary: nullableText(value.transmission_summary),
    impactDirection: enumValue<ResearchDirection>(value.impact_direction, directionValues),
    impactStrength: enumValue<ResearchImpactStrength>(value.impact_strength, strengthValues),
    impactSummary: nullableText(value.impact_summary),
    conclusionBoundarySummary: nullableText(value.conclusion_boundary_summary),
    supportSummary: nullableText(value.support_summary),
    counterSummary: nullableText(value.counter_summary),
    invalidationConditions: array(value.invalidation_conditions).map(text),
    checkpoints: array(value.checkpoints).map((item) => {
      const checkpoint = record(item);
      onlyKeys(checkpoint, ['type', 'summary']);
      return {
        type: enumValue(checkpoint.type, ['event', 'relationship', 'metric'] as const),
        summary: text(checkpoint.summary)
      };
    }),
    publishedAt: timestamp(value.published_at),
    eventCount: events.length,
    events,
    nodes
  };
}

function mapEvent(value: RecordValue, index: number): ResearchReasoningTreeEvent {
  const withEvidence = 'evidence_ids' in value;
  onlyKeys(value, [
    'event_id',
    ...(withEvidence ? ['evidence_ids'] : []),
    'title',
    'summary',
    'event_time',
    'evidence_role',
    'supported_claim',
    'display_order'
  ]);
  const displayOrder = positiveInteger(value.display_order);
  if (displayOrder !== index + 1) invalid();
  if (withEvidence) array(value.evidence_ids).forEach(uuid);
  return {
    eventId: uuid(value.event_id),
    title: text(value.title),
    summary: text(value.summary),
    eventTime: value.event_time === null ? null : timestamp(value.event_time),
    evidenceRole: enumValue<ResearchEvidenceRole>(value.evidence_role, evidenceRoleValues),
    supportedClaim: nullableText(value.supported_claim),
    displayOrder
  };
}

function mapNode(value: RecordValue, index: number): ResearchReasoningTreeNode {
  const snapshot = 'node_key' in value;
  onlyKeys(value, [
    ...(snapshot ? ['node_key', 'display_name'] : []),
    'id',
    'position',
    'chain_node_id',
    'name',
    'state_summary',
    'impact_direction',
    'impact_strength',
    'impact_summary',
    'reasoning_basis_summary',
    'evidence_gap_summary',
    'incoming_industry_chain_graph_edge_id',
    'incoming_transmission_title',
    'incoming_transmission_mechanism',
    'incoming_condition_summary',
    'incoming_graph_edge',
    'signals',
    'primary_signal',
    'signal_display_summary'
  ]);
  const position = positiveInteger(value.position);
  if (position !== index + 1) invalid();
  const signals = array(value.signals).map((item, signalIndex) =>
    mapSignal(record(item), signalIndex)
  );
  if (signals.length < 1 || signals.length > 5) invalid();
  const primarySignal = mapSignal(record(value.primary_signal), 0);
  const primary = signals.filter((signal) => signal.signalRole === 'primary');
  if (primary.length !== 1 || !sameSignal(primary[0], primarySignal)) invalid();
  const signalDisplaySummary = textAllowEmpty(value.signal_display_summary);
  if (
    signalDisplaySummary !==
    signals
      .filter((signal) => signal.signalRole !== 'primary')
      .map((signal) => signal.displaySummary)
      .join(' · ')
  ) {
    invalid();
  }
  const incomingGraphEdge =
    value.incoming_graph_edge === null ? null : mapGraphEdge(record(value.incoming_graph_edge));
  return {
    nodeKey: snapshot ? localKey(value.node_key) : entityID(value.chain_node_id),
    displayName: text(snapshot ? value.display_name : value.name),
    id: uuid(value.id),
    position,
    chainNodeEntityId: snapshot
      ? nullableEntityIDString(value.chain_node_id)
      : entityID(value.chain_node_id),
    name: text(snapshot ? value.display_name : value.name),
    stateSummary: nullableText(value.state_summary),
    impactDirection: enumValue<ResearchDirection>(value.impact_direction, directionValues),
    impactStrength: enumValue<ResearchImpactStrength>(value.impact_strength, strengthValues),
    impactSummary: nullableText(value.impact_summary),
    reasoningBasisSummary: nullableText(value.reasoning_basis_summary),
    evidenceGapSummary: nullableText(value.evidence_gap_summary),
    incomingIndustryChainGraphEdgeId: nullableUUID(value.incoming_industry_chain_graph_edge_id),
    incomingTransmissionTitle: nullableText(value.incoming_transmission_title),
    incomingTransmissionMechanism: nullableText(value.incoming_transmission_mechanism),
    incomingConditionSummary: nullableText(value.incoming_condition_summary),
    incomingGraphEdge,
    signals,
    primarySignal,
    signalDisplaySummary
  };
}

function mapSignal(value: RecordValue, index: number): ResearchReasoningTreeSignal {
  const snapshot = 'signal_key' in value;
  onlyKeys(value, [
    ...(snapshot ? ['signal_key', 'variable_name', 'direction'] : []),
    'variable_signal_key',
    'signal_role',
    'signal_direction',
    'display_summary',
    'display_order'
  ]);
  const displayOrder = positiveInteger(value.display_order);
  if (displayOrder !== index + 1) invalid();
  return {
    signalKey: snapshot ? localKey(value.signal_key) : text(value.variable_signal_key),
    variableName: snapshot ? nullableText(value.variable_name) : null,
    direction: snapshot
      ? nullableEnum<ResearchSignalDirection>(value.direction, signalDirectionValues)
      : enumValue<ResearchSignalDirection>(value.signal_direction, signalDirectionValues),
    variableSignalKey: snapshot
      ? nullableTextAllowEmpty(value.variable_signal_key)
      : text(value.variable_signal_key),
    signalRole: enumValue(value.signal_role, signalRoleValues),
    signalDirection: snapshot
      ? nullableEnum<ResearchSignalDirection>(value.direction, signalDirectionValues)
      : enumValue<ResearchSignalDirection>(value.signal_direction, signalDirectionValues),
    displaySummary: text(value.display_summary),
    displayOrder
  };
}

function mapGraphEdge(value: RecordValue) {
  onlyKeys(value, ['id', 'relation_type', 'review_status', 'status']);
  return {
    id: uuid(value.id),
    relationType: text(value.relation_type),
    reviewStatus: text(value.review_status),
    status: text(value.status)
  };
}

function sameSignal(left: ResearchReasoningTreeSignal, right: ResearchReasoningTreeSignal) {
  return (
    left.signalKey === right.signalKey &&
    left.signalRole === right.signalRole &&
    left.signalDirection === right.signalDirection &&
    left.displaySummary === right.displaySummary &&
    left.displayOrder === right.displayOrder
  );
}

function localKey(value: unknown): string {
  const result = text(value);
  if (!/^[a-z0-9][a-z0-9._:-]{0,127}$/.test(result)) invalid();
  return result;
}

function nullableEntityIDString(value: unknown): string | null {
  return value === '' || value === null ? null : entityID(value);
}

function nullableTextAllowEmpty(value: unknown): string | null {
  return value === '' || value === null ? null : text(value);
}

function failClosed<T>(operation: () => T): T {
  try {
    return operation();
  } catch (error) {
    if (error instanceof ResearchReasoningTreeError) throw error;
    throw new ResearchReasoningTreeError('serviceUnavailable');
  }
}

function invalid(): never {
  throw new ResearchReasoningTreeError('serviceUnavailable');
}

function record(value: unknown): RecordValue {
  if (typeof value !== 'object' || value === null || Array.isArray(value)) invalid();
  return value as RecordValue;
}

function array(value: unknown): unknown[] {
  if (!Array.isArray(value)) invalid();
  return value;
}

function onlyKeys(value: RecordValue, allowed: string[]) {
  const keys = Object.keys(value);
  if (keys.length !== allowed.length || keys.some((key) => !allowed.includes(key))) invalid();
}

function text(value: unknown): string {
  if (typeof value !== 'string' || value.length === 0) invalid();
  return value;
}

function textAllowEmpty(value: unknown): string {
  if (typeof value !== 'string') invalid();
  return value;
}

function nullableText(value: unknown): string | null {
  return value === null ? null : textAllowEmpty(value);
}

function uuid(value: unknown): string {
  const parsed = text(value);
  if (!uuidPattern.test(parsed)) invalid();
  return parsed;
}

function entityID(value: unknown): string {
  const parsed = text(value);
  if (!entityIDPattern.test(parsed)) invalid();
  return parsed;
}

function nullableUUID(value: unknown): string | null {
  return value === null ? null : uuid(value);
}

function timestamp(value: unknown): string {
  const parsed = text(value);
  if (!parsed.endsWith('Z') || !Number.isFinite(Date.parse(parsed))) invalid();
  return parsed;
}

function positiveInteger(value: unknown): number {
  if (!Number.isInteger(value) || (value as number) < 1) invalid();
  return value as number;
}

function nonNegativeInteger(value: unknown): number {
  if (!Number.isInteger(value) || (value as number) < 0) invalid();
  return value as number;
}

function enumValue<T extends string>(value: unknown, allowed: readonly T[]): T {
  if (typeof value !== 'string' || !allowed.includes(value as T)) invalid();
  return value as T;
}

function nullableEnum<T extends string>(value: unknown, allowed: readonly T[]): T | null {
  return value === null ? null : enumValue(value, allowed);
}

export function isLowercaseUUID(value: string): boolean {
  return uuidPattern.test(value);
}
