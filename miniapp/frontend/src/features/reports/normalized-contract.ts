import { ReportError } from './contract';

export const analysisKinds = [
  'geopolitical_stories',
  'macroeconomic_stories',
  'concept_analyses',
  'industry_chain_analyses'
] as const;
export type AnalysisKind = (typeof analysisKinds)[number];
export const analysisLabels: Record<AnalysisKind, string> = {
  geopolitical_stories: '地缘政治',
  macroeconomic_stories: '宏观经济',
  concept_analyses: '产业链',
  industry_chain_analyses: '产业链'
};
export interface EvidenceScope {
  evidence_scope_token: string | null;
  evidence_count: number;
}
export type JudgmentOrigin = 'direct' | 'inferred';
export interface ReasoningSources {
  signal_ids: string[];
  event_ids: string[];
  upstream_refs: { entity_id: string; local_key: string; mechanism?: string; condition?: string }[];
}
export interface VariableSignal extends EvidenceScope {
  variable_id: string;
  variable_name: string;
  signal_id: string;
  signal: string;
  source_direction: 'UP' | 'DOWN' | 'STABLE' | 'MIXED' | 'UNKNOWN';
  adoption: 'adopted' | 'qualified';
  qualification: string;
  event_ids: string[];
}
export interface Provenance {
  judgment_origin?: JudgmentOrigin;
  reasoning_sources?: ReasoningSources;
  variable_signals?: VariableSignal[];
}
export interface Assessment extends EvidenceScope {
  weight_delta_pp?: number;
  adjustment_purpose?: string;
  conclusion: string;
  direction: 'warming' | 'cooling' | 'diverging' | 'pending';
  conclusion_basis: 'reasoning_hypothesis' | 'observation_only';
  validation_status: string;
  confidence: 'low' | 'medium' | 'high' | null;
  forecast_window: {
    kind: string;
    description: string;
    start_at: string | null;
    end_at: string | null;
  };
  scope: string;
  conditions: string[];
  follow_up: string[];
  transmission_logic: string;
}
export interface Claim extends EvidenceScope {
  text: string;
  basis: string;
}
export interface Objections {
  summary: string;
  counterevidence: Claim[];
  buffers: Claim[];
  evidence_gaps: string[];
  scope_limits: string[];
  counterevidence_status: string;
}
export interface MacroImpact extends Provenance {
  local_key: string;
  source_id: string;
  name: string;
  assessment: Assessment;
  objections: Objections;
}
export interface NodeImpact extends MacroImpact {
  node_local_key: string;
}
export interface EmptyState {
  code: string;
  reason: string;
  follow_up: string[];
}
export interface ChainHeader {
  judgment_origin?: JudgmentOrigin;
  local_key: string;
  source_id: string;
  name: string;
  assessment: Assessment;
  empty_state: EmptyState | null;
}
export interface AnalysisChain extends ChainHeader, Provenance {
  reasoning_summary: { logic: string; support: Claim; objections: Objections };
  graph: {
    scope?: 'assessed_nodes_only';
    nodes: { local_key: string; source_id: string; name: string }[];
    edges: { from_node_local_key: string; to_node_local_key: string; relation_label: string }[];
  };
  affected_nodes: NodeImpact[];
}
export interface AnchorRef {
  reasoning_local_key?: string;
  target_type: string;
  local_key: string;
  chain_local_key: string | null;
}
export interface AnalysisSummary {
  schema_version: 'report-publication/v4' | 'report-publication/v5' | 'report-publication/v6';
  judgment_origin?: JudgmentOrigin;
  local_key: string;
  source_id: string;
  title: string;
  summary: EvidenceScope & {
    judgment?: string;
    conclusion: string;
    transmission_logic: string;
    impact_assessment: EvidenceScope & { level: string; rationale: string };
    affected_refs: AnchorRef[];
  };
  affected_anchors: {
    judgment_origin?: JudgmentOrigin;
    reference: AnchorRef;
    source_id: string;
    name: string;
    assessment: Assessment;
  }[];
  chain_count: number;
  reasoning_count?: number;
}
export interface AnalysisPage {
  items: AnalysisSummary[];
  next_cursor: string | null;
}
export interface AnalysisGroup extends AnalysisPage {
  kind: AnalysisKind;
}
export interface AnalysisDetail extends Provenance {
  reasonings?: UnifiedReasoning[];
  published_at?: string;
  companies?: MacroImpact[];
  summary: AnalysisSummary;
  macro_impacts: MacroImpact[];
  industry_chains: ChainHeader[];
}

const obj = (v: unknown): Record<string, unknown> => {
  if (!v || typeof v !== 'object' || Array.isArray(v)) fail();
  return v as Record<string, unknown>;
};
const fail = (): never => {
  throw new ReportError('invalidResponse');
};
const str = (v: unknown): string => (typeof v === 'string' && v.length <= 16000 ? v : fail());
const key = (v: unknown): string =>
  /^[A-Za-z0-9][A-Za-z0-9._-]{0,127}$/.test(str(v)) ? str(v) : fail();
const num = (v: unknown): number =>
  typeof v === 'number' && Number.isSafeInteger(v) && v >= 0 ? v : fail();
const list = <T>(v: unknown, parse: (v: unknown) => T): T[] =>
  Array.isArray(v) ? v.map(parse) : fail();
const nullable = <T>(v: unknown, parse: (v: unknown) => T): T | null =>
  v === null ? null : parse(v);
const choice = <T extends string>(v: unknown, choices: readonly T[]): T =>
  choices.includes(v as T) ? (v as T) : fail();
function unique(keys: string[]) {
  if (new Set(keys).size !== keys.length) fail();
}
function scope(v: Record<string, unknown>): EvidenceScope {
  const token = nullable(v.evidence_scope_token, str),
    count = num(v.evidence_count);
  if (
    (count === 0) !== (token === null) ||
    (token &&
      !/^RPE[0-9a-f]{8}-[0-9a-f]{4}-[1-5][0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$/.test(token))
  )
    fail();
  return { evidence_scope_token: token, evidence_count: count };
}
const finite = (v: unknown): number => (typeof v === 'number' && Number.isFinite(v) ? v : fail());
function assessment(value: unknown): Assessment {
  const v = obj(value),
    w = obj(v.forecast_window);
  return {
    ...scope(v),
    ...(v.weight_delta_pp !== undefined ? { weight_delta_pp: finite(v.weight_delta_pp) } : {}),
    ...(v.adjustment_purpose !== undefined
      ? { adjustment_purpose: str(v.adjustment_purpose) }
      : {}),
    conclusion: str(v.conclusion),
    direction: choice(v.direction, ['warming', 'cooling', 'diverging', 'pending']),
    conclusion_basis: choice(v.conclusion_basis, ['reasoning_hypothesis', 'observation_only']),
    validation_status: choice(v.validation_status, ['pending_validation', 'insufficient_evidence']),
    confidence: nullable(v.confidence, (confidence) =>
      choice(confidence, ['low', 'medium', 'high'])
    ),
    forecast_window: {
      kind: choice(w.kind, ['relative', 'calendar', 'stage', 'mixed', 'not_applicable']),
      description: str(w.description),
      start_at: nullable(w.start_at, str),
      end_at: nullable(w.end_at, str)
    },
    scope: str(v.scope),
    conditions: list(v.conditions, str),
    follow_up: list(v.follow_up, str),
    transmission_logic: str(v.transmission_logic)
  };
}
function claim(value: unknown): Claim {
  const v = obj(value);
  return {
    ...scope(v),
    text: str(v.text),
    basis: choice(v.basis, ['source_fact', 'inference', 'hypothetical_buffer'])
  };
}
function objections(value: unknown): Objections {
  const v = obj(value);
  return {
    summary: str(v.summary),
    counterevidence: list(v.counterevidence, claim),
    buffers: list(v.buffers, claim),
    evidence_gaps: list(v.evidence_gaps, str),
    scope_limits: list(v.scope_limits, str),
    counterevidence_status: choice(v.counterevidence_status, ['identified', 'none_identified'])
  };
}
function macro(value: unknown): MacroImpact {
  const v = obj(value);
  return {
    ...provenance(v),
    local_key: key(v.local_key),
    source_id: str(v.source_id),
    name: str(v.name),
    assessment: assessment(v.assessment),
    objections: objections(v.objections)
  };
}
function empty(value: unknown): EmptyState {
  const v = obj(value);
  return {
    code: choice(v.code, ['observation_only']),
    reason: str(v.reason),
    follow_up: list(v.follow_up, str)
  };
}
function header(value: unknown): ChainHeader {
  const v = obj(value);
  return {
    ...origin(v),
    local_key: key(v.local_key),
    source_id: str(v.source_id),
    name: str(v.name),
    assessment: assessment(v.assessment),
    empty_state: nullable(v.empty_state, empty)
  };
}
function anchorRef(value: unknown): AnchorRef {
  const v = obj(value);
  if (v.reasoning_local_key !== undefined) {
    if (v.target_type || v.chain_local_key) fail();
    return {
      reasoning_local_key: key(v.reasoning_local_key),
      local_key: key(v.local_key),
      target_type: '',
      chain_local_key: null
    };
  }
  return {
    target_type: choice(v.target_type, [
      'macroeconomic_story',
      'industry_chain',
      'industry_chain_node'
    ]),
    local_key: key(v.local_key),
    chain_local_key: nullable(v.chain_local_key, key)
  };
}
export function parseAnalysisSummary(value: unknown): AnalysisSummary {
  const v = obj(value),
    s = obj(v.summary),
    impact = obj(s.impact_assessment);
  if (
    v.schema_version === 'report-publication/v5' &&
    (!origin(v).judgment_origin ||
      list(v.affected_anchors, (a) => origin(obj(a))).some((a) => !a.judgment_origin))
  )
    fail();
  return {
    schema_version: choice(v.schema_version, [
      'report-publication/v4',
      'report-publication/v5',
      'report-publication/v6'
    ]),
    ...origin(v),
    local_key: key(v.local_key),
    source_id: str(v.source_id),
    title: str(v.title),
    summary: {
      ...scope(s),
      ...(s.judgment !== undefined ? { judgment: str(s.judgment) } : {}),
      conclusion: str(s.conclusion),
      transmission_logic: str(s.transmission_logic),
      impact_assessment: {
        ...scope(impact),
        level: choice(impact.level, ['high', 'medium', 'low', 'pending']),
        rationale: str(impact.rationale)
      },
      affected_refs: list(s.affected_refs, anchorRef)
    },
    affected_anchors: list(v.affected_anchors, (x) => {
      const a = obj(x);
      return {
        reference: anchorRef(a.reference),
        source_id: str(a.source_id),
        name: str(a.name),
        ...origin(a),
        assessment: assessment(a.assessment)
      };
    }),
    chain_count: num(v.chain_count ?? 0),
    ...(v.reasoning_count !== undefined ? { reasoning_count: num(v.reasoning_count) } : {})
  };
}
export function parseAnalysisPage(value: unknown): AnalysisPage {
  const v = obj(value);
  const items = list(v.items, parseAnalysisSummary);
  unique(items.map((x) => x.local_key));
  const cursor = nullable(v.next_cursor, str);
  if (cursor && (!items.length || cursor.length > 2048)) fail();
  return { items, next_cursor: cursor };
}
export function parseAnalysisGroups(value: unknown): AnalysisGroup[] {
  const groups = list(value, (v) => ({
    ...parseAnalysisPage(v),
    kind: choice(obj(v).kind, analysisKinds)
  }));
  unique(groups.map((g) => g.kind));
  if (
    ['geopolitical_stories', 'macroeconomic_stories', 'concept_analyses'].some(
      (kind) => !groups.some((group) => group.kind === kind)
    )
  )
    fail();
  return groups;
}
export function parseAnalysisDetail(value: unknown, expectedKey: string): AnalysisDetail {
  const v = obj(value);
  const d: AnalysisDetail = {
    ...provenance(v),
    ...(typeof v.published_at === 'string' && Number.isFinite(Date.parse(v.published_at))
      ? { published_at: v.published_at }
      : {}),
    ...(v.companies !== undefined ? { companies: list(v.companies, macro) } : {}),
    summary: parseAnalysisSummary(v.summary),
    ...(v.reasonings !== undefined ? { reasonings: list(v.reasonings, unifiedReasoning) } : {}),
    macro_impacts: list(v.macro_impacts ?? [], macro),
    industry_chains: list(v.industry_chains ?? [], header)
  };
  if (d.summary.local_key !== expectedKey) fail();
  if (d.summary.schema_version === 'report-publication/v6') {
    if (!d.reasonings?.length || d.macro_impacts.length || d.industry_chains.length) fail();
    const reasonings = d.reasonings ?? [];
    unique(reasonings.map((r) => r.local_key));
    for (const a of d.summary.affected_anchors) {
      const r = reasonings.find((x) => x.local_key === a.reference.reasoning_local_key);
      const asset = r?.affected_assets.find((x) => x.local_key === a.reference.local_key);
      if (!asset || asset.name !== a.name) fail();
    }
  }
  if (
    d.summary.schema_version === 'report-publication/v5' &&
    (!d.judgment_origin ||
      !d.companies ||
      [...d.macro_impacts, ...d.industry_chains, ...d.companies].some((x) => !x.judgment_origin))
  )
    fail();
  unique([...d.macro_impacts, ...d.industry_chains].map((x) => x.local_key));
  return d;
}
export function parseAnalysisChain(value: unknown, expectedKey: string): AnalysisChain {
  const v = obj(value),
    r = obj(v.reasoning_summary),
    g = obj(v.graph);
  const c: AnalysisChain = {
    ...header(v),
    ...provenance(v),
    reasoning_summary: {
      logic: str(r.logic),
      support: claim(r.support),
      objections: objections(r.objections)
    },
    graph: {
      ...(g.scope !== undefined
        ? { scope: choice(g.scope, ['assessed_nodes_only'] as const) }
        : {}),
      nodes: list(g.nodes, (x) => {
        const n = obj(x);
        return { local_key: key(n.local_key), source_id: str(n.source_id), name: str(n.name) };
      }),
      edges: list(g.edges, (x) => {
        const e = obj(x);
        return {
          from_node_local_key: key(e.from_node_local_key),
          to_node_local_key: key(e.to_node_local_key),
          relation_label: str(e.relation_label)
        };
      })
    },
    affected_nodes: list(v.affected_nodes, (x) => ({
      ...macro(x),
      node_local_key: key(obj(x).node_local_key)
    }))
  };
  if (c.local_key !== expectedKey) fail();
  if (
    c.judgment_origin &&
    (c.graph.scope !== 'assessed_nodes_only' ||
      c.graph.nodes.length !== c.affected_nodes.length ||
      c.empty_state ||
      c.affected_nodes.some((x) => !x.judgment_origin))
  )
    fail();
  unique(c.graph.nodes.map((x) => x.local_key));
  unique(c.affected_nodes.map((x) => x.local_key));
  for (const n of c.affected_nodes) {
    const top = c.graph.nodes.find((x) => x.local_key === n.node_local_key);
    if (!top || top.source_id !== n.source_id || top.name !== n.name) fail();
  }
  for (const e of c.graph.edges)
    if (
      !c.graph.nodes.some((x) => x.local_key === e.from_node_local_key) ||
      !c.graph.nodes.some((x) => x.local_key === e.to_node_local_key)
    )
      fail();
  if (
    c.empty_state &&
    (c.affected_nodes.length ||
      c.assessment.direction !== 'pending' ||
      c.assessment.confidence !== null)
  )
    fail();
  return c;
}

function origin(v: Record<string, unknown>): Pick<Provenance, 'judgment_origin'> {
  return v.judgment_origin === undefined
    ? {}
    : { judgment_origin: choice(v.judgment_origin, ['direct', 'inferred'] as const) };
}
function provenance(v: Record<string, unknown>): Provenance {
  const o = origin(v);
  if (!o.judgment_origin) {
    if (v.reasoning_sources !== undefined || v.variable_signals !== undefined) fail();
    return {};
  }
  const r = obj(v.reasoning_sources);
  const sources: ReasoningSources = {
    signal_ids: list(r.signal_ids, key),
    event_ids: list(r.event_ids, key),
    upstream_refs: list(r.upstream_refs, (x) => {
      const ref = obj(x);
      return {
        entity_id: str(ref.entity_id),
        local_key: key(ref.local_key),
        ...(ref.mechanism !== undefined ? { mechanism: str(ref.mechanism) } : {}),
        ...(ref.condition !== undefined ? { condition: str(ref.condition) } : {})
      };
    })
  };
  const signals: VariableSignal[] = list(v.variable_signals, (x) => {
    const row = obj(x);
    return {
      ...scope(row),
      variable_id: key(row.variable_id),
      variable_name: str(row.variable_name),
      signal_id: key(row.signal_id),
      signal: str(row.signal),
      source_direction: choice(row.source_direction, ['UP', 'DOWN', 'STABLE', 'MIXED', 'UNKNOWN']),
      adoption: choice(row.adoption, ['adopted', 'qualified']),
      qualification: str(row.qualification),
      event_ids: list(row.event_ids, key)
    };
  });
  unique(sources.signal_ids);
  unique(sources.event_ids);
  unique(sources.upstream_refs.map((x) => x.local_key));
  if (
    (o.judgment_origin === 'direct') !== signals.length > 0 ||
    sources.signal_ids.length !== signals.length ||
    (!sources.event_ids.length && !sources.upstream_refs.length)
  )
    fail();
  signals.forEach((row, i) => {
    unique(row.event_ids);
    if (
      row.signal_id !== sources.signal_ids[i] ||
      !row.event_ids.length ||
      row.event_ids.some((id) => !sources.event_ids.includes(id)) ||
      row.evidence_count === 0
    )
      fail();
  });
  return { ...o, reasoning_sources: sources, variable_signals: signals };
}

export interface ReasoningMetric extends EvidenceScope {
  name: string;
  value?: number;
  display_value?: string;
  unit: string;
  measure_type: 'level' | 'change_rate' | 'count';
  period_label?: string;
  as_of?: string;
  value_nature: 'observed' | 'forecast' | 'assumption';
}
export interface ReasoningBlock {
  local_key: string;
  title: string;
  explanation: string;
  relation_type: 'causal' | 'comparison';
  nodes: { local_key: string; name: string; description?: string; metrics: ReasoningMetric[] }[];
  links?: { from_node_local_key: string; to_node_local_key: string; label?: string }[];
}
export interface UnifiedReasoning extends Provenance {
  local_key: string;
  source_id?: string;
  title: string;
  assessment: Assessment;
  reasoning_summary: { logic: string; support?: Claim; objections: Objections };
  reasoning_blocks: ReasoningBlock[];
  graph?: AnalysisChain['graph'];
  affected_assets: NodeImpact[];
  empty_state?: EmptyState | null;
}
function unifiedReasoning(value: unknown): UnifiedReasoning {
  const v = obj(value),
    r = obj(v.reasoning_summary);
  const assets = list(v.affected_assets, (x) => {
    const a = obj(x);
    return {
      ...macro({ ...a, source_id: a.source_id ?? '' }),
      node_local_key: a.node_local_key === undefined ? '' : str(a.node_local_key)
    };
  });
  unique(assets.map((a) => a.local_key));
  const blocks = list(v.reasoning_blocks, (blockValue) => {
    const b = obj(blockValue);
    const nodes = list(b.nodes, (nodeValue) => {
      const n = obj(nodeValue);
      return {
        local_key: key(n.local_key),
        name: str(n.name),
        ...(n.description !== undefined ? { description: str(n.description) } : {}),
        metrics: list(n.metrics, (x) => {
          const m = obj(x);
          if (m.value === undefined && !m.display_value) fail();
          return {
            ...scope(m),
            name: str(m.name),
            ...(m.value !== undefined ? { value: finite(m.value) } : {}),
            ...(m.display_value !== undefined ? { display_value: str(m.display_value) } : {}),
            unit: str(m.unit),
            measure_type: choice(m.measure_type, ['level', 'change_rate', 'count'] as const),
            value_nature: choice(m.value_nature, ['observed', 'forecast', 'assumption'] as const),
            ...(m.period_label !== undefined ? { period_label: str(m.period_label) } : {}),
            ...(m.as_of !== undefined ? { as_of: str(m.as_of) } : {})
          };
        })
      };
    });
    unique(nodes.map((n) => n.local_key));
    const links =
      b.links === undefined
        ? undefined
        : list(b.links, (x) => {
            const e = obj(x);
            return {
              from_node_local_key: key(e.from_node_local_key),
              to_node_local_key: key(e.to_node_local_key),
              ...(e.label !== undefined ? { label: str(e.label) } : {})
            };
          });
    const relation = choice(b.relation_type, ['causal', 'comparison'] as const);
    for (const e of links ?? [])
      if (
        relation !== 'causal' ||
        !nodes.some((n) => n.local_key === e.from_node_local_key) ||
        !nodes.some((n) => n.local_key === e.to_node_local_key)
      )
        fail();
    return {
      local_key: key(b.local_key),
      title: str(b.title),
      explanation: str(b.explanation),
      relation_type: relation,
      nodes,
      ...(links ? { links } : {})
    };
  });
  unique(blocks.map((b) => b.local_key));
  let graph: AnalysisChain['graph'] | undefined;
  if (v.graph) {
    const g = obj(v.graph);
    graph = {
      nodes: list(g.nodes, (x) => {
        const n = obj(x);
        return { local_key: key(n.local_key), source_id: str(n.source_id), name: str(n.name) };
      }),
      edges: list(g.edges, (x) => {
        const e = obj(x);
        return {
          from_node_local_key: key(e.from_node_local_key),
          to_node_local_key: key(e.to_node_local_key),
          relation_label: str(e.relation_label)
        };
      })
    };
    unique(graph.nodes.map((n) => n.local_key));
    for (const a of assets)
      if (
        a.node_local_key &&
        !graph.nodes.some(
          (n) =>
            n.local_key === a.node_local_key && n.name === a.name && n.source_id === a.source_id
        )
      )
        fail();
    for (const e of graph.edges)
      if (
        !graph.nodes.some((n) => n.local_key === e.from_node_local_key) ||
        !graph.nodes.some((n) => n.local_key === e.to_node_local_key)
      )
        fail();
  }
  return {
    ...provenance(v),
    local_key: key(v.local_key),
    ...(v.source_id !== undefined ? { source_id: str(v.source_id) } : {}),
    title: str(v.title),
    assessment: assessment(v.assessment),
    reasoning_summary: {
      logic: str(r.logic),
      ...(r.support ? { support: claim(r.support) } : {}),
      objections: objections(r.objections)
    },
    reasoning_blocks: blocks,
    affected_assets: assets,
    ...(graph ? { graph } : {}),
    ...(v.empty_state ? { empty_state: empty(v.empty_state) } : {})
  };
}
