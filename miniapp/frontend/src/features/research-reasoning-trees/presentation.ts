import type { ResearchEvidenceRole, ResearchSignalDirection } from './contract';
import type { ResearchDirection, ResearchImpactStrength } from '../research-themes/contract';

const evidenceLabels: Record<ResearchEvidenceRole, string> = {
  driver: '驱动',
  supporting: '支持',
  contradicting: '反证',
  context: '背景'
};
const signalLabels: Record<ResearchSignalDirection, string> = {
  increase: '↑',
  decrease: '↓',
  mixed: '↕',
  unchanged: '→',
  uncertain: '待确认'
};
const directionLabels: Record<ResearchDirection, string> = {
  positive: '正向',
  negative: '负向',
  mixed: '分化',
  neutral: '中性',
  uncertain: '待验证'
};
const strengthLabels: Record<ResearchImpactStrength, string> = {
  strong: '强影响',
  medium: '中等影响',
  weak: '弱影响',
  unknown: '影响待判断'
};
export const researchEvidenceRoleLabel = (role: ResearchEvidenceRole) => evidenceLabels[role];
export const researchSignalDirectionLabel = (direction: ResearchSignalDirection) =>
  signalLabels[direction];
export const researchDirectionLabel = (direction: ResearchDirection) => directionLabels[direction];
export const researchStrengthLabel = (strength: ResearchImpactStrength) => strengthLabels[strength];
export function researchTreeConclusionMeta(
  direction: ResearchDirection,
  strength: ResearchImpactStrength,
  impactSummary: string | null
): string {
  const base = `${researchDirectionLabel(direction)} · ${researchStrengthLabel(strength)}`;
  return impactSummary ? `${base} | ${impactSummary}` : base;
}
export function formatReasoningTimestamp(value: string): string {
  const timestamp = Date.parse(value);
  if (!Number.isFinite(timestamp)) return '时间未知';
  const date = new Date(timestamp);
  return `${String(date.getUTCMonth() + 1).padStart(2, '0')}-${String(date.getUTCDate()).padStart(2, '0')} ${String(date.getUTCHours()).padStart(2, '0')}:${String(date.getUTCMinutes()).padStart(2, '0')}`;
}
