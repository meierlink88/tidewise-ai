import { describe, expect, it } from 'vitest';
import fixture from '../../mocks/reports/normalized.json';
import v5 from '../../mocks/reports/normalized-v5.json';
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

it('preserves all v5 read fields without changing the published judgment', () => {
  expect(parseAnalysisGroups(v5.groups)).toEqual(v5.groups);
  Object.values(v5.details).forEach((d) =>
    expect(parseAnalysisDetail(d, d.summary.local_key)).toEqual(d)
  );
  Object.values(v5.chains).forEach((c) => expect(parseAnalysisChain(c, c.local_key)).toEqual(c));
});

it('rejects malformed v5 provenance and unknown report versions', () => {
  const chain = structuredClone(v5.chains['geopolitical_stories/g1/g1-2-chain']);
  chain.variable_signals[0].signal_id = 'mismatch';
  expect(() => parseAnalysisChain(chain, chain.local_key)).toThrow();
  const detail = { ...v5.details['geopolitical_stories/g1'], variable_signals: null };
  expect(() => parseAnalysisDetail(detail, 'g1')).toThrow();
  expect(() =>
    parseAnalysisGroups(
      v5.groups.map((g) => ({
        ...g,
        items: g.items.map((i) => ({ ...i, schema_version: 'report-publication/v99' }))
      }))
    )
  ).toThrow();
});
