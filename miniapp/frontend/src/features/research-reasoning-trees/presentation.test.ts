import { describe, expect, it } from 'vitest';
import {
  formatReasoningTimestamp,
  researchDirectionLabel,
  researchEvidenceRoleLabel,
  researchSignalDirectionLabel,
  researchStrengthLabel,
  researchTreeConclusionMeta
} from './presentation';

describe('research reasoning tree presentation', () => {
  it('uses the V1 vocabulary without conflating direction and strength', () => {
    expect(researchEvidenceRoleLabel('contradicting')).toBe('反证');
    expect(researchSignalDirectionLabel('increase')).toBe('↑');
    expect(researchDirectionLabel('positive')).toBe('正向');
    expect(researchStrengthLabel('medium')).toBe('中等影响');
    expect(researchDirectionLabel('uncertain')).toBe('待验证');
    expect(researchStrengthLabel('unknown')).toBe('影响待判断');
    expect(researchTreeConclusionMeta('uncertain', 'unknown', '仍需核验')).toBe(
      '待验证 · 影响待判断 | 仍需核验'
    );
    expect(researchTreeConclusionMeta('positive', 'medium', null)).toBe('正向 · 中等影响');
  });

  it('formats API timestamps deterministically without device locale', () => {
    expect(formatReasoningTimestamp('2026-07-20T08:05:00Z')).toBe('07-20 08:05');
    expect(formatReasoningTimestamp('invalid')).toBe('时间未知');
  });
});
