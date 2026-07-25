import { describe, expect, it } from 'vitest';
import {
  formatResearchUpdateLabel,
  researchImpactLabel,
  researchTransmissionStageLabel
} from './presentation';

describe('research theme presentation', () => {
  it('translates conclusion progress without using industry-chain positions', () => {
    expect(researchTransmissionStageLabel('identification')).toBe('识别');
    expect(researchTransmissionStageLabel('validation')).toBe('验证');
    expect(researchTransmissionStageLabel('diffusion')).toBe('扩散');
    expect(researchTransmissionStageLabel('dampening')).toBe('钝化');
  });

  it('uses one shared impact-level vocabulary across Theme surfaces', () => {
    expect(researchImpactLabel('high')).toBe('高影响');
    expect(researchImpactLabel('focus')).toBe('重点关注');
    expect(researchImpactLabel('watch')).toBe('持续观察');
  });

  it('derives a stable update label from API timestamps', () => {
    expect(formatResearchUpdateLabel('2026-07-18T09:00:00Z', '2026-07-18T10:00:00Z')).toBe('1小时前更新');
    expect(formatResearchUpdateLabel('2026-07-17T22:00:00Z', '2026-07-18T10:00:00Z')).toBe('12小时前更新');
  });
});
