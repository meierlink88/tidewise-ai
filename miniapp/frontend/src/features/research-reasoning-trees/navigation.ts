import { isLowercaseUUID } from './session';

interface NavigateToOptions {
  url: string;
}

export type NavigateTo = (options: NavigateToOptions) => Promise<unknown>;

export function researchReasoningTreeRoute(themeId: string): string {
  if (!isLowercaseUUID(themeId)) throw new Error('invalid Theme ID');
  return `/pages/research-theme/reasoning-trees/index?theme_id=${themeId}`;
}

export async function navigateToResearchReasoningTrees(
  themeId: string,
  navigateTo: NavigateTo
): Promise<void> {
  await navigateTo({ url: researchReasoningTreeRoute(themeId) });
}
