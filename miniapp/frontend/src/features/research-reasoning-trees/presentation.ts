import type { ResearchChangeDirection, ResearchEvidenceRole } from './contract';

const evidenceRoleLabels: Record<ResearchEvidenceRole, string> = {
  driver: '驱动',
  supporting: '支持',
  contradicting: '反证',
  context: '背景'
};

const changeDirectionPresentations: Record<
  ResearchChangeDirection,
  { label: string; tone: 'positive' | 'negative' | 'mixed' | 'neutral' | 'uncertain' }
> = {
  increase: { label: '↑ 增强', tone: 'positive' },
  decrease: { label: '↓ 减弱', tone: 'negative' },
  mixed: { label: '↕ 分化', tone: 'mixed' },
  unchanged: { label: '→ 持平', tone: 'neutral' },
  uncertain: { label: '待验证', tone: 'uncertain' }
};

export function researchEvidenceRoleLabel(role: ResearchEvidenceRole): string {
  return evidenceRoleLabels[role];
}

export function researchChangeDirectionPresentation(direction: ResearchChangeDirection) {
  return changeDirectionPresentations[direction];
}

export function formatReasoningTimestamp(value: string): string {
  const timestamp = Date.parse(value);
  if (!Number.isFinite(timestamp)) return '时间未知';

  const date = new Date(timestamp);
  const month = String(date.getUTCMonth() + 1).padStart(2, '0');
  const day = String(date.getUTCDate()).padStart(2, '0');
  const hour = String(date.getUTCHours()).padStart(2, '0');
  const minute = String(date.getUTCMinutes()).padStart(2, '0');
  return `${month}-${day} ${hour}:${minute}`;
}
