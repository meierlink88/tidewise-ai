import { describe, expect, it, vi } from 'vitest';
import { createMockResearchReasoningTreePort } from '../../mocks/research-reasoning-trees/mock-port';
import { ResearchReasoningTreeError } from './contract';
import type { ResearchReasoningTreePort } from './contract';
import { ResearchReasoningTreeSession } from './session';

const themeId = 'c26337f2-a79f-5089-84f4-63d57bc32230';

describe('research reasoning tree page session', () => {
  it('rejects an invalid route parameter without calling the Port', async () => {
    const port: ResearchReasoningTreePort = { list: vi.fn(), get: vi.fn() };
    const session = new ResearchReasoningTreeSession('INVALID', port);

    await session.start();

    expect(session.getState()).toMatchObject({ routeStatus: 'invalid', index: { status: 'idle' } });
    expect(port.list).not.toHaveBeenCalled();
  });

  it('loads the list, selects the first Reason Tree, and caches its detail', async () => {
    const source = createMockResearchReasoningTreePort();
    const index = await source.list(themeId);
    const firstId = index.reasoningTrees[0].reasoningTreeId;
    const first = await source.get(themeId, firstId);
    const port: ResearchReasoningTreePort = {
      list: vi.fn().mockResolvedValue(index),
      get: vi.fn().mockResolvedValue(first)
    };
    const session = new ResearchReasoningTreeSession(themeId, port);

    await session.start();
    await flushPromises();

    expect(session.getState()).toMatchObject({
      index: { status: 'ready', value: index },
      selectedReasoningTreeId: firstId,
      detailsByReasoningTreeId: { [firstId]: { status: 'ready', value: first } }
    });
    session.selectReasoningTree(firstId);
    expect(port.get).toHaveBeenCalledOnce();
  });

  it('treats an empty published-tree list as a legal Theme without a Tree detail', async () => {
    const source = createMockResearchReasoningTreePort();
    const index = await source.list(themeId);
    const session = new ResearchReasoningTreeSession(themeId, {
      list: vi.fn().mockResolvedValue({ ...index, reasoningTrees: [] }),
      get: vi.fn()
    });

    await session.start();

    expect(session.getState()).toMatchObject({
      index: { status: 'treesNotPublished' },
      selectedReasoningTreeId: null,
      detailsByReasoningTreeId: {}
    });
  });

  it.each([
    ['themeUnavailable', 'themeUnavailable'],
    ['treesNotPublished', 'treesNotPublished'],
    ['serviceUnavailable', 'error']
  ] as const)('maps %s list failures to %s', async (kind, status) => {
    const session = new ResearchReasoningTreeSession(themeId, {
      list: vi.fn().mockRejectedValue(new ResearchReasoningTreeError(kind)),
      get: vi.fn()
    });
    await session.start();
    expect(session.getState().index.status).toBe(status);
  });
});

async function flushPromises() {
  await Promise.resolve();
  await Promise.resolve();
}
