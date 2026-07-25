import type { HomeResearchThemeItem } from './contract';
import { researchTransmissionStageLabel } from './presentation';

const ALL_CATEGORY = '全部';
const HOME_CATEGORY_ORDER = ['算力基建', '地缘政治', '贸易管制', '货币政策'];

export interface HomeThemeFilter {
  category: string;
  query: string;
}

export function getHomeThemeCategories(items: HomeResearchThemeItem[]): string[] {
  const categories = new Set<string>();

  for (const item of items) {
    for (const category of item.categories) {
      categories.add(category);
    }
  }

  const orderedCategories = HOME_CATEGORY_ORDER.filter((category) => categories.delete(category));
  const remainingCategories = [...categories].sort((left, right) => left.localeCompare(right, 'zh-CN'));

  return [ALL_CATEGORY, ...orderedCategories, ...remainingCategories];
}

export function filterHomeResearchThemes(
  items: HomeResearchThemeItem[],
  { category, query }: HomeThemeFilter
): HomeResearchThemeItem[] {
  const normalizedQuery = query.trim().toLocaleLowerCase();

  return items.filter((item) => {
    const matchesCategory = category === ALL_CATEGORY || item.categories.includes(category);
    if (!matchesCategory || normalizedQuery.length === 0) {
      return matchesCategory;
    }

    const searchableText = [
      item.name,
      item.oneLineConclusion,
      item.transmissionPath,
      item.tradingDirection,
      researchTransmissionStageLabel(item.transmissionStage),
      item.nextCheckpoint,
      ...item.categories,
      ...item.affectedChainNodes.flatMap((node) => [node.name, node.impactSummary])
    ]
      .join(' ')
      .toLocaleLowerCase();

    return searchableText.includes(normalizedQuery);
  });
}
