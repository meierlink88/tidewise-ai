import type {
  HomeResearchThemeItem,
  ResearchAttentionLevel,
  ResearchConclusionStatus,
  ResearchDirection,
  ResearchImpactStrength,
  ResearchInvestmentGuidanceAction,
  ResearchTransmissionStage
} from './contract';
import { formatResearchUpdateLabel } from './presentation';

const uuidPattern = /^[0-9a-f]{8}-[0-9a-f]{4}-[1-5][0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$/;
const directionValues = ['positive', 'negative', 'mixed', 'neutral', 'uncertain'] as const;
const strengthValues = ['strong', 'medium', 'weak', 'unknown'] as const;
const attentionValues = ['high', 'medium', 'low'] as const;
const conclusionStatusValues = ['supported', 'partial', 'conflicted'] as const;
const transmissionStageValues = ['identification', 'validation', 'diffusion', 'dampening'] as const;
const guidanceActionValues = ['focus', 'avoid', 'observe', 'differentiate'] as const;
const timeHorizonValues = ['short_term', 'medium_term', 'long_term', 'custom'] as const;
const relationRoleValues = ['driver', 'beneficiary', 'constraint', 'exposure'] as const;

type RecordValue = Record<string, unknown>;

export function parseResearchThemeWire(value: unknown, asOf?: string): HomeResearchThemeItem {
  const theme = record(value);
  onlyKeys(theme, [
    'id',
    'analysis_batch_id',
    'title',
    'one_line_conclusion',
    'conclusion_direction',
    'impact_strength',
    'attention_level',
    'conclusion_status',
    'transmission_stage',
    'investment_guidance_action',
    'investment_guidance_summary',
    'time_horizon_category',
    'time_horizon_summary',
    'transmission_summary',
    'checkpoint_summary',
    'risk_summary',
    'analysis_as_of',
    'window_start',
    'window_end',
    'published_at',
    'impacts',
    'evidence_event_count',
    'reasoning_tree_count'
  ]);
  const title = text(theme.title);
  const publishedAt = timestamp(theme.published_at);
  return {
    id: uuid(theme.id),
    analysisBatchId: text(theme.analysis_batch_id),
    title,
    oneLineConclusion: text(theme.one_line_conclusion),
    conclusionDirection: enumValue<ResearchDirection>(theme.conclusion_direction, directionValues),
    impactStrength: enumValue<ResearchImpactStrength>(theme.impact_strength, strengthValues),
    attentionLevel: nullableEnum<ResearchAttentionLevel>(theme.attention_level, attentionValues),
    conclusionStatus: nullableEnum<ResearchConclusionStatus>(
      theme.conclusion_status,
      conclusionStatusValues
    ),
    transmissionStage: enumValue<ResearchTransmissionStage>(
      theme.transmission_stage,
      transmissionStageValues
    ),
    investmentGuidanceAction: enumValue<ResearchInvestmentGuidanceAction>(
      theme.investment_guidance_action,
      guidanceActionValues
    ),
    investmentGuidanceSummary: text(theme.investment_guidance_summary),
    timeHorizonCategory: enumValue(theme.time_horizon_category, timeHorizonValues),
    timeHorizonSummary: nullableText(theme.time_horizon_summary),
    transmissionSummary: nullableText(theme.transmission_summary),
    checkpointSummary: nullableText(theme.checkpoint_summary),
    riskSummary: nullableText(theme.risk_summary),
    analysisAsOf: timestamp(theme.analysis_as_of),
    windowStart: timestamp(theme.window_start),
    windowEnd: timestamp(theme.window_end),
    publishedAt,
    updateLabel: asOf ? formatResearchUpdateLabel(publishedAt, asOf) : '',
    categories: categoriesForTheme(title),
    impacts: array(theme.impacts).map((item, index) => {
      const impact = record(item);
      onlyKeys(impact, [
        'chain_node_entity_id',
        'name',
        'relation_role',
        'impact_direction',
        'impact_summary',
        'primary_signal_display_summary',
        'display_order'
      ]);
      const displayOrder = positiveInteger(impact.display_order);
      if (displayOrder !== index + 1) invalid();
      return {
        chainNodeEntityId: uuid(impact.chain_node_entity_id),
        name: text(impact.name),
        relationRole: enumValue(impact.relation_role, relationRoleValues),
        impactDirection: enumValue<ResearchDirection>(impact.impact_direction, directionValues),
        impactSummary: nullableText(impact.impact_summary),
        primarySignalDisplaySummary: nullableText(impact.primary_signal_display_summary),
        displayOrder
      };
    }),
    evidenceEventCount: nonNegativeInteger(theme.evidence_event_count),
    reasoningTreeCount: nonNegativeInteger(theme.reasoning_tree_count)
  };
}

function record(value: unknown): RecordValue {
  if (typeof value !== 'object' || value === null || Array.isArray(value)) invalid();
  return value as RecordValue;
}
function array(value: unknown): unknown[] {
  if (!Array.isArray(value)) invalid();
  return value;
}
function onlyKeys(value: RecordValue, keys: string[]): void {
  const expected = new Set(keys);
  if (
    Object.keys(value).length !== keys.length ||
    Object.keys(value).some((key) => !expected.has(key))
  ) {
    invalid();
  }
}
function text(value: unknown): string {
  if (typeof value !== 'string' || value.trim() === '') invalid();
  return value;
}
function nullableText(value: unknown): string | null {
  if (value === null) return null;
  if (typeof value !== 'string') invalid();
  return value;
}
function uuid(value: unknown): string {
  const result = text(value);
  if (!uuidPattern.test(result)) invalid();
  return result;
}
function timestamp(value: unknown): string {
  const result = text(value);
  if (!result.endsWith('Z') || !Number.isFinite(Date.parse(result))) invalid();
  return result;
}
function nonNegativeInteger(value: unknown): number {
  if (!Number.isInteger(value) || (value as number) < 0) invalid();
  return value as number;
}
function positiveInteger(value: unknown): number {
  const result = nonNegativeInteger(value);
  if (result < 1) invalid();
  return result;
}
function enumValue<T extends string>(value: unknown, allowed: readonly T[]): T {
  if (typeof value !== 'string' || !allowed.includes(value as T)) invalid();
  return value as T;
}
function nullableEnum<T extends string>(value: unknown, allowed: readonly T[]): T | null {
  return value === null ? null : enumValue(value, allowed);
}
function categoriesForTheme(title: string): string[] {
  if (title.includes('中东') || title.includes('冲突')) return ['地缘政治'];
  if (title.includes('AI') || title.includes('光模块') || title.includes('半导体'))
    return ['算力基建'];
  return ['货币政策'];
}
function invalid(): never {
  throw new Error('invalid Research Theme wire contract');
}
