import { mockSectors } from '../mocks/mock-sectors';
import type { SectorSignal } from '../models/sector';

export type SectorSignalsLoader = () => Promise<SectorSignal[]>;

export function createSectorService(load: SectorSignalsLoader) {
	return {
		getSectorSignals: load
	};
}

const defaultService = createSectorService(() => Promise.resolve(mockSectors));

export function getSectorSignals(): Promise<SectorSignal[]> {
	return defaultService.getSectorSignals();
}
