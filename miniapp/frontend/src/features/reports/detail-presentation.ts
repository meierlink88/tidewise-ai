import type { AnalysisChain, AnalysisDetail, MacroImpact } from './normalized-contract';

type DetailTab =
  | (MacroImpact & { type: 'macro' })
  | { local_key: string; name: string; type: 'inline-chain'; chain: AnalysisChain };

// v6 changes the transport container, not the established macro/graph presentations.
// A graph must come from the report; asset lists alone do not imply a supply-chain graph.
export function reasoningTabs(detail: AnalysisDetail): DetailTab[] {
  return (detail.reasonings ?? []).map((reasoning) => {
    const base = {
      ...reasoning,
      source_id: reasoning.source_id ?? '',
      name: reasoning.title,
      empty_state: reasoning.empty_state ?? null
    };
    if (reasoning.graph) {
      return {
        local_key: reasoning.local_key,
        name: reasoning.title,
        type: 'inline-chain',
        chain: {
          ...base,
          graph: reasoning.graph,
          affected_nodes: reasoning.affected_assets.filter((asset) => asset.node_local_key),
          reasoning_summary: {
            ...reasoning.reasoning_summary,
            support: reasoning.reasoning_summary.support ?? {
              text: '',
              basis: 'inference',
              evidence_count: 0,
              evidence_scope_token: null
            }
          }
        }
      };
    }
    return {
      ...base,
      type: 'macro',
      assessment: {
        ...reasoning.assessment,
        transmission_logic: reasoning.reasoning_summary.logic,
        conditions: reasoning.reasoning_summary.support
          ? [reasoning.reasoning_summary.support.text]
          : reasoning.assessment.conditions
      },
      objections: reasoning.reasoning_summary.objections
    };
  });
}
