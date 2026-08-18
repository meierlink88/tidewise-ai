import type { HomeResearchThemeItem } from './contract';
import { researchImpactStrengthLabel, researchTransmissionStageLabel } from './presentation';

export function filterHomeResearchThemes(
  items: HomeResearchThemeItem[],
  query: string
): HomeResearchThemeItem[] {
  const normalizedQuery = query.trim().toLocaleLowerCase();

  return items.filter((item) => {
    if (normalizedQuery.length === 0) return true;

    const searchableText = [
      item.title,
      item.oneLineConclusion,
      item.transmissionSummary ?? '',
      item.investmentGuidanceSummary,
      item.checkpointSummary ?? '',
      researchImpactStrengthLabel(item.impactStrength),
      researchTransmissionStageLabel(item.transmissionStage),
		...item.impacts.flatMap((node) => [node.displayName, node.impactSummary ?? ''])
    ]
      .join(' ')
      .toLocaleLowerCase();

    return searchableText.includes(normalizedQuery);
  });
}
