import { mockSectors } from '@/data/mock-sectors';
import type { SectorSignal } from '@/models/sector';
import { request } from './request';

export function getSectorSignals(): Promise<SectorSignal[]> {
  return request({
    mock: () => mockSectors
  });
}
