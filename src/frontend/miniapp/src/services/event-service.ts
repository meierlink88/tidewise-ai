import { mockEvents } from '../mocks/mock-events';
import type { EventHighlight } from '../models/event';

export type EventHighlightsLoader = () => Promise<EventHighlight[]>;

export function createEventService(load: EventHighlightsLoader) {
	return {
		getEventHighlights: load
	};
}

const defaultService = createEventService(() => Promise.resolve(mockEvents));

export function getEventHighlights(): Promise<EventHighlight[]> {
	return defaultService.getEventHighlights();
}
