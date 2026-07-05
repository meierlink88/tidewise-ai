import { mockEvents } from '@/data/mock-events';
import type { EventHighlight } from '@/models/event';
import { request } from './request';

export function getEventHighlights(): Promise<EventHighlight[]> {
  return request({
    mock: () => mockEvents
  });
}
