export interface GraphNode {
  id: string;
  label: string;
  type: 'event' | 'sector' | 'asset' | 'company' | 'policy';
}

export interface GraphEdge {
  id: string;
  source: string;
  target: string;
  relation: string;
}
