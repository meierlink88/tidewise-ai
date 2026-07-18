import type { DailyBriefHomeView } from '../models/daily-brief-view';

export type HomeSectionKey = 'brief-summary' | 'themes' | 'conclusions' | 'impacts' | 'evidence' | 'safety-note';

export interface HomeSectionDefinition {
  key: HomeSectionKey;
  visible: (view: DailyBriefHomeView) => boolean;
}

export const homeSectionRegistry: HomeSectionDefinition[] = [
  { key: 'brief-summary', visible: () => true },
  { key: 'themes', visible: (view) => view.themes.length > 0 },
  { key: 'conclusions', visible: (view) => view.conclusions.length > 0 },
  { key: 'impacts', visible: (view) => view.conclusions.some((item) => item.impacts.length > 0) },
  { key: 'evidence', visible: (view) => view.conclusions.some((item) => item.evidence.length > 0) },
  { key: 'safety-note', visible: () => true }
];

export function getVisibleHomeSections(view: DailyBriefHomeView) {
  return homeSectionRegistry.filter((section) => section.visible(view));
}
