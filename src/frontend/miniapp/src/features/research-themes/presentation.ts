import type { ResearchImpactLevel, ResearchTransmissionStage } from './contract';

const impactLabels: Record<ResearchImpactLevel, string> = {
  high: '高影响',
  focus: '重点关注',
  watch: '持续观察'
};

const transmissionStageLabels: Record<ResearchTransmissionStage, string> = {
  identification: '识别',
  validation: '验证',
  diffusion: '扩散',
  dampening: '钝化'
};

export function researchTransmissionStageLabel(stage: ResearchTransmissionStage): string {
  return transmissionStageLabels[stage];
}

export function researchImpactLabel(level: ResearchImpactLevel): string {
  return impactLabels[level];
}

export function formatResearchUpdateLabel(publishedAt: string, asOf: string): string {
  const published = Date.parse(publishedAt);
  const reference = Date.parse(asOf);
  if (!Number.isFinite(published) || !Number.isFinite(reference)) {
    return '更新时间未知';
  }

  const elapsedHours = Math.max(0, Math.floor((reference - published) / 3_600_000));
  if (elapsedHours < 1) {
    return '刚刚更新';
  }
  if (elapsedHours < 24) {
    return `${elapsedHours}小时前更新`;
  }

  const date = new Date(published);
  const month = String(date.getUTCMonth() + 1).padStart(2, '0');
  const day = String(date.getUTCDate()).padStart(2, '0');
  const hour = String(date.getUTCHours()).padStart(2, '0');
  const minute = String(date.getUTCMinutes()).padStart(2, '0');
  return `${month}-${day} ${hour}:${minute}更新`;
}
