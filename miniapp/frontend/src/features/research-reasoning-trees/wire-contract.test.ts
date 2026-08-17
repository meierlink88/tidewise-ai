import { describe, expect, it } from 'vitest';
import { parseResearchReasoningTreeDetail } from './wire-contract';

const themeId = '70000000-0000-4000-8000-000000000001';
const treeId = '70000000-0000-4000-8000-000000000002';

describe('analyst snapshot reasoning tree wire contract', () => {
  it('accepts display metadata without formal ontology IDs', () => {
    const signal = {
      signal_key: 'signal:purchase-volume',
      variable_name: '采购数量',
      direction: null,
      variable_signal_key: '',
      signal_role: 'primary',
      signal_direction: '',
      display_summary: '采购数量待确认',
      display_order: 1
    };
    const readback = {
      theme_id: themeId,
      impact_node_ids: ['node:optical-module'],
      reasoning_tree: {
        tree_key: 'tree:optical-module',
        display_name: '高速光模块产业链',
        reasoning_tree_id: treeId,
        theme_id: themeId,
        industry_chain_id: '',
        industry_chain_name: '高速光模块产业链',
        title: '高速光模块',
        display_order: 1,
        one_line_conclusion: '采购数量仍需确认。',
        fact_summary: null,
        transmission_summary: null,
        impact_direction: 'uncertain',
        impact_strength: 'unknown',
        impact_summary: null,
        conclusion_boundary_summary: null,
        support_summary: null,
        counter_summary: null,
        invalidation_conditions: [],
        checkpoints: [],
        published_at: '2026-08-03T11:00:00Z',
        event_count: 1,
        events: [
          {
            event_id: '70000000-0000-4000-8000-000000000003',
            evidence_ids: [],
            title: 'Data 正式 Event',
            summary: 'Data 正式 Event 摘要',
            event_time: null,
            evidence_role: 'driver',
            supported_claim: null,
            display_order: 1
          }
        ],
        nodes: [
          {
            node_key: 'node:optical-module',
            display_name: '高速光模块',
            id: '70000000-0000-4000-8000-000000000004',
            position: 1,
            chain_node_id: '',
            name: '高速光模块',
            state_summary: null,
            impact_direction: 'uncertain',
            impact_strength: 'unknown',
            impact_summary: null,
            reasoning_basis_summary: null,
            evidence_gap_summary: null,
            incoming_industry_chain_graph_edge_id: null,
            incoming_transmission_title: null,
            incoming_transmission_mechanism: null,
            incoming_condition_summary: null,
            incoming_graph_edge: null,
            signals: [signal],
            primary_signal: signal,
            signal_display_summary: ''
          }
        ]
      }
    };

    const parsed = parseResearchReasoningTreeDetail(readback, themeId, treeId);

    expect(parsed.reasoningTree).toMatchObject({
      treeKey: 'tree:optical-module',
      displayName: '高速光模块产业链',
      industryChainEntityId: null
    });
    expect(parsed.reasoningTree.nodes[0]).toMatchObject({
      nodeKey: 'node:optical-module',
      chainNodeEntityId: null,
      displayName: '高速光模块',
      primarySignal: {
        signalKey: 'signal:purchase-volume',
        displaySummary: '采购数量待确认',
        direction: null
      }
    });
  });
});
