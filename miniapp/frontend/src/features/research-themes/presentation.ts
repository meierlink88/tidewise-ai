import type {
  ResearchImpactStrength,
  ResearchInvestmentGuidanceAction,
  ResearchTransmissionStage
} from './contract';

const strengthLabels: Record<ResearchImpactStrength, string> = {
  strong: '强影响',
  medium: '中等影响',
  weak: '弱影响',
  unknown: '影响待判断'
};
const stageLabels: Record<ResearchTransmissionStage, string> = {
  identification: '识别',
  validation: '验证',
  diffusion: '扩散',
  dampening: '钝化'
};
const guidanceActionLabels: Record<ResearchInvestmentGuidanceAction, string> = {
  focus: '重点关注',
  avoid: '回避',
  observe: '继续观察',
  differentiate: '区别对待'
};
export const researchImpactStrengthLabel = (value: ResearchImpactStrength) => strengthLabels[value];
export const researchTransmissionStageLabel = (value: ResearchTransmissionStage) =>
  stageLabels[value];
export const researchInvestmentGuidanceActionLabel = (value: ResearchInvestmentGuidanceAction) =>
  guidanceActionLabels[value];
export function formatResearchUpdateLabel(publishedAt: string, asOf: string): string {
  const published = Date.parse(publishedAt),
    reference = Date.parse(asOf);
  if (!Number.isFinite(published) || !Number.isFinite(reference)) return '更新时间未知';
  const minutes = Math.max(0, Math.floor((reference - published) / 60_000));
  if (minutes < 1) return '刚刚更新';
  if (minutes < 60) return `${minutes} 分钟前`;
  const hours = Math.floor(minutes / 60);
  if (hours < 24) return `${hours} 小时前`;
  const date = new Date(published);
  return `${String(date.getUTCMonth() + 1).padStart(2, '0')}-${String(date.getUTCDate()).padStart(2, '0')} ${String(date.getUTCHours()).padStart(2, '0')}:${String(date.getUTCMinutes()).padStart(2, '0')}`;
}
