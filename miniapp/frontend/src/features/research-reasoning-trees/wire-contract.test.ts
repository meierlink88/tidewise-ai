import { describe, expect, it } from 'vitest';
import preparedRequest from '../../../../../testdata/research-theme-analyst-snapshot-v3/01-uat-at01-prepared-request.json';
import { parseResearchReasoningTreeDetail } from './wire-contract';

const themeId = '70000000-0000-4000-8000-000000000001';
const treeId = '70000000-0000-4000-8000-000000000002';

describe('analyst snapshot reasoning tree wire contract', () => {
  it('renders the prepared real-UAT display contract without formal ontology IDs', () => {
    const preparedTree = preparedRequest.reasoning_trees[0];
    const readback = {
      theme_id: themeId,
      impact_node_ids: preparedRequest.theme.impacts.map((impact) => impact.node_key),
      reasoning_tree: {
        tree_key: preparedTree.tree_key,
        display_name: preparedTree.display_name,
        reasoning_tree_id: treeId,
        theme_id: themeId,
        industry_chain_entity_id: '',
        industry_chain_name: preparedTree.display_name,
        title: preparedTree.title,
        display_order: preparedTree.display_order,
        one_line_conclusion: preparedTree.one_line_conclusion,
        fact_summary: preparedTree.fact_summary,
        transmission_summary: preparedTree.transmission_summary,
        impact_direction: preparedTree.impact_direction,
        impact_strength: preparedTree.impact_strength,
        impact_summary: preparedTree.impact_summary,
        conclusion_boundary_summary: preparedTree.conclusion_boundary_summary,
        support_summary: preparedTree.support_summary,
        counter_summary: preparedTree.counter_summary,
        invalidation_conditions: preparedTree.invalidation_conditions,
        checkpoints: preparedTree.checkpoints,
        published_at: preparedRequest.analysis_as_of,
        event_count: preparedTree.events.length,
        events: preparedTree.events.map((event) => ({
          event_id: event.event_id,
          evidence_ids: event.evidence_ids,
          title: 'Data 正式 Event',
          summary: 'Data 正式 Event 摘要',
          event_time: null,
          evidence_role: event.evidence_role,
          supported_claim: null,
          display_order: event.display_order
        })),
        nodes: preparedTree.nodes.map((node, nodeIndex) => {
          const signals = node.signals.map((signal) => ({
            signal_key: signal.signal_key,
            variable_name: signal.variable_name,
            direction: signal.direction,
            variable_signal_key: '',
            signal_role: signal.role,
            signal_direction: signal.direction ?? '',
            display_summary: signal.display_summary,
            display_order: signal.display_order
          }));
          return {
            node_key: node.node_key,
            display_name: node.display_name,
            id: `70000000-0000-4000-8000-${String(nodeIndex + 3).padStart(12, '0')}`,
            position: node.position,
            chain_node_entity_id: '',
            name: node.display_name,
            state_summary: node.state_summary,
            impact_direction: node.impact_direction,
            impact_strength: node.impact_strength,
            impact_summary: node.impact_summary,
            reasoning_basis_summary: node.reasoning_basis_summary,
            evidence_gap_summary: node.evidence_gap_summary,
            incoming_industry_chain_graph_edge_id: null,
            incoming_transmission_title: node.incoming_transmission?.title ?? null,
            incoming_transmission_mechanism: node.incoming_transmission?.mechanism ?? null,
            incoming_condition_summary: node.incoming_transmission?.condition_summary ?? null,
            incoming_graph_edge: null,
            signals,
            primary_signal: signals[0],
            signal_display_summary: signals
              .filter((signal) => signal.signal_role !== 'primary')
              .map((signal) => signal.display_summary)
              .join(' · ')
          };
        })
      }
    };

    const parsed = parseResearchReasoningTreeDetail(readback, themeId, treeId);

    expect(parsed.reasoningTree.treeKey).toBe(preparedTree.tree_key);
    expect(parsed.reasoningTree.displayName).toBe(preparedTree.display_name);
    expect(parsed.reasoningTree.industryChainEntityId).toBeNull();
    expect(parsed.reasoningTree.nodes[0]).toMatchObject({
      nodeKey: preparedTree.nodes[0].node_key,
      chainNodeEntityId: null,
      displayName: preparedTree.nodes[0].display_name,
      primarySignal: {
        signalKey: preparedTree.nodes[0].signals[0].signal_key,
        displaySummary: preparedTree.nodes[0].signals[0].display_summary,
        direction: null
      }
    });
    expect(parsed.reasoningTree.nodes[1].incomingTransmissionTitle).toBeNull();
    expect(parsed.reasoningTree.nodes[1].incomingTransmissionMechanism).toBe(
      preparedTree.nodes[1].incoming_transmission?.mechanism
    );
  });
});
