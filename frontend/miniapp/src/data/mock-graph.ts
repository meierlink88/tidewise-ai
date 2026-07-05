import type { GraphEdge, GraphNode } from '@/models/graph';

export const mockGraphNodes: GraphNode[] = [
  {
    id: 'node-event-policy',
    label: '产业政策',
    type: 'policy'
  },
  {
    id: 'node-sector-semiconductor',
    label: '半导体',
    type: 'sector'
  }
];

export const mockGraphEdges: GraphEdge[] = [
  {
    id: 'edge-policy-semiconductor',
    source: 'node-event-policy',
    target: 'node-sector-semiconductor',
    relation: '催化'
  }
];
