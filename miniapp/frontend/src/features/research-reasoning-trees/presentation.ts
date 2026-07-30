import type { ResearchDirection } from '../research-themes/contract';

export type ResearchNodeJudgmentKind = 'opportunity' | 'risk' | 'uncertain';
export interface ResearchNodeJudgment {
  kind: ResearchNodeJudgmentKind;
  label: '机会' | '风险' | '不确定';
}
const nodeJudgments: Record<ResearchDirection, ResearchNodeJudgment> = {
  positive: { kind: 'opportunity', label: '机会' },
  negative: { kind: 'risk', label: '风险' },
  mixed: { kind: 'uncertain', label: '不确定' },
  neutral: { kind: 'uncertain', label: '不确定' },
  uncertain: { kind: 'uncertain', label: '不确定' }
};
export const researchNodeJudgment = (direction: ResearchDirection) => nodeJudgments[direction];
export function formatReasoningTimestamp(value: string): string {
  const timestamp = Date.parse(value);
  if (!Number.isFinite(timestamp)) return '时间未知';
  const date = new Date(timestamp);
  return `${String(date.getUTCMonth() + 1).padStart(2, '0')}-${String(date.getUTCDate()).padStart(2, '0')} ${String(date.getUTCHours()).padStart(2, '0')}:${String(date.getUTCMinutes()).padStart(2, '0')}`;
}
