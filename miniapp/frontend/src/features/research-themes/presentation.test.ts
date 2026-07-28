import { describe, expect, it } from 'vitest';
import {
  formatResearchUpdateLabel,
  researchImpactStrengthLabel,
  researchInvestmentGuidanceActionLabel,
  researchTransmissionStageLabel
} from './presentation';

describe('research theme presentation', () => {
  it('keeps transmission stage separate from conclusion status', () => {
    expect(researchTransmissionStageLabel('identification')).toBe('识别');
    expect(researchTransmissionStageLabel('validation')).toBe('验证');
    expect(researchTransmissionStageLabel('diffusion')).toBe('扩散');
    expect(researchTransmissionStageLabel('dampening')).toBe('钝化');
  });

  it('uses the Theme impact-strength vocabulary', () => {
    expect(researchImpactStrengthLabel('strong')).toBe('强影响');
    expect(researchImpactStrengthLabel('medium')).toBe('中等影响');
    expect(researchImpactStrengthLabel('weak')).toBe('弱影响');
    expect(researchImpactStrengthLabel('unknown')).toBe('影响待判断');
  });

  it('uses the controlled investment-guidance vocabulary', () => {
    expect(researchInvestmentGuidanceActionLabel('focus')).toBe('重点关注');
    expect(researchInvestmentGuidanceActionLabel('avoid')).toBe('回避');
    expect(researchInvestmentGuidanceActionLabel('observe')).toBe('继续观察');
    expect(researchInvestmentGuidanceActionLabel('differentiate')).toBe('区别对待');
  });

  it('derives a stable update label from API timestamps', () => {
    expect(formatResearchUpdateLabel('2026-07-18T09:59:00Z', '2026-07-18T10:00:00Z')).toBe(
      '1 分钟前'
    );
    expect(formatResearchUpdateLabel('2026-07-18T09:00:00Z', '2026-07-18T10:00:00Z')).toBe(
      '1 小时前'
    );
    expect(formatResearchUpdateLabel('2026-07-18T10:00:00Z', '2026-07-18T10:00:00Z')).toBe(
      '刚刚更新'
    );
    expect(formatResearchUpdateLabel('2026-07-17T09:00:00Z', '2026-07-18T10:00:00Z')).toBe(
      '07-17 09:00'
    );
  });
});
