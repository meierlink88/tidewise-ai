import { describe, expect, it } from 'vitest';
import { createMockResearchThemeFeedPort } from '../research-themes/mock-port';
import { createMockResearchReasoningTreePort } from './mock-port';

const themeId = '11111111-1111-4111-8111-111111111111';

describe('research reasoning tree mock Port', () => {
  it('serves the frozen shared list and both detail fixtures', async () => {
    const port = createMockResearchReasoningTreePort();
    const index = await port.list(themeId);

    expect(index.reasoningTrees.map((tree) => tree.centerChainNode.name)).toEqual([
      '先进封装',
      '光模块'
    ]);

    const details = await Promise.all(
      index.reasoningTrees.map((tree) => port.get(themeId, tree.anchorId))
    );
    expect(details.map((detail) => detail.reasoningTree.anchorId)).toEqual(
      index.reasoningTrees.map((tree) => tree.anchorId)
    );
    expect(
      details[0].reasoningTree.events.some((event) => event.evidenceRole === 'contradicting')
    ).toBe(false);
    expect(
      details[1].reasoningTree.events.some((event) => event.evidenceRole === 'contradicting')
    ).toBe(true);
  });

  it('uses the same stable missing-resource semantics as the API Port', async () => {
    const port = createMockResearchReasoningTreePort();

    await expect(port.list('dddddddd-dddd-4ddd-8ddd-dddddddddddd')).rejects.toMatchObject({
      kind: 'themeUnavailable'
    });
    await expect(port.get(themeId, 'aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa')).rejects.toMatchObject({
      kind: 'treeUnavailable'
    });
  });

  it('provides a usable reasoning-tree skeleton for every homepage mock Theme', async () => {
    const themes = await createMockResearchThemeFeedPort().list();
    const port = createMockResearchReasoningTreePort();

    for (const theme of themes.items) {
      const index = await port.list(theme.id);
      expect(index.theme.id).toBe(theme.id);
      expect(index.reasoningTrees.length).toBeGreaterThan(0);

      const detail = await port.get(theme.id, index.reasoningTrees[0].anchorId);
      expect(detail.themeId).toBe(theme.id);
      expect(detail.reasoningTree.centerChainNode).toEqual(index.reasoningTrees[0].centerChainNode);
    }
  });
});
