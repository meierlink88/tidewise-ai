export interface SectorSignal {
  id: string;
  name: string;
  heat: number;
  direction: 'up' | 'down' | 'flat';
}
