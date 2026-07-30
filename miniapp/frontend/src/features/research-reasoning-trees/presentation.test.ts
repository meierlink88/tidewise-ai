import { describe, expect, it } from 'vitest';
import { formatReasoningTimestamp, researchNodeJudgment } from './presentation';

describe('research reasoning tree presentation', () => {
  it('formats API timestamps deterministically without device locale', () => {
    expect(formatReasoningTimestamp('2026-07-20T08:05:00Z')).toBe('07-20 08:05');
    expect(formatReasoningTimestamp('invalid')).toBe('时间未知');
  });

  it('maps node impact direction into the three investment judgments', () => {
    expect(researchNodeJudgment('positive')).toEqual({
      kind: 'opportunity',
      label: '机会'
    });
    expect(researchNodeJudgment('negative')).toEqual({
      kind: 'risk',
      label: '风险'
    });
    expect(researchNodeJudgment('uncertain')).toEqual({
      kind: 'uncertain',
      label: '不确定'
    });
    expect(researchNodeJudgment('mixed')).toEqual({
      kind: 'uncertain',
      label: '不确定'
    });
    expect(researchNodeJudgment('neutral')).toEqual({
      kind: 'uncertain',
      label: '不确定'
    });
  });
});
