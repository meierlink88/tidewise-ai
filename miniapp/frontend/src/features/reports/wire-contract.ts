import type {
  ReportAnchor,
  ReportCandidateMechanism,
  ReportCard,
  ReportCheckpoint,
  ReportConfidence,
  ReportDownwardTransmission,
  ReportEvidence,
  ReportEvidenceList,
  ReportEvidenceScope,
  ReportEvidenceScopeType,
  ReportGraphEdge,
  ReportHome,
  ReportHomeGroup,
  ReportIndustryChainDetail,
  ReportIndustryChainDetailContent,
  ReportIndustryChainNode,
  ReportIndustryChainUncertainty,
  ReportLayerDetail,
  ReportLayerDetailContent,
  ReportLayerKey,
  ReportLayerUncertainty,
  ReportNature,
  ReportNatureCode,
  ReportReasoningStep,
  ReportReference,
  ReportReferenceType,
  ReportRelatedIndustryChain,
  ReportResult,
  ReportResultCode,
  ReportSummary,
  ReportTransmissionPath,
  ReportTransmissionTarget
} from './contract';

type RecordValue = Record<string, unknown>;

const resultCodes = ['warming', 'cooling', 'diverging', 'pending'] as const;
const resultLabels: Record<ReportResultCode, ReportResult['label']> = {
  warming: '升温',
  cooling: '降温',
  diverging: '分化',
  pending: '待验证'
};
const natureCodes = ['direct_evidence', 'reasoning_hypothesis', 'pending_validation'] as const;
const natureLabels: Record<ReportNatureCode, ReportNature['label']> = {
  direct_evidence: '直接证据',
  reasoning_hypothesis: '推理假设',
  pending_validation: '待验证'
};
const layerKeys = ['geopolitics', 'macroeconomics'] as const;
const cardKinds = ['geopolitics', 'macroeconomics', 'industry_chain'] as const;
const referenceTypes = ['layer', 'anchor', 'industry_chain', 'industry_chain_node'] as const;
const evidenceScopeTypes = [
  'report_card',
  'layer',
  'anchor',
  'reasoning_step',
  'transmission_path',
  'candidate_mechanism',
  'industry_chain',
  'industry_chain_node'
] as const;
const selectionModes = ['today', 'latest_fallback'] as const;
const reportIDPattern =
  /^RPT[0-9a-f]{8}-[0-9a-f]{4}-[1-5][0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$/;
const localKeyPattern = /^[a-z0-9][a-z0-9._-]{0,127}$/;
const utcTimestampPattern = /^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}(?:\.\d{1,9})?Z$/;
const datePattern = /^\d{4}-\d{2}-\d{2}$/;
const displayTextMaxCodePoints = 10_000;

export function parseReportHomeWire(value: unknown): ReportHome {
  const root = exactRecord(value, ['selection', 'reports']);
  const selectionWire = exactRecord(root.selection, ['mode', 'date', 'timezone']);
  const selection = {
    mode: enumValue(selectionWire.mode, selectionModes),
    date: dateString(selectionWire.date),
    timezone: literal(selectionWire.timezone, 'Asia/Shanghai')
  } as const;
  const reports = array(root.reports).map(parseHomeGroup);

  unique(reports.map((group) => group.report.id));
  validateSummaryOrder(reports.map((group) => group.report));
  if (selection.mode === 'latest_fallback' && reports.length !== 1) invalid();
  if (selection.mode === 'today') {
    for (const group of reports) {
      if (shanghaiDate(group.report.publishedAt) !== selection.date) invalid();
    }
  }
  return { selection, reports };
}

export function parseReportLayerDetailWire(
  value: unknown,
  expectedReportId: string,
  expectedLayerKey: ReportLayerKey
): ReportLayerDetail {
  reportID(expectedReportId);
  const root = exactRecord(value, ['report', 'layer', 'related_industry_chains']);
  const report = parseReportSummary(root.report);
  const layer = parseLayer(root.layer);
  const relatedIndustryChains = array(root.related_industry_chains).map(parseRelatedChain);

  if (report.id !== expectedReportId || layer.key !== expectedLayerKey) invalid();
  unique(relatedIndustryChains.map((chain) => chain.key));
  if (
    layer.relatedChainKeys.length !== relatedIndustryChains.length ||
    layer.relatedChainKeys.some((key, index) => key !== relatedIndustryChains[index].key)
  ) {
    invalid();
  }
  return { report, layer, relatedIndustryChains };
}

export function parseReportIndustryChainDetailWire(
  value: unknown,
  expectedReportId: string,
  expectedChainKey: string
): ReportIndustryChainDetail {
  reportID(expectedReportId);
  localKey(expectedChainKey);
  const root = exactRecord(value, ['report', 'industry_chain']);
  const report = parseReportSummary(root.report);
  const industryChain = parseIndustryChain(root.industry_chain);
  if (report.id !== expectedReportId || industryChain.key !== expectedChainKey) invalid();
  return { report, industryChain };
}

export function parseReportEvidenceListWire(
  value: unknown,
  expectedReportId: string,
  expectedScope: ReportEvidenceScope
): ReportEvidenceList {
  reportID(expectedReportId);
  validateExpectedScope(expectedScope);
  const root = exactRecord(value, ['report_id', 'scope', 'items']);
  const reportId = reportID(root.report_id);
  const scope = parseScope(root.scope);
  const items = array(root.items).map(parseEvidence);
  if (
    reportId !== expectedReportId ||
    scope.type !== expectedScope.type ||
    scope.key !== expectedScope.key
  ) {
    invalid();
  }
  return { reportId, scope, items };
}

function parseHomeGroup(value: unknown): ReportHomeGroup {
  const group = exactRecord(value, ['report', 'cards', 'company']);
  const report = parseReportSummary(group.report);
  const cards = parseOrderedArray(group.cards, parseCard);
  const companyWire = exactRecord(group.company, [
    'key',
    'display_order',
    'title',
    'published',
    'boundary'
  ]);
  if (
    companyWire.key !== 'company' ||
    companyWire.display_order !== 4 ||
    companyWire.published !== false
  ) {
    invalid();
  }

  unique(cards.map((card) => referenceIdentity(card.detailRef)));
  if (
    cards.filter((card) => card.kind === 'geopolitics').length !== 1 ||
    cards.filter((card) => card.kind === 'macroeconomics').length !== 1
  ) {
    invalid();
  }

  return {
    report,
    cards,
    company: {
      key: 'company',
      displayOrder: 4,
      title: displayText(companyWire.title),
      published: false,
      boundary: displayText(companyWire.boundary)
    }
  };
}

function parseCard(value: unknown): ReportCard {
  const card = exactRecord(value, [
    'key',
    'kind',
    'display_order',
    'detail_ref',
    'title',
    'subtitle',
    'conclusion',
    'result',
    'confidence',
    'time_window',
    'impact_items',
    'has_evidence'
  ]);
  const kind = enumValue(card.kind, cardKinds);
  const detailRef = parseReference(card.detail_ref);
  if (
    (kind === 'geopolitics' && (detailRef.type !== 'layer' || detailRef.key !== 'geopolitics')) ||
    (kind === 'macroeconomics' &&
      (detailRef.type !== 'layer' || detailRef.key !== 'macroeconomics')) ||
    (kind === 'industry_chain' && detailRef.type !== 'industry_chain')
  ) {
    invalid();
  }
  const impactItems = array(card.impact_items).map((item) => parseImpactItem(item, kind));
  if (impactItems.length === 0) invalid();
  unique(impactItems.map((item) => referenceIdentity(item.ref)));
  return {
    key: localKey(card.key),
    kind,
    displayOrder: positiveInteger(card.display_order),
    detailRef: detailRef as ReportCard['detailRef'],
    title: displayText(card.title),
    subtitle: displayText(card.subtitle),
    conclusion: displayText(card.conclusion),
    result: parseResult(card.result),
    confidence: parseConfidence(card.confidence),
    timeWindow: displayText(card.time_window),
    impactItems,
    hasEvidence: boolean(card.has_evidence)
  };
}

function parseImpactItem(value: unknown, kind: ReportCard['kind']) {
  const item = exactRecord(value, [
    'ref',
    'name',
    'result',
    'confidence',
    'time_window',
    'has_evidence'
  ]);
  const ref = parseReference(item.ref);
  const expectedType = kind === 'industry_chain' ? 'industry_chain_node' : 'anchor';
  if (ref.type !== expectedType) invalid();
  return {
    ref,
    name: displayText(item.name),
    result: parseResult(item.result),
    confidence: parseConfidence(item.confidence),
    timeWindow: displayText(item.time_window),
    hasEvidence: boolean(item.has_evidence)
  };
}

function parseReportSummary(value: unknown): ReportSummary {
  const summary = exactRecord(value, ['id', 'title', 'generated_at', 'published_at']);
  return {
    id: reportID(summary.id),
    title: displayText(summary.title),
    generatedAt: utcTimestamp(summary.generated_at),
    publishedAt: utcTimestamp(summary.published_at)
  };
}

function parseLayer(value: unknown): ReportLayerDetailContent & {
  relatedAnchorKeys: string[];
  relatedChainKeys: string[];
} {
  const layer = exactRecord(value, [
    'key',
    'display_order',
    'title',
    'conclusion',
    'result',
    'confidence',
    'time_window',
    'anchors',
    'reasoning_steps',
    'related_anchor_keys',
    'related_chain_keys',
    'downward_transmission',
    'uncertainty',
    'scope',
    'has_evidence'
  ]);
  const key = enumValue<ReportLayerKey>(layer.key, layerKeys);
  const displayOrder = positiveInteger(layer.display_order);
  if (
    (key === 'geopolitics' && displayOrder !== 1) ||
    (key === 'macroeconomics' && displayOrder !== 2)
  ) {
    invalid();
  }
  return {
    key,
    displayOrder,
    scope: parseScope(layer.scope, 'layer', key),
    title: displayText(layer.title),
    conclusion: displayText(layer.conclusion),
    result: parseResult(layer.result),
    confidence: parseConfidence(layer.confidence),
    timeWindow: displayText(layer.time_window),
    anchors: parseOrderedArray(layer.anchors, parseAnchor),
    reasoningSteps: parseOrderedArray(layer.reasoning_steps, parseReasoningStep),
    relatedAnchorKeys: localKeyArray(layer.related_anchor_keys),
    relatedChainKeys: localKeyArray(layer.related_chain_keys),
    downwardTransmission: parseDownwardTransmission(layer.downward_transmission),
    uncertainty: parseLayerUncertainty(layer.uncertainty),
    hasEvidence: boolean(layer.has_evidence)
  };
}

function parseAnchor(value: unknown): ReportAnchor {
  const anchor = exactRecord(value, [
    'key',
    'display_order',
    'name',
    'current_state',
    'result',
    'nature',
    'reasoning',
    'time_window',
    'confidence',
    'scope',
    'has_evidence'
  ]);
  const key = localKey(anchor.key);
  return {
    key,
    displayOrder: positiveInteger(anchor.display_order),
    name: displayText(anchor.name),
    currentState: displayText(anchor.current_state),
    result: parseResult(anchor.result),
    nature: parseNature(anchor.nature),
    reasoning: displayText(anchor.reasoning),
    timeWindow: displayText(anchor.time_window),
    confidence: parseConfidence(anchor.confidence),
    scope: parseScope(anchor.scope, 'anchor', key),
    hasEvidence: boolean(anchor.has_evidence)
  };
}

function parseReasoningStep(value: unknown): ReportReasoningStep {
  const step = exactRecord(value, [
    'key',
    'display_order',
    'input',
    'mechanism',
    'output',
    'type',
    'confidence',
    'scope',
    'has_evidence'
  ]);
  const key = localKey(step.key);
  return {
    key,
    displayOrder: positiveInteger(step.display_order),
    input: displayText(step.input),
    mechanism: displayText(step.mechanism),
    output: displayText(step.output),
    type: displayText(step.type),
    confidence: parseConfidence(step.confidence),
    scope: parseScope(step.scope, 'reasoning_step', key),
    hasEvidence: boolean(step.has_evidence)
  };
}

function parseDownwardTransmission(value: unknown): ReportDownwardTransmission {
  const transmission = exactRecord(value, [
    'summary',
    'published_paths',
    'candidate_mechanisms',
    'boundary_notes'
  ]);
  return {
    summary: displayText(transmission.summary),
    publishedPaths: parseOrderedArray(transmission.published_paths, parseTransmissionPath),
    candidateMechanisms: parseOrderedArray(
      transmission.candidate_mechanisms,
      parseCandidateMechanism
    ),
    boundaryNotes: displayTextArray(transmission.boundary_notes, 0)
  };
}

function parseTransmissionPath(value: unknown): ReportTransmissionPath {
  const path = exactRecord(value, [
    'key',
    'display_order',
    'source_conclusion',
    'target_refs',
    'logic',
    'relation_nature',
    'evidence_role',
    'confidence',
    'status',
    'scope',
    'has_evidence'
  ]);
  const key = localKey(path.key);
  const targetRefs = array(path.target_refs).map(parseTransmissionTarget);
  if (targetRefs.length === 0) invalid();
  unique(targetRefs.map((target) => referenceIdentity(target.ref)));
  return {
    key,
    displayOrder: positiveInteger(path.display_order),
    sourceConclusion: displayText(path.source_conclusion),
    targetRefs,
    logic: displayText(path.logic),
    relationNature: displayText(path.relation_nature),
    evidenceRole: displayText(path.evidence_role),
    confidence: parseConfidence(path.confidence),
    status: displayText(path.status),
    scope: parseScope(path.scope, 'transmission_path', key),
    hasEvidence: boolean(path.has_evidence)
  };
}

function parseTransmissionTarget(value: unknown): ReportTransmissionTarget {
  const target = exactRecord(value, ['ref', 'label', 'result']);
  return {
    ref: parseReference(target.ref),
    label: displayText(target.label),
    result: parseResult(target.result)
  };
}

function parseCandidateMechanism(value: unknown): ReportCandidateMechanism {
  const candidate = exactRecord(value, [
    'key',
    'display_order',
    'mechanism',
    'evidence_gap',
    'confidence',
    'scope',
    'has_evidence'
  ]);
  const key = localKey(candidate.key);
  return {
    key,
    displayOrder: positiveInteger(candidate.display_order),
    mechanism: displayText(candidate.mechanism),
    evidenceGap: nullableDisplayText(candidate.evidence_gap),
    confidence: parseConfidence(candidate.confidence),
    scope: parseScope(candidate.scope, 'candidate_mechanism', key),
    hasEvidence: boolean(candidate.has_evidence)
  };
}

function parseLayerUncertainty(value: unknown): ReportLayerUncertainty {
  const uncertainty = exactRecord(value, [
    'counterevidence',
    'evidence_gap',
    'boundary',
    'reversal_condition',
    'checkpoints'
  ]);
  return {
    counterevidence: nullableDisplayText(uncertainty.counterevidence),
    evidenceGap: nullableDisplayText(uncertainty.evidence_gap),
    boundary: nullableDisplayText(uncertainty.boundary),
    reversalCondition: nullableDisplayText(uncertainty.reversal_condition),
    checkpoints: parseOrderedArray(uncertainty.checkpoints, parseCheckpoint)
  };
}

function parseCheckpoint(value: unknown): ReportCheckpoint {
  const checkpoint = exactRecord(value, ['key', 'display_order', 'summary']);
  return {
    key: localKey(checkpoint.key),
    displayOrder: positiveInteger(checkpoint.display_order),
    summary: displayText(checkpoint.summary)
  };
}

function parseRelatedChain(value: unknown): ReportRelatedIndustryChain {
  const chain = exactRecord(value, [
    'key',
    'display_order',
    'name',
    'conclusion',
    'status',
    'result',
    'confidence',
    'time_window',
    'scope',
    'has_evidence'
  ]);
  const key = localKey(chain.key);
  parseScope(chain.scope, 'industry_chain', key);
  parseConfidence(chain.confidence);
  displayText(chain.conclusion);
  displayText(chain.status);
  displayText(chain.time_window);
  boolean(chain.has_evidence);
  return {
    key,
    displayOrder: positiveInteger(chain.display_order),
    name: displayText(chain.name),
    result: parseResult(chain.result),
    detailRef: { type: 'industry_chain', key }
  };
}

function parseIndustryChain(value: unknown): ReportIndustryChainDetailContent {
  const chain = exactRecord(value, [
    'key',
    'claim_key',
    'display_order',
    'name',
    'conclusion',
    'status',
    'result',
    'confidence',
    'time_window',
    'path_summary',
    'accepted_hypothesis_summary',
    'nodes',
    'edges',
    'uncertainty',
    'scope',
    'has_evidence'
  ]);
  const key = localKey(chain.key);
  const nodes = parseOrderedArray(chain.nodes, parseIndustryChainNode);
  const edges = parseOrderedArray(chain.edges, parseGraphEdge);
  const nodeKeys = new Set(nodes.map((node) => node.key));
  for (const edge of edges) {
    if (!nodeKeys.has(edge.fromNodeKey) || !nodeKeys.has(edge.toNodeKey)) invalid();
  }
  return {
    key,
    claimKey: localKey(chain.claim_key),
    displayOrder: positiveInteger(chain.display_order),
    name: displayText(chain.name),
    conclusion: displayText(chain.conclusion),
    status: displayText(chain.status),
    result: parseResult(chain.result),
    confidence: parseConfidence(chain.confidence),
    timeWindow: displayText(chain.time_window),
    pathSummary: nullableDisplayText(chain.path_summary),
    acceptedHypothesisSummary: nullableDisplayText(chain.accepted_hypothesis_summary),
    nodes,
    edges,
    uncertainty: parseIndustryChainUncertainty(chain.uncertainty),
    scope: parseScope(chain.scope, 'industry_chain', key),
    hasEvidence: boolean(chain.has_evidence)
  };
}

function parseIndustryChainNode(value: unknown): ReportIndustryChainNode {
  const node = exactRecord(value, [
    'key',
    'display_order',
    'name',
    'impact',
    'result',
    'nature',
    'reasoning',
    'time_window',
    'confidence',
    'scope',
    'has_evidence'
  ]);
  const key = localKey(node.key);
  return {
    key,
    displayOrder: positiveInteger(node.display_order),
    name: displayText(node.name),
    impact: displayText(node.impact),
    result: parseResult(node.result),
    nature: parseNature(node.nature),
    reasoning: displayText(node.reasoning),
    timeWindow: displayText(node.time_window),
    confidence: parseConfidence(node.confidence),
    scope: parseScope(node.scope, 'industry_chain_node', key),
    hasEvidence: boolean(node.has_evidence)
  };
}

function parseGraphEdge(value: unknown): ReportGraphEdge {
  const edge = exactRecord(value, [
    'key',
    'display_order',
    'from_node_key',
    'to_node_key',
    'relation_label'
  ]);
  const fromNodeKey = localKey(edge.from_node_key);
  const toNodeKey = localKey(edge.to_node_key);
  if (fromNodeKey === toNodeKey) invalid();
  return {
    key: localKey(edge.key),
    displayOrder: positiveInteger(edge.display_order),
    fromNodeKey,
    toNodeKey,
    relationLabel: displayText(edge.relation_label)
  };
}

function parseIndustryChainUncertainty(value: unknown): ReportIndustryChainUncertainty {
  const uncertainty = exactRecord(value, [
    'counterevidence_and_gap',
    'stop_condition',
    'checkpoints'
  ]);
  return {
    counterevidenceAndGap: nullableDisplayText(uncertainty.counterevidence_and_gap),
    stopCondition: nullableDisplayText(uncertainty.stop_condition),
    checkpoints: parseOrderedArray(uncertainty.checkpoints, parseCheckpoint)
  };
}

function parseReference(value: unknown): ReportReference {
  const ref = exactRecord(value, ['type', 'key']);
  const type = enumValue<ReportReferenceType>(ref.type, referenceTypes);
  const key = localKey(ref.key);
  if (type === 'layer' && !layerKeys.includes(key as ReportLayerKey)) invalid();
  return { type, key };
}

function parseScope(
  value: unknown,
  expectedType?: ReportEvidenceScopeType,
  expectedKey?: string
): ReportEvidenceScope {
  const scope = exactRecord(value, ['type', 'key']);
  const result = {
    type: enumValue<ReportEvidenceScopeType>(scope.type, evidenceScopeTypes),
    key: localKey(scope.key)
  };
  if (result.type === 'layer' && !layerKeys.includes(result.key as ReportLayerKey)) invalid();
  if (
    (expectedType !== undefined && result.type !== expectedType) ||
    (expectedKey !== undefined && result.key !== expectedKey)
  ) {
    invalid();
  }
  return result;
}

function validateExpectedScope(scope: ReportEvidenceScope): void {
  if (!evidenceScopeTypes.includes(scope.type)) invalid();
  const key = localKey(scope.key);
  if (scope.type === 'layer' && !layerKeys.includes(key as ReportLayerKey)) invalid();
}

function parseResult(value: unknown): ReportResult {
  const result = exactRecord(value, ['code', 'label']);
  const code = enumValue<ReportResultCode>(result.code, resultCodes);
  if (result.label !== resultLabels[code]) invalid();
  return { code, label: resultLabels[code] };
}

function parseNature(value: unknown): ReportNature {
  const nature = exactRecord(value, ['code', 'label']);
  const code = enumValue<ReportNatureCode>(nature.code, natureCodes);
  if (nature.label !== natureLabels[code]) invalid();
  return { code, label: natureLabels[code] };
}

function parseConfidence(value: unknown): ReportConfidence {
  const confidence = exactRecord(value, ['label', 'score']);
  const score = confidence.score;
  if (
    score !== null &&
    (typeof score !== 'number' || !Number.isFinite(score) || score < 0 || score > 1)
  ) {
    invalid();
  }
  return {
    label: displayText(confidence.label),
    score: score as number | null
  };
}

function parseEvidence(value: unknown): ReportEvidence {
  const evidence = exactRecord(value, ['published_at', 'summary', 'keywords']);
  return {
    publishedAt: evidence.published_at === null ? null : utcTimestamp(evidence.published_at),
    summary: displayText(evidence.summary),
    keywords: displayTextArray(evidence.keywords, 50)
  };
}

function parseOrderedArray<T extends { key: string; displayOrder: number }>(
  value: unknown,
  parser: (item: unknown) => T
): T[] {
  const items = array(value).map(parser);
  for (let index = 0; index < items.length; index += 1) {
    if (items[index].displayOrder !== index + 1) invalid();
  }
  unique(items.map((item) => item.key));
  return items;
}

function localKeyArray(value: unknown): string[] {
  const values = array(value).map(localKey);
  unique(values);
  return values;
}

function displayTextArray(value: unknown, maxItems: number): string[] {
  const values = array(value);
  if (maxItems > 0 && values.length > maxItems) invalid();
  const texts = values.map(displayText);
  unique(texts);
  return texts;
}

function validateSummaryOrder(summaries: ReportSummary[]): void {
  for (let index = 1; index < summaries.length; index += 1) {
    const previous = summaries[index - 1];
    const current = summaries[index];
    const previousTime = Date.parse(previous.publishedAt);
    const currentTime = Date.parse(current.publishedAt);
    if (
      currentTime > previousTime ||
      (currentTime === previousTime && current.id.localeCompare(previous.id) < 0)
    ) {
      invalid();
    }
  }
}

function shanghaiDate(timestamp: string): string {
  return new Date(Date.parse(timestamp) + 8 * 60 * 60 * 1_000).toISOString().slice(0, 10);
}

function exactRecord(value: unknown, keys: readonly string[]): RecordValue {
  const result = record(value);
  const actualKeys = Object.keys(result);
  const expected = new Set(keys);
  if (actualKeys.length !== keys.length || actualKeys.some((key) => !expected.has(key))) {
    invalid();
  }
  return result;
}

function record(value: unknown): RecordValue {
  if (typeof value !== 'object' || value === null || Array.isArray(value)) invalid();
  return value as RecordValue;
}

function array(value: unknown): unknown[] {
  if (!Array.isArray(value)) invalid();
  return value;
}

function displayText(value: unknown): string {
  if (
    typeof value !== 'string' ||
    value === '' ||
    value.trim() !== value ||
    Array.from(value).length > displayTextMaxCodePoints
  ) {
    invalid();
  }
  return value;
}

function nullableDisplayText(value: unknown): string | null {
  return value === null ? null : displayText(value);
}

function localKey(value: unknown): string {
  if (typeof value !== 'string' || !localKeyPattern.test(value)) invalid();
  return value;
}

function reportID(value: unknown): string {
  if (typeof value !== 'string' || !reportIDPattern.test(value)) invalid();
  return value;
}

function utcTimestamp(value: unknown): string {
  if (typeof value !== 'string' || !utcTimestampPattern.test(value)) invalid();
  const parsed = Date.parse(value);
  if (!Number.isFinite(parsed)) invalid();
  return value;
}

function dateString(value: unknown): string {
  if (typeof value !== 'string' || !datePattern.test(value)) invalid();
  const [year, month, day] = value.split('-').map(Number);
  const parsed = new Date(Date.UTC(year, month - 1, day));
  if (parsed.toISOString().slice(0, 10) !== value) invalid();
  return value;
}

function positiveInteger(value: unknown): number {
  if (!Number.isInteger(value) || (value as number) < 1) invalid();
  return value as number;
}

function boolean(value: unknown): boolean {
  if (typeof value !== 'boolean') invalid();
  return value;
}

function literal<T extends string | number | boolean>(value: unknown, expected: T): T {
  if (value !== expected) invalid();
  return expected;
}

function enumValue<T extends string>(value: unknown, values: readonly T[]): T {
  if (typeof value !== 'string' || !values.includes(value as T)) invalid();
  return value as T;
}

function referenceIdentity(ref: ReportReference): string {
  return `${ref.type}:${ref.key}`;
}

function unique(values: string[]): void {
  if (new Set(values).size !== values.length) invalid();
}

function invalid(): never {
  throw new Error('invalid Report wire contract');
}
