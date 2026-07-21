import { describe, expect, it, vi } from 'vitest';
import { ResearchReasoningTreeError } from './contract';
import type { ResearchReasoningTreeDetail, ResearchReasoningTreePort } from './contract';
import { createMockResearchReasoningTreePort } from '../../mocks/research-reasoning-trees/mock-port';
import { ResearchReasoningTreeSession } from './session';

const themeId = '11111111-1111-4111-8111-111111111111';

describe('research reasoning tree page session', () => {
  it('rejects an invalid route parameter without calling the Port', async () => {
    const port: ResearchReasoningTreePort = { list: vi.fn(), get: vi.fn() };
    const session = new ResearchReasoningTreeSession('11111111-1111-4111-8111-11111111111A', port);

    await session.start();

    expect(session.getState()).toMatchObject({ routeStatus: 'invalid', index: { status: 'idle' } });
    expect(port.list).not.toHaveBeenCalled();
    expect(port.get).not.toHaveBeenCalled();
  });

  it('loads the index, selects the first stable tab and loads its detail', async () => {
    const source = createMockResearchReasoningTreePort();
    const index = await source.list(themeId);
    const first = await source.get(themeId, index.reasoningTrees[0].anchorId);
    const port: ResearchReasoningTreePort = {
      list: vi.fn().mockResolvedValue(index),
      get: vi.fn().mockResolvedValue(first)
    };
    const session = new ResearchReasoningTreeSession(themeId, port);

    await session.start();
    await flushPromises();

    expect(session.getState()).toMatchObject({
      routeStatus: 'valid',
      index: { status: 'ready', value: index },
      selectedAnchorId: index.reasoningTrees[0].anchorId,
      detailsByAnchorId: {
        [index.reasoningTrees[0].anchorId]: { status: 'ready', value: first }
      }
    });
    expect(port.get).toHaveBeenCalledOnce();
  });

  it('keeps concurrent tab results isolated and reuses successful session cache', async () => {
    const source = createMockResearchReasoningTreePort();
    const index = await source.list(themeId);
    const firstDetail = await source.get(themeId, index.reasoningTrees[0].anchorId);
    const secondDetail = await source.get(themeId, index.reasoningTrees[1].anchorId);
    const first = deferred<ResearchReasoningTreeDetail>();
    const second = deferred<ResearchReasoningTreeDetail>();
    const get = vi
      .fn()
      .mockImplementationOnce(() => first.promise)
      .mockImplementationOnce(() => second.promise);
    const session = new ResearchReasoningTreeSession(themeId, {
      list: vi.fn().mockResolvedValue(index),
      get
    });

    await session.start();
    session.selectAnchor(index.reasoningTrees[1].anchorId);
    second.resolve(secondDetail);
    await flushPromises();

    expect(session.getState()).toMatchObject({
      selectedAnchorId: index.reasoningTrees[1].anchorId,
      detailsByAnchorId: {
        [index.reasoningTrees[0].anchorId]: { status: 'loading' },
        [index.reasoningTrees[1].anchorId]: { status: 'ready', value: secondDetail }
      }
    });

    first.resolve(firstDetail);
    await flushPromises();
    expect(session.getState().selectedAnchorId).toBe(index.reasoningTrees[1].anchorId);
    expect(session.getState().detailsByAnchorId[index.reasoningTrees[0].anchorId]).toMatchObject({
      status: 'ready',
      value: firstDetail
    });

    session.selectAnchor(index.reasoningTrees[0].anchorId);
    session.selectAnchor(index.reasoningTrees[1].anchorId);
    expect(get).toHaveBeenCalledTimes(2);
  });

  it('retries only a failed tab and preserves the loaded index and other detail', async () => {
    const source = createMockResearchReasoningTreePort();
    const index = await source.list(themeId);
    const firstDetail = await source.get(themeId, index.reasoningTrees[0].anchorId);
    const secondDetail = await source.get(themeId, index.reasoningTrees[1].anchorId);
    const list = vi.fn().mockResolvedValue(index);
    const get = vi
      .fn()
      .mockResolvedValueOnce(firstDetail)
      .mockRejectedValueOnce(new ResearchReasoningTreeError('treeUnavailable'))
      .mockResolvedValueOnce(secondDetail);
    const session = new ResearchReasoningTreeSession(themeId, { list, get });

    await session.start();
    await flushPromises();
    session.selectAnchor(index.reasoningTrees[1].anchorId);
    await flushPromises();
    expect(session.getState().detailsByAnchorId[index.reasoningTrees[1].anchorId]).toMatchObject({
      status: 'error',
      errorKind: 'treeUnavailable'
    });

    session.retryAnchor(index.reasoningTrees[1].anchorId);
    await flushPromises();

    expect(list).toHaveBeenCalledOnce();
    expect(get).toHaveBeenCalledTimes(3);
    expect(session.getState().detailsByAnchorId[index.reasoningTrees[0].anchorId]).toMatchObject({
      status: 'ready',
      value: firstDetail
    });
    expect(session.getState().detailsByAnchorId[index.reasoningTrees[1].anchorId]).toMatchObject({
      status: 'ready',
      value: secondDetail
    });
  });

  it('promotes a missing Theme during detail loading to the page-level unavailable state', async () => {
    const source = createMockResearchReasoningTreePort();
    const index = await source.list(themeId);
    const session = new ResearchReasoningTreeSession(themeId, {
      list: vi.fn().mockResolvedValue(index),
      get: vi.fn().mockRejectedValue(new ResearchReasoningTreeError('themeUnavailable'))
    });

    await session.start();
    await flushPromises();

    expect(session.getState()).toMatchObject({
      index: { status: 'themeUnavailable' },
      selectedAnchorId: null,
      detailsByAnchorId: {}
    });
  });

  it.each([
    ['themeUnavailable', 'themeUnavailable'],
    ['treesNotPublished', 'treesNotPublished'],
    ['serviceUnavailable', 'error']
  ] as const)('maps %s list failures to the %s page state', async (kind, status) => {
    const session = new ResearchReasoningTreeSession(themeId, {
      list: vi.fn().mockRejectedValue(new ResearchReasoningTreeError(kind)),
      get: vi.fn()
    });

    await session.start();

    expect(session.getState().index.status).toBe(status);
  });
});

function deferred<T>() {
  let resolve!: (value: T) => void;
  let reject!: (reason?: unknown) => void;
  const promise = new Promise<T>((nextResolve, nextReject) => {
    resolve = nextResolve;
    reject = nextReject;
  });
  return { promise, resolve, reject };
}

async function flushPromises() {
  await Promise.resolve();
  await Promise.resolve();
}
