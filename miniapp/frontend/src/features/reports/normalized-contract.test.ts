import { describe, expect, it } from 'vitest';
import fixture from '../../mocks/reports/normalized.json';
import {
  parseAnalysisChain,
  parseAnalysisDetail,
  parseAnalysisGroups
} from './normalized-contract';

describe('normalized report contract', () => {
  it('accepts every group, macro impact, chain and observation-only state in the provider fixture', () => {
    expect(parseAnalysisGroups(fixture.groups)).toHaveLength(3);
    Object.values(fixture.details).forEach((d) =>
      expect(parseAnalysisDetail(d, d.summary.local_key)).toEqual(d)
    );
    Object.values(fixture.chains).forEach((c) =>
      expect(parseAnalysisChain(c, c.local_key)).toEqual(c)
    );
  });
  it('rejects inconsistent evidence counters and cross-graph node references', () => {
    const original = Object.values(fixture.chains)[0];
    for (const count of [-1, 0, 1.5]) {
      const chain = structuredClone(original);
      chain.assessment.evidence_count = count;
      expect(() => parseAnalysisChain(chain, chain.local_key)).toThrow();
    }
    const chain = structuredClone(original);
    chain.affected_nodes[0].node_local_key = 'outside-chain';
    expect(() => parseAnalysisChain(chain, chain.local_key)).toThrow();
  });
  it('does not turn observation-only content into a scored node', () => {
    const chain = parseAnalysisChain(fixture.chains['concept_analyses/c1/c5-ai1'], 'c5-ai1');
    expect(chain.assessment.confidence).toBeNull();
    expect(chain.affected_nodes).toEqual([]);
    expect(chain.empty_state?.follow_up).toHaveLength(1);
  });
});
