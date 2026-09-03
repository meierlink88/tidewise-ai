import type {
  ReportAnchor,
  ReportCard,
  ReportCardPage,
  ReportCodedLabel,
  ReportConfidence,
  ReportEvidence,
  ReportEvidenceList,
  ReportGraphEdge,
  ReportHome,
  ReportHomeGroup,
  ReportIndustryChainDetail,
  ReportIndustryChainDetailContent,
  ReportIndustryChainNode,
  ReportLayerDetail,
  ReportLayerDetailContent,
  ReportLayerKey,
  ReportReasoningStep,
  ReportReference,
  ReportRelatedIndustryChain,
  ReportSummary,
  ReportTimeWindow,
  ReportTransmissionPath,
  ReportTransmissionTarget
} from './contract';

type RecordValue = Record<string, unknown>;

const reportIDPattern =
  /^RPT[0-9a-f]{8}-[0-9a-f]{4}-[1-5][0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$/;
const scopeTokenPattern =
  /^RPE[0-9a-f]{8}-[0-9a-f]{4}-[1-5][0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$/;
const localKeyPattern = /^[A-Za-z0-9][A-Za-z0-9._-]{0,127}$/;
const datePattern = /^\d{4}-\d{2}-\d{2}$/;
const layerKeys = ['geopolitics', 'macroeconomics'] as const;
const cardKinds = ['geopolitics', 'macroeconomics', 'industry_chain'] as const;
const referenceTypes = [
  'layer',
  'anchor',
  'macro_anchor',
  'industry_chain',
  'industry_chain_node'
] as const;

export function parseReportHomeWire(value: unknown): ReportHome {
  const root = exact(value, ['selection', 'reports']);
  const selectionWire = exact(root.selection, ['mode', 'date', 'timezone']);
  const selection = {
    mode: enumeration(selectionWire.mode, ['today', 'latest_fallback'] as const),
    date: match(selectionWire.date, datePattern),
    timezone: literal(selectionWire.timezone, 'Asia/Shanghai')
  } as const;
  const reports = list(root.reports).map(parseHomeGroup);
  unique(reports.map((item) => item.report.id));
  if (reports.length > 1 || (selection.mode === 'latest_fallback' && reports.length !== 1))
    invalid();
  return { selection, reports };
}

export function parseReportCardPageWire(value: unknown, expectedReportId: string): ReportCardPage {
  reportID(expectedReportId);
  const root = exact(value, ['items', 'next_cursor']);
  const items = list(root.items).map(parseCard);
  unique(items.map((item) => item.key));
  return { items, nextCursor: nullableCursor(root.next_cursor) };
}

export function parseReportLayerDetailWire(
  value: unknown,
  expectedReportId: string,
  expectedLayerKey: ReportLayerKey
): ReportLayerDetail {
  const root = exact(value, ['report', 'layer', 'related_industry_chains']);
  const report = parseSummary(root.report);
  const layer = parseLayer(root.layer);
  const relatedIndustryChains = list(root.related_industry_chains).map(parseRelatedChain);
  if (report.id !== reportID(expectedReportId) || layer.key !== expectedLayerKey) invalid();
  unique(relatedIndustryChains.map((item) => item.key));
  return { report, layer, relatedIndustryChains };
}

export function parseReportIndustryChainDetailWire(
  value: unknown,
  expectedReportId: string,
  expectedChainKey: string
): ReportIndustryChainDetail {
  const root = exact(value, ['report', 'industry_chain']);
  const report = parseSummary(root.report);
  const industryChain = parseIndustryChain(root.industry_chain);
  if (
    report.id !== reportID(expectedReportId) ||
    industryChain.key !== localKey(expectedChainKey)
  ) {
    invalid();
  }
  return { report, industryChain };
}

export function parseReportEvidenceListWire(
  value: unknown,
  expectedReportId: string,
  expectedScopeToken: string
): ReportEvidenceList {
  const root = exact(value, ['report_id', 'scope_token', 'items']);
  const reportId = reportID(root.report_id);
  const scopeToken = token(root.scope_token);
  if (reportId !== reportID(expectedReportId) || scopeToken !== token(expectedScopeToken))
    invalid();
  return { reportId, scopeToken, items: list(root.items).map(parseEvidence) };
}

function parseHomeGroup(value: unknown): ReportHomeGroup {
  const root = exact(value, ['report', 'cards', 'next_cursor']);
  const cards = list(root.cards).map(parseCard);
  unique(cards.map((item) => item.key));
  return { report: parseSummary(root.report), cards, nextCursor: nullableCursor(root.next_cursor) };
}

function parseSummary(value: unknown): ReportSummary {
  const root = exact(value, ['id', 'generated_at', 'published_at', 'industry_chain_count']);
  return {
    id: reportID(root.id),
    generatedAt: timestamp(root.generated_at),
    publishedAt: timestamp(root.published_at),
    industryChainCount: nonNegativeInteger(root.industry_chain_count)
  };
}

function parseCard(value: unknown): ReportCard {
  const root = exact(value, [
    'local_key',
    'kind',
    'detail_ref',
    'title',
    'subtitle',
    'conclusion',
    'result',
    'confidence',
    'time_window',
    'impact_items',
    'evidence_scope_token'
  ]);
  const kind = enumeration(root.kind, cardKinds);
  const detailRef = parseReference(root.detail_ref);
  if ((kind === 'industry_chain') !== (detailRef.type === 'industry_chain')) invalid();
  if (kind !== 'industry_chain' && (detailRef.type !== 'layer' || detailRef.localKey !== kind)) {
    invalid();
  }
  return {
    key: localKey(root.local_key),
    kind,
    detailRef: detailRef as ReportCard['detailRef'],
    title: text(root.title),
    subtitle: textOrEmpty(root.subtitle),
    conclusion: text(root.conclusion),
    result: coded(root.result),
    confidence: confidence(root.confidence),
    timeWindow: timeWindow(root.time_window),
    impactItems: list(root.impact_items).map(parseImpactItem),
    evidenceScopeToken: nullableToken(root.evidence_scope_token)
  };
}

function parseImpactItem(value: unknown) {
  const root = exact(value, [
    'ref',
    'name',
    'result',
    'conclusion_basis',
    'validation_status',
    'confidence',
    'time_window',
    'evidence_scope_token'
  ]);
  return {
    ref: parseReference(root.ref),
    name: text(root.name),
    result: coded(root.result),
    conclusionBasis: coded(root.conclusion_basis),
    validationStatus: coded(root.validation_status),
    confidence: confidence(root.confidence),
    timeWindow: timeWindow(root.time_window),
    evidenceScopeToken: nullableToken(root.evidence_scope_token)
  };
}

function parseLayer(value: unknown): ReportLayerDetailContent {
  const root = exact(value, [
    'key',
    'title',
    'conclusion',
    'result',
    'confidence',
    'time_window',
    'anchors',
    'reasoning_steps',
    'transmissions',
    'uncertainty',
    'evidence_scope_token'
  ]);
  return {
    key: enumeration(root.key, layerKeys),
    title: text(root.title),
    conclusion: text(root.conclusion),
    result: coded(root.result),
    confidence: confidence(root.confidence),
    timeWindow: timeWindow(root.time_window),
    anchors: list(root.anchors).map(parseAnchor),
    reasoningSteps: list(root.reasoning_steps).map(parseReasoningStep),
    transmissions: list(root.transmissions).map(parseTransmission),
    uncertainty: parseLayerUncertainty(root.uncertainty),
    evidenceScopeToken: nullableToken(root.evidence_scope_token)
  };
}

function parseAnchor(value: unknown): ReportAnchor {
  const root = exact(value, [
    'local_key',
    'name',
    'current_state',
    'result',
    'conclusion_basis',
    'validation_status',
    'reasoning',
    'time_window',
    'confidence',
    'evidence_scope_token'
  ]);
  return {
    key: localKey(root.local_key),
    name: text(root.name),
    currentState: text(root.current_state),
    result: coded(root.result),
    conclusionBasis: coded(root.conclusion_basis),
    validationStatus: coded(root.validation_status),
    reasoning: text(root.reasoning),
    timeWindow: timeWindow(root.time_window),
    confidence: confidence(root.confidence),
    evidenceScopeToken: nullableToken(root.evidence_scope_token)
  };
}

function parseReasoningStep(value: unknown): ReportReasoningStep {
  const root = exact(value, [
    'local_key',
    'input',
    'mechanism',
    'output',
    'confidence',
    'evidence_scope_token'
  ]);
  return {
    key: localKey(root.local_key),
    input: text(root.input),
    mechanism: text(root.mechanism),
    output: text(root.output),
    confidence: confidence(root.confidence),
    evidenceScopeToken: nullableToken(root.evidence_scope_token)
  };
}

function parseTransmission(value: unknown): ReportTransmissionPath {
  const root = exact(value, [
    'local_key',
    'source_conclusion',
    'targets',
    'logic',
    'kind',
    'confidence',
    'status'
  ]);
  return {
    key: localKey(root.local_key),
    sourceConclusion: text(root.source_conclusion),
    targets: list(root.targets).map(parseTransmissionTarget),
    logic: text(root.logic),
    kind: coded(root.kind),
    confidence: confidence(root.confidence),
    status: coded(root.status)
  };
}

function parseTransmissionTarget(value: unknown): ReportTransmissionTarget {
  const root = exact(value, ['ref', 'name', 'result']);
  return { ref: parseReference(root.ref), name: text(root.name), result: coded(root.result) };
}

function parseLayerUncertainty(value: unknown) {
  const root = exact(value, ['counterevidence', 'evidence_gap', 'boundary', 'reversal_condition']);
  return {
    counterevidence: nullableText(root.counterevidence),
    evidenceGap: nullableText(root.evidence_gap),
    boundary: nullableText(root.boundary),
    reversalCondition: nullableText(root.reversal_condition)
  };
}

function parseRelatedChain(value: unknown): ReportRelatedIndustryChain {
  const root = exact(value, ['local_key', 'name', 'result']);
  return { key: localKey(root.local_key), name: text(root.name), result: coded(root.result) };
}

function parseIndustryChain(value: unknown): ReportIndustryChainDetailContent {
  const root = exact(value, [
    'local_key',
    'name',
    'conclusion',
    'result',
    'confidence',
    'time_window',
    'path_summary',
    'accepted_hypothesis_summary',
    'topology_nodes',
    'nodes',
    'edges',
    'counterevidence_and_gap',
    'stop_condition',
    'evidence_scope_token'
  ]);
  const topologyNodes = list(root.topology_nodes).map(parseGraphNode);
  const topologyKeys = new Set(topologyNodes.map((node) => node.key));
  if (topologyKeys.size !== topologyNodes.length) invalid();
  const nodes = list(root.nodes).map(parseIndustryNode);
  const assessmentKeys = new Set(nodes.map((node) => node.key));
  if (assessmentKeys.size !== nodes.length || nodes.some((node) => !topologyKeys.has(node.key)))
    invalid();
  const edges = list(root.edges).map(parseGraphEdge);
  if (
    edges.some(
      (edge) => !topologyKeys.has(edge.fromNodeLocalKey) || !topologyKeys.has(edge.toNodeLocalKey)
    )
  )
    invalid();
  return {
    key: localKey(root.local_key),
    name: text(root.name),
    conclusion: text(root.conclusion),
    result: coded(root.result),
    confidence: confidence(root.confidence),
    timeWindow: timeWindow(root.time_window),
    pathSummary: nullableText(root.path_summary),
    acceptedHypothesisSummary: nullableText(root.accepted_hypothesis_summary),
    topologyNodes,
    nodes,
    edges,
    counterevidenceAndGap: nullableText(root.counterevidence_and_gap),
    stopCondition: nullableText(root.stop_condition),
    evidenceScopeToken: nullableToken(root.evidence_scope_token)
  };
}

function parseGraphNode(value: unknown) {
  const root = exact(value, ['local_key', 'name']);
  return { key: localKey(root.local_key), name: text(root.name) };
}

function parseIndustryNode(value: unknown): ReportIndustryChainNode {
  const root = exact(value, [
    'local_key',
    'name',
    'impact',
    'result',
    'conclusion_basis',
    'validation_status',
    'reasoning',
    'time_window',
    'confidence',
    'evidence_scope_token'
  ]);
  return {
    key: localKey(root.local_key),
    name: text(root.name),
    impact: text(root.impact),
    result: coded(root.result),
    conclusionBasis: coded(root.conclusion_basis),
    validationStatus: coded(root.validation_status),
    reasoning: text(root.reasoning),
    timeWindow: timeWindow(root.time_window),
    confidence: confidence(root.confidence),
    evidenceScopeToken: nullableToken(root.evidence_scope_token)
  };
}

function parseGraphEdge(value: unknown): ReportGraphEdge {
  const root = exact(value, ['from_node_local_key', 'to_node_local_key', 'relation_label']);
  return {
    fromNodeLocalKey: localKey(root.from_node_local_key),
    toNodeLocalKey: localKey(root.to_node_local_key),
    relationLabel: text(root.relation_label)
  };
}

function parseEvidence(value: unknown): ReportEvidence {
  const root = exact(value, ['published_at', 'summary', 'keywords']);
  return {
    publishedAt: root.published_at === null ? null : timestamp(root.published_at),
    summary: text(root.summary),
    keywords: list(root.keywords).map(text)
  };
}

function parseReference(value: unknown): ReportReference {
  const root = exact(value, ['type', 'local_key']);
  return { type: enumeration(root.type, referenceTypes), localKey: localKey(root.local_key) };
}

function coded(value: unknown): ReportCodedLabel {
  const root = exact(value, ['code', 'label']);
  return { code: text(root.code), label: text(root.label) };
}

function confidence(value: unknown): ReportConfidence {
  return coded(value);
}

function timeWindow(value: unknown): ReportTimeWindow {
  return coded(value);
}

function exact(value: unknown, keys: readonly string[]): RecordValue {
  if (typeof value !== 'object' || value === null || Array.isArray(value)) invalid();
  const record = value as RecordValue;
  const actual = Object.keys(record);
  if (
    actual.length !== keys.length ||
    keys.some((key) => !Object.prototype.hasOwnProperty.call(record, key))
  ) {
    invalid();
  }
  return record;
}

function list(value: unknown): unknown[] {
  if (!Array.isArray(value)) invalid();
  return value;
}

function text(value: unknown): string {
  if (typeof value !== 'string' || value.trim() !== value || value.length === 0) invalid();
  return value;
}

function textOrEmpty(value: unknown): string {
  return value === '' ? '' : text(value);
}

function nullableText(value: unknown): string | null {
  return value === null ? null : text(value);
}

function localKey(value: unknown): string {
  const parsed = text(value);
  if (!localKeyPattern.test(parsed)) invalid();
  return parsed;
}

function reportID(value: unknown): string {
  const parsed = text(value);
  if (!reportIDPattern.test(parsed)) invalid();
  return parsed;
}

function token(value: unknown): string {
  const parsed = text(value);
  if (!scopeTokenPattern.test(parsed)) invalid();
  return parsed;
}

function nullableToken(value: unknown): string | null {
  return value === null ? null : token(value);
}

function nullableCursor(value: unknown): string | null {
  if (value === null) return null;
  const parsed = text(value);
  if (parsed.length > 2048) invalid();
  return parsed;
}

function timestamp(value: unknown): string {
  const parsed = text(value);
  if (!Number.isFinite(Date.parse(parsed))) invalid();
  return parsed;
}

function nonNegativeInteger(value: unknown): number {
  if (typeof value !== 'number' || !Number.isSafeInteger(value) || value < 0) invalid();
  return value;
}

function enumeration<T extends string>(value: unknown, values: readonly T[]): T {
  const parsed = text(value);
  if (!values.includes(parsed as T)) invalid();
  return parsed as T;
}

function literal<T extends string>(value: unknown, expected: T): T {
  if (value !== expected) invalid();
  return expected;
}

function match(value: unknown, pattern: RegExp): string {
  const parsed = text(value);
  if (!pattern.test(parsed)) invalid();
  return parsed;
}

function unique(values: string[]): void {
  if (new Set(values).size !== values.length) invalid();
}

function invalid(): never {
  throw new Error('invalid Report wire response');
}
