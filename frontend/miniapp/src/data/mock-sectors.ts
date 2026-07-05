import type { SectorSignal } from '@/models/sector';

export const mockSectors: SectorSignal[] = [
  {
    id: 'sector-001',
    name: '半导体',
    heat: 82,
    direction: 'up'
  },
  {
    id: 'sector-002',
    name: '新能源',
    heat: 76,
    direction: 'up'
  },
  {
    id: 'sector-003',
    name: '创新药',
    heat: 68,
    direction: 'flat'
  }
];
