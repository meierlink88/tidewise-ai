import { describe, expect, it } from 'vitest';
import {
  formatReasoningTimestamp,
  researchChangeDirectionPresentation,
  researchEvidenceRoleLabel
} from './presentation';

describe('research reasoning tree presentation', () => {
  it('uses the confirmed product labels for evidence roles', () => {
    expect(researchEvidenceRoleLabel('driver')).toBe('驱动');
    expect(researchEvidenceRoleLabel('supporting')).toBe('支持');
    expect(researchEvidenceRoleLabel('contradicting')).toBe('反证');
    expect(researchEvidenceRoleLabel('context')).toBe('背景');
  });

  it.each([
    ['increase', '↑ 增强', 'positive'],
    ['decrease', '↓ 减弱', 'negative'],
    ['mixed', '↕ 分化', 'mixed'],
    ['unchanged', '→ 持平', 'neutral'],
    ['uncertain', '待验证', 'uncertain']
  ] as const)('maps %s without inventing another direction', (direction, label, tone) => {
    expect(researchChangeDirectionPresentation(direction)).toEqual({ label, tone });
  });

  it('formats API timestamps deterministically without relying on device locale', () => {
    expect(formatReasoningTimestamp('2026-07-20T08:05:00Z')).toBe('07-20 08:05');
    expect(formatReasoningTimestamp('invalid')).toBe('时间未知');
  });
});
