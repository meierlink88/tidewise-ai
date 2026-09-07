import { ReportError } from './contract';

export const analysisKinds = [
  'geopolitical_stories',
  'macroeconomic_stories',
  'concept_analyses'
] as const;
export type AnalysisKind = (typeof analysisKinds)[number];
export const analysisLabels: Record<AnalysisKind, string> = {
  geopolitical_stories: '地缘政治',
  macroeconomic_stories: '宏观经济',
  concept_analyses: '产业链'
};
export interface EvidenceScope {
  evidence_scope_token: string | null;
  evidence_count: number;
}
export interface Assessment extends EvidenceScope {
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
export interface MacroImpact {
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
  local_key: string;
  source_id: string;
  name: string;
  assessment: Assessment;
  empty_state: EmptyState | null;
}
export interface AnalysisChain extends ChainHeader {
  reasoning_summary: { logic: string; support: Claim; objections: Objections };
  graph: {
    nodes: { local_key: string; source_id: string; name: string }[];
    edges: { from_node_local_key: string; to_node_local_key: string; relation_label: string }[];
  };
  affected_nodes: NodeImpact[];
}
export interface AnchorRef {
  target_type: string;
  local_key: string;
  chain_local_key: string | null;
}
export interface AnalysisSummary {
  schema_version: 'report-publication/v4';
  local_key: string;
  source_id: string;
  title: string;
  summary: EvidenceScope & {
    conclusion: string;
    transmission_logic: string;
    impact_assessment: EvidenceScope & { level: string; rationale: string };
    affected_refs: AnchorRef[];
  };
  affected_anchors: {
    reference: AnchorRef;
    source_id: string;
    name: string;
    assessment: Assessment;
  }[];
  chain_count: number;
}
export interface AnalysisPage {
  items: AnalysisSummary[];
  next_cursor: string | null;
}
export interface AnalysisGroup extends AnalysisPage {
  kind: AnalysisKind;
}
export interface AnalysisDetail {
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
function assessment(value: unknown): Assessment {
  const v = obj(value),
    w = obj(v.forecast_window);
  return {
    ...scope(v),
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
    local_key: key(v.local_key),
    source_id: str(v.source_id),
    name: str(v.name),
    assessment: assessment(v.assessment),
    empty_state: nullable(v.empty_state, empty)
  };
}
function anchorRef(value: unknown): AnchorRef {
  const v = obj(value);
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
  return {
    schema_version: choice(v.schema_version, ['report-publication/v4']),
    local_key: key(v.local_key),
    source_id: str(v.source_id),
    title: str(v.title),
    summary: {
      ...scope(s),
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
        assessment: assessment(a.assessment)
      };
    }),
    chain_count: num(v.chain_count)
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
  if (groups.length !== 3) fail();
  return groups;
}
export function parseAnalysisDetail(value: unknown, expectedKey: string): AnalysisDetail {
  const v = obj(value);
  const d = {
    summary: parseAnalysisSummary(v.summary),
    macro_impacts: list(v.macro_impacts, macro),
    industry_chains: list(v.industry_chains, header)
  };
  if (d.summary.local_key !== expectedKey) fail();
  unique([...d.macro_impacts, ...d.industry_chains].map((x) => x.local_key));
  return d;
}
export function parseAnalysisChain(value: unknown, expectedKey: string): AnalysisChain {
  const v = obj(value),
    r = obj(v.reasoning_summary),
    g = obj(v.graph);
  const c: AnalysisChain = {
    ...header(v),
    reasoning_summary: {
      logic: str(r.logic),
      support: claim(r.support),
      objections: objections(r.objections)
    },
    graph: {
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
