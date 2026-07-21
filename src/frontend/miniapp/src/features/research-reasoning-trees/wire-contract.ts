import {
  ResearchReasoningTreeError,
  type ResearchChangeDirection,
  type ResearchEvidenceRole,
  type ResearchReasoningTreeDetail,
  type ResearchReasoningTreeIndex,
  type ResearchReasoningTreeTheme
} from './contract';

const uuidPattern = /^[0-9a-f]{8}-[0-9a-f]{4}-[1-5][0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$/;

interface APIChainNode {
  id: string;
  name: string;
}

interface APIAffectedChainNode extends APIChainNode {
  relation_role: 'driver' | 'beneficiary' | 'constraint' | 'exposure';
  impact_summary: string;
}

interface APIIndex extends APIChainNode {
  impact_direction: 'positive' | 'negative' | 'mixed' | 'neutral';
  impact_summary: string;
}

interface APITheme {
  id: string;
  name: string;
  one_line_conclusion: string;
  impact_level: 'high' | 'focus' | 'watch';
  transmission_path: string;
  trading_direction: string;
  transmission_stage: 'identification' | 'validation' | 'diffusion' | 'dampening';
  next_checkpoint: string;
  market_confirmation_summary: string;
  published_at: string;
  affected_chain_nodes: APIAffectedChainNode[];
  related_indices: APIIndex[];
  supporting_event_count: number;
  contradicting_event_count: number;
}

interface APISummary {
  anchor_id: string;
  center_chain_node: APIChainNode;
}

interface APIIndexResponse {
  theme: APITheme;
  reasoning_trees: APISummary[];
}

interface APIEvent {
  event_id: string;
  title: string;
  summary: string;
  event_time: string | null;
  evidence_role: ResearchEvidenceRole;
  evidence_summary: string;
}

interface APIPathNode {
  chain_node_id: string;
  name: string;
  change_direction: ResearchChangeDirection;
  change_summary: string;
  impact_summary: string;
  incoming_transmission_mechanism: string | null;
}

interface APIDetailResponse {
  theme_id: string;
  reasoning_tree: {
    anchor_id: string;
    center_chain_node: APIChainNode;
    one_line_conclusion: string;
    fact_summary: string;
    net_direction_summary: string;
    support_summary: string;
    counter_summary: string | null;
    trading_direction: string;
    next_checkpoint: string;
    event_count: number;
    events: APIEvent[];
    path_nodes: APIPathNode[];
  };
}

export function parseResearchReasoningTreeIndex(value: unknown): ResearchReasoningTreeIndex {
  if (!isIndexResponse(value)) throw new ResearchReasoningTreeError('serviceUnavailable');
  return mapIndex(value);
}

export function parseResearchReasoningTreeDetail(
  value: unknown,
  themeId: string,
  anchorId: string
): ResearchReasoningTreeDetail {
  if (!isDetailResponse(value, themeId, anchorId)) {
    throw new ResearchReasoningTreeError('serviceUnavailable');
  }
  return mapDetail(value);
}

function isIndexResponse(value: unknown): value is APIIndexResponse {
  if (!isRecord(value) || !isTheme(value.theme) || !Array.isArray(value.reasoning_trees)) {
    return false;
  }
  if (value.reasoning_trees.length === 0) return false;
  const anchorIds = new Set<string>();
  return value.reasoning_trees.every((tree) => {
    if (!isRecord(tree) || !isUUID(tree.anchor_id) || !isChainNode(tree.center_chain_node)) {
      return false;
    }
    if (anchorIds.has(tree.anchor_id)) return false;
    anchorIds.add(tree.anchor_id);
    return true;
  });
}

function isDetailResponse(
  value: unknown,
  themeId: string,
  anchorId: string
): value is APIDetailResponse {
  if (!isRecord(value) || value.theme_id !== themeId || !isRecord(value.reasoning_tree)) {
    return false;
  }
  const tree = value.reasoning_tree;
  if (
    tree.anchor_id !== anchorId ||
    !isChainNode(tree.center_chain_node) ||
    !hasStrings(tree, [
      'one_line_conclusion',
      'fact_summary',
      'net_direction_summary',
      'support_summary',
      'trading_direction',
      'next_checkpoint'
    ]) ||
    (tree.counter_summary !== null && !isNonEmptyString(tree.counter_summary)) ||
    !isNonNegativeInteger(tree.event_count) ||
    !Array.isArray(tree.events) ||
    !Array.isArray(tree.path_nodes) ||
    tree.path_nodes.length === 0
  ) {
    return false;
  }
  if (tree.event_count !== tree.events.length) return false;
  const eventIds = new Set<string>();
  const nodeIds = new Set<string>();
  return (
    tree.events.every((event) => {
      if (!isEvent(event) || eventIds.has(event.event_id)) return false;
      eventIds.add(event.event_id);
      return true;
    }) &&
    tree.path_nodes.every((node, index) => {
      if (!isPathNode(node) || nodeIds.has(node.chain_node_id)) return false;
      if (
        (index === 0 && node.incoming_transmission_mechanism !== null) ||
        (index > 0 && node.incoming_transmission_mechanism === null)
      ) {
        return false;
      }
      nodeIds.add(node.chain_node_id);
      return true;
    })
  );
}

function isTheme(value: unknown): value is APITheme {
  if (!isRecord(value)) return false;
  return (
    isUUID(value.id) &&
    hasStrings(value, [
      'name',
      'one_line_conclusion',
      'transmission_path',
      'trading_direction',
      'next_checkpoint',
      'market_confirmation_summary',
      'published_at'
    ]) &&
    isOneOf(value.impact_level, ['high', 'focus', 'watch']) &&
    isOneOf(value.transmission_stage, ['identification', 'validation', 'diffusion', 'dampening']) &&
    Array.isArray(value.affected_chain_nodes) &&
    value.affected_chain_nodes.every(isAffectedChainNode) &&
    Array.isArray(value.related_indices) &&
    value.related_indices.every(isIndex) &&
    isNonNegativeInteger(value.supporting_event_count) &&
    isNonNegativeInteger(value.contradicting_event_count)
  );
}

function isAffectedChainNode(value: unknown): value is APIAffectedChainNode {
  const record = value as Record<string, unknown>;
  return (
    isChainNode(value) &&
    isOneOf(record.relation_role, ['driver', 'beneficiary', 'constraint', 'exposure']) &&
    isNonEmptyString(record.impact_summary)
  );
}

function isIndex(value: unknown): value is APIIndex {
  const record = value as Record<string, unknown>;
  return (
    isChainNode(value) &&
    isOneOf(record.impact_direction, ['positive', 'negative', 'mixed', 'neutral']) &&
    isNonEmptyString(record.impact_summary)
  );
}

function isEvent(value: unknown): value is APIEvent {
  if (!isRecord(value)) return false;
  return (
    isUUID(value.event_id) &&
    hasStrings(value, ['title', 'summary', 'evidence_summary']) &&
    (value.event_time === null || isUTCRFC3339(value.event_time)) &&
    isOneOf(value.evidence_role, ['driver', 'supporting', 'contradicting', 'context'])
  );
}

function isPathNode(value: unknown): value is APIPathNode {
  if (!isRecord(value)) return false;
  return (
    isUUID(value.chain_node_id) &&
    hasStrings(value, ['name', 'change_summary', 'impact_summary']) &&
    isOneOf(value.change_direction, ['increase', 'decrease', 'mixed', 'unchanged', 'uncertain']) &&
    (value.incoming_transmission_mechanism === null ||
      isNonEmptyString(value.incoming_transmission_mechanism))
  );
}

function isChainNode(value: unknown): value is APIChainNode {
  return isRecord(value) && isUUID(value.id) && isNonEmptyString(value.name);
}

function isRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === 'object' && value !== null;
}

function hasStrings(value: Record<string, unknown>, keys: string[]): boolean {
  return keys.every((key) => isNonEmptyString(value[key]));
}

function isNonEmptyString(value: unknown): value is string {
  return typeof value === 'string' && value.trim().length > 0;
}

function isUUID(value: unknown): value is string {
  return typeof value === 'string' && uuidPattern.test(value);
}

function isUTCRFC3339(value: unknown): value is string {
  return (
    typeof value === 'string' &&
    /^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}(?:\.\d+)?Z$/.test(value) &&
    !Number.isNaN(Date.parse(value))
  );
}

function isNonNegativeInteger(value: unknown): value is number {
  return typeof value === 'number' && Number.isInteger(value) && value >= 0;
}

function isOneOf<T extends string>(value: unknown, allowed: readonly T[]): value is T {
  return typeof value === 'string' && allowed.includes(value as T);
}

function mapIndex(value: APIIndexResponse): ResearchReasoningTreeIndex {
  return {
    theme: mapTheme(value.theme),
    reasoningTrees: value.reasoning_trees.map((tree) => ({
      anchorId: tree.anchor_id,
      centerChainNode: mapChainNode(tree.center_chain_node)
    }))
  };
}

function mapTheme(value: APITheme): ResearchReasoningTreeTheme {
  return {
    id: value.id,
    name: value.name,
    oneLineConclusion: value.one_line_conclusion,
    impactLevel: value.impact_level,
    transmissionPath: value.transmission_path,
    tradingDirection: value.trading_direction,
    transmissionStage: value.transmission_stage,
    nextCheckpoint: value.next_checkpoint,
    marketConfirmationSummary: value.market_confirmation_summary,
    publishedAt: value.published_at,
    affectedChainNodes: value.affected_chain_nodes.map((node) => ({
      id: node.id,
      name: node.name,
      relationRole: node.relation_role,
      impactSummary: node.impact_summary
    })),
    relatedIndices: value.related_indices.map((index) => ({
      id: index.id,
      name: index.name,
      impactDirection: index.impact_direction,
      impactSummary: index.impact_summary
    })),
    supportingEventCount: value.supporting_event_count,
    contradictingEventCount: value.contradicting_event_count
  };
}

function mapDetail(value: APIDetailResponse): ResearchReasoningTreeDetail {
  const tree = value.reasoning_tree;
  return {
    themeId: value.theme_id,
    reasoningTree: {
      anchorId: tree.anchor_id,
      centerChainNode: mapChainNode(tree.center_chain_node),
      oneLineConclusion: tree.one_line_conclusion,
      factSummary: tree.fact_summary,
      netDirectionSummary: tree.net_direction_summary,
      supportSummary: tree.support_summary,
      counterSummary: tree.counter_summary,
      tradingDirection: tree.trading_direction,
      nextCheckpoint: tree.next_checkpoint,
      eventCount: tree.event_count,
      events: tree.events.map((event) => ({
        eventId: event.event_id,
        title: event.title,
        summary: event.summary,
        eventTime: event.event_time,
        evidenceRole: event.evidence_role,
        evidenceSummary: event.evidence_summary
      })),
      pathNodes: tree.path_nodes.map((node) => ({
        chainNodeId: node.chain_node_id,
        name: node.name,
        changeDirection: node.change_direction,
        changeSummary: node.change_summary,
        impactSummary: node.impact_summary,
        incomingTransmissionMechanism: node.incoming_transmission_mechanism
      }))
    }
  };
}

function mapChainNode(value: APIChainNode) {
  return { id: value.id, name: value.name };
}
