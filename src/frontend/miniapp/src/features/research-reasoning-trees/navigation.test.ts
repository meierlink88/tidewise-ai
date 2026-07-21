import { describe, expect, it, vi } from 'vitest';
import { navigateToResearchReasoningTrees, researchReasoningTreeRoute } from './navigation';

const themeId = '11111111-1111-4111-8111-111111111111';

describe('research reasoning tree navigation', () => {
  it('builds the fixed non-tabBar route with the Theme UUID', () => {
    expect(researchReasoningTreeRoute(themeId)).toBe(
      `/pages/research-theme/reasoning-trees/index?theme_id=${themeId}`
    );
  });

  it('delegates navigation through the supplied Taro-compatible seam', async () => {
    const navigateTo = vi.fn().mockResolvedValue(undefined);

    await navigateToResearchReasoningTrees(themeId, navigateTo);

    expect(navigateTo).toHaveBeenCalledWith({
      url: `/pages/research-theme/reasoning-trees/index?theme_id=${themeId}`
    });
  });

  it('rejects a non-canonical Theme ID before navigation', async () => {
    const navigateTo = vi.fn();

    await expect(
      navigateToResearchReasoningTrees('11111111-1111-4111-8111-11111111111A', navigateTo)
    ).rejects.toThrow('invalid Theme ID');
    expect(navigateTo).not.toHaveBeenCalled();
  });
});
