import { describe, expect, it } from 'vitest';
import unified from '../../mocks/reports/unified-v6.json';
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

it('accepts optional industry groups but requires all historical groups', () => {
  const industry = { ...v5.groups[2], kind: 'industry_chain_analyses' };
  expect(parseAnalysisGroups([...v5.groups, industry])).toHaveLength(4);
  expect(() => parseAnalysisGroups([v5.groups[0], v5.groups[1], industry])).toThrow();
});

it('accepts additive publication metadata and preserves older detail responses', () => {
  const detail = fixture.details['geopolitical_stories/g1'];
  expect(parseAnalysisDetail(detail, 'g1').published_at).toBeUndefined();
  expect(
    parseAnalysisDetail({ ...detail, published_at: '2026-09-09T04:00:00Z', future: true }, 'g1')
      .published_at
  ).toBe('2026-09-09T04:00:00Z');
  expect(
    parseAnalysisDetail({ ...detail, published_at: 'invalid' }, 'g1').published_at
  ).toBeUndefined();
});

function detailWithGraphKey(newKey: string) {
  const value = structuredClone(unified.details['concept_analyses/c1']);
  const reasoning = value.reasonings[0];
  const oldKey = reasoning.graph.nodes[0].local_key;
  reasoning.graph.nodes[0].local_key = newKey;
  reasoning.graph.edges.forEach((edge) => {
    if (edge.from_node_local_key === oldKey) edge.from_node_local_key = newKey;
    if (edge.to_node_local_key === oldKey) edge.to_node_local_key = newKey;
  });
  reasoning.affected_assets.forEach((asset) => {
    if (asset.node_local_key === oldKey) asset.node_local_key = newKey;
  });
  return value;
}

it.each([128, 134, 16000])(
  'preserves v6 graph references with %i-character node keys',
  (length) => {
    const value = detailWithGraphKey('n'.repeat(length));
    const reasoning = value.reasonings[0];
    const detail = parseAnalysisDetail(value, value.summary.local_key);
    expect(detail.reasonings?.[0].graph?.nodes).toEqual(reasoning.graph.nodes);
    expect(detail.reasonings?.[0].graph?.edges).toEqual(reasoning.graph.edges);
    expect(detail.reasonings?.[0].affected_assets.map((a) => a.node_local_key)).toEqual(
      reasoning.affected_assets.map((a) => a.node_local_key)
    );
    reasoning.graph.edges[0].from_node_local_key = 'missing';
    expect(() => parseAnalysisDetail(value, value.summary.local_key)).toThrow();
  }
);

it.each(['', ' node ', 'n'.repeat(16001), '\ud800'])(
  'rejects invalid v6 graph node text',
  (key) => {
    const value = detailWithGraphKey(key);
    expect(() => parseAnalysisDetail(value, value.summary.local_key)).toThrow();
  }
);

it('counts graph key characters consistently with the provider', () => {
  const value = detailWithGraphKey('𠮷'.repeat(16000));
  expect(
    parseAnalysisDetail(value, value.summary.local_key).reasonings?.[0].graph?.nodes[0].local_key
  ).toBe(value.reasonings[0].graph.nodes[0].local_key);
});

it('reads the v6 provider fixture for all three reasoning domains', () => {
  expect(parseAnalysisGroups(unified.groups)).toHaveLength(4);
  Object.values(unified.details).forEach((value) => {
    const detail = parseAnalysisDetail(value, value.summary.local_key);
    expect(detail.reasonings?.length).toBe(value.reasonings.length);
    value.reasonings.forEach((reasoning, i) => {
      expect(detail.reasonings?.[i].assessment).toEqual(reasoning.assessment);
      expect(detail.reasonings?.[i].reasoning_summary).toEqual(reasoning.reasoning_summary);
    });
  });
});

it('keeps missing allocation distinct from unchanged and checks reasoning-local references', () => {
  const value = structuredClone(unified.details['geopolitical_stories/g1']);
  const parsed = parseAnalysisDetail(value, 'g1');
  expect(parsed.reasonings?.[0].affected_assets[0].assessment.weight_delta_pp).toBe(4);
  expect(parsed.reasonings?.[1].affected_assets[0].assessment.weight_delta_pp).toBeUndefined();
  value.summary.affected_anchors[0].reference.reasoning_local_key = 'missing';
  expect(() => parseAnalysisDetail(value, 'g1')).toThrow();
});
