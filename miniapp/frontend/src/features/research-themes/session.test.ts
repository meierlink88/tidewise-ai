import { describe, expect, it, vi } from 'vitest';
import {
  mockResearchThemeDetail,
  mockResearchThemeFeed
} from '../../mocks/research-themes/mock-port';
import { ResearchThemeDetailError, type ResearchThemeHomepagePort } from './contract';
import { ResearchThemeHomeSession } from './session';

describe('research theme homepage session', () => {
  it('loads the initial Theme feed into a ready page state', async () => {
    const port: ResearchThemeHomepagePort = {
      list: vi.fn().mockResolvedValue(mockResearchThemeFeed),
      getDetail: vi.fn()
    };
    const session = new ResearchThemeHomeSession(port);

    await session.start();

    expect(session.getState()).toMatchObject({
      feed: { status: 'ready', value: mockResearchThemeFeed },
      selectedThemeId: null,
      detailsByThemeId: {}
    });
    expect(port.list).toHaveBeenCalledOnce();
    expect(port.list).toHaveBeenCalledWith({ period: 'today', limit: 20 });
  });

  it('loads history in five-item pages and appends the next cursor page', async () => {
    const first = { ...mockResearchThemeFeed, nextCursor: 'history-cursor' };
    const secondItem = {
      ...mockResearchThemeFeed.items[0],
      id: 'bbbbbbbb-1111-4111-8111-111111111111',
      title: '历史主题'
    };
    const second = {
      ...mockResearchThemeFeed,
      items: [mockResearchThemeFeed.items[0], secondItem],
      nextCursor: null
    };
    const port: ResearchThemeHomepagePort = {
      list: vi.fn().mockResolvedValueOnce(first).mockResolvedValueOnce(second),
      getDetail: vi.fn()
    };
    const session = new ResearchThemeHomeSession(port, { period: 'history' });
    await session.start();

    await expect(session.loadMore()).resolves.toBe('updated');

    expect(port.list).toHaveBeenNthCalledWith(1, { period: 'history', limit: 5 });
    expect(port.list).toHaveBeenNthCalledWith(2, {
      period: 'history',
      limit: 5,
      cursor: 'history-cursor'
    });
    expect(session.getState().feed).toMatchObject({
      status: 'ready',
      value: { items: [mockResearchThemeFeed.items[0], secondItem], nextCursor: null }
    });
    expect(session.getState().pagination).toBe('exhausted');
  });

  it('coalesces concurrent history bottom events into one cursor request', async () => {
    const nextPage = deferred<typeof mockResearchThemeFeed>();
    const first = { ...mockResearchThemeFeed, nextCursor: 'one-cursor' };
    const port: ResearchThemeHomepagePort = {
      list: vi.fn().mockResolvedValueOnce(first).mockReturnValueOnce(nextPage.promise),
      getDetail: vi.fn()
    };
    const session = new ResearchThemeHomeSession(port, { period: 'history' });
    await session.start();

    const firstBottom = session.loadMore();
    await expect(session.loadMore()).resolves.toBe('ignored');
    nextPage.resolve({ ...mockResearchThemeFeed, items: [], nextCursor: null });

    await expect(firstBottom).resolves.toBe('updated');
    expect(port.list).toHaveBeenCalledTimes(2);
  });

  it('preserves loaded history and retries a failed next page from the same cursor', async () => {
    const first = { ...mockResearchThemeFeed, nextCursor: 'retry-cursor' };
    const port: ResearchThemeHomepagePort = {
      list: vi
        .fn()
        .mockResolvedValueOnce(first)
        .mockRejectedValueOnce(new Error('temporary'))
        .mockResolvedValueOnce({ ...mockResearchThemeFeed, items: [], nextCursor: null }),
      getDetail: vi.fn()
    };
    const session = new ResearchThemeHomeSession(port, { period: 'history' });
    await session.start();

    await expect(session.loadMore()).resolves.toBe('failed');
    expect(session.getState().feed).toEqual({ status: 'ready', value: first });
    expect(session.getState().pagination).toBe('error');
    await expect(session.loadMore()).resolves.toBe('updated');

    expect(port.list).toHaveBeenNthCalledWith(2, {
      period: 'history',
      limit: 5,
      cursor: 'retry-cursor'
    });
    expect(port.list).toHaveBeenNthCalledWith(3, {
      period: 'history',
      limit: 5,
      cursor: 'retry-cursor'
    });
  });

  it('retries an initial feed failure from the visible error state', async () => {
    const port: ResearchThemeHomepagePort = {
      list: vi.fn().mockRejectedValueOnce(new Error()).mockResolvedValueOnce(mockResearchThemeFeed),
      getDetail: vi.fn()
    };
    const session = new ResearchThemeHomeSession(port);

    await session.start();
    expect(session.getState().feed).toEqual({ status: 'error' });

    await session.retryFeed();

    expect(session.getState().feed).toEqual({ status: 'ready', value: mockResearchThemeFeed });
    expect(port.list).toHaveBeenCalledTimes(2);
  });

  it('deduplicates overlapping pull-down refresh requests', async () => {
    const refresh = deferred<typeof mockResearchThemeFeed>();
    const refreshedFeed = { ...mockResearchThemeFeed, asOf: '2026-07-28T09:00:00Z' };
    const port: ResearchThemeHomepagePort = {
      list: vi
        .fn()
        .mockResolvedValueOnce(mockResearchThemeFeed)
        .mockReturnValueOnce(refresh.promise),
      getDetail: vi.fn()
    };
    const session = new ResearchThemeHomeSession(port);
    await session.start();

    const firstRefresh = session.refreshFeed();
    const duplicateResult = await session.refreshFeed();
    refresh.resolve(refreshedFeed);

    await expect(firstRefresh).resolves.toBe('updated');
    expect(duplicateResult).toBe('ignored');
    expect(port.list).toHaveBeenCalledTimes(2);
    expect(session.getState().feed).toEqual({ status: 'ready', value: refreshedFeed });
  });

  it('opens and caches the selected Theme event timeline for the page session', async () => {
    const port: ResearchThemeHomepagePort = {
      list: vi.fn().mockResolvedValue(mockResearchThemeFeed),
      getDetail: vi.fn().mockResolvedValue(mockResearchThemeDetail)
    };
    const session = new ResearchThemeHomeSession(port);
    await session.start();

    session.openThemeEvents(mockResearchThemeDetail.id);
    await flushPromises();

    expect(session.getState()).toMatchObject({
      selectedThemeId: mockResearchThemeDetail.id,
      detailsByThemeId: {
        [mockResearchThemeDetail.id]: { status: 'ready', value: mockResearchThemeDetail }
      }
    });
    session.closeThemeEvents();
    session.openThemeEvents(mockResearchThemeDetail.id);
    expect(port.getDetail).toHaveBeenCalledOnce();
  });

  it('keeps a detail failure inside the sheet and retries only that Theme', async () => {
    const port: ResearchThemeHomepagePort = {
      list: vi.fn().mockResolvedValue(mockResearchThemeFeed),
      getDetail: vi
        .fn()
        .mockRejectedValueOnce(new ResearchThemeDetailError('themeUnavailable'))
        .mockResolvedValueOnce(mockResearchThemeDetail)
    };
    const session = new ResearchThemeHomeSession(port);
    await session.start();

    session.openThemeEvents(mockResearchThemeDetail.id);
    await flushPromises();
    expect(session.getState().detailsByThemeId[mockResearchThemeDetail.id]).toEqual({
      status: 'error',
      errorKind: 'themeUnavailable'
    });

    session.retryThemeEvents();
    await flushPromises();

    expect(session.getState().detailsByThemeId[mockResearchThemeDetail.id]).toEqual({
      status: 'ready',
      value: mockResearchThemeDetail
    });
    expect(port.getDetail).toHaveBeenCalledTimes(2);
  });

  it('does not open an event sheet when the Theme count is zero', async () => {
    const feed = {
      ...mockResearchThemeFeed,
      items: [{ ...mockResearchThemeFeed.items[0], evidenceEventCount: 0 }]
    };
    const port: ResearchThemeHomepagePort = {
      list: vi.fn().mockResolvedValue(feed),
      getDetail: vi.fn()
    };
    const session = new ResearchThemeHomeSession(port);
    await session.start();

    session.openThemeEvents(feed.items[0].id);

    expect(session.getState().selectedThemeId).toBeNull();
    expect(port.getDetail).not.toHaveBeenCalled();
  });

  it('invalidates the detail cache after a successful feed refresh', async () => {
    const port: ResearchThemeHomepagePort = {
      list: vi.fn().mockResolvedValue(mockResearchThemeFeed),
      getDetail: vi.fn().mockResolvedValue(mockResearchThemeDetail)
    };
    const session = new ResearchThemeHomeSession(port);
    await session.start();
    session.openThemeEvents(mockResearchThemeDetail.id);
    await flushPromises();
    session.closeThemeEvents();

    await session.refreshFeed();
    session.openThemeEvents(mockResearchThemeDetail.id);
    await flushPromises();

    expect(port.getDetail).toHaveBeenCalledTimes(2);
  });

  it('closes an open event sheet and clears its detail cache after feed refresh', async () => {
    const port: ResearchThemeHomepagePort = {
      list: vi.fn().mockResolvedValue(mockResearchThemeFeed),
      getDetail: vi.fn().mockResolvedValue(mockResearchThemeDetail)
    };
    const session = new ResearchThemeHomeSession(port);
    await session.start();
    session.openThemeEvents(mockResearchThemeDetail.id);
    await flushPromises();

    await session.refreshFeed();

    expect(session.getState().selectedThemeId).toBeNull();
    expect(session.getState().detailsByThemeId).toEqual({});
    expect(port.getDetail).toHaveBeenCalledOnce();
  });

  it('ignores an old load-more response after pull-to-refresh restarts page one', async () => {
    const oldPage = deferred<typeof mockResearchThemeFeed>();
    const first = { ...mockResearchThemeFeed, nextCursor: 'old-cursor' };
    const refreshed = { ...mockResearchThemeFeed, asOf: '2026-08-04T03:00:00Z' };
    const port: ResearchThemeHomepagePort = {
      list: vi
        .fn()
        .mockResolvedValueOnce(first)
        .mockReturnValueOnce(oldPage.promise)
        .mockResolvedValueOnce(refreshed),
      getDetail: vi.fn()
    };
    const session = new ResearchThemeHomeSession(port, { period: 'history' });
    await session.start();
    const loadingMore = session.loadMore();

    await expect(session.refreshFeed()).resolves.toBe('updated');
    oldPage.resolve({
      ...mockResearchThemeFeed,
      items: [{ ...mockResearchThemeFeed.items[0], id: 'stale-theme' }],
      nextCursor: null
    });

    await expect(loadingMore).resolves.toBe('ignored');
    expect(session.getState().feed).toEqual({ status: 'ready', value: refreshed });
  });

  it('ignores an old load-more failure after pull-to-refresh restarts page one', async () => {
    const oldPage = deferred<typeof mockResearchThemeFeed>();
    const first = { ...mockResearchThemeFeed, nextCursor: 'old-cursor' };
    const refreshed = { ...mockResearchThemeFeed, asOf: '2026-08-04T03:00:00Z' };
    const port: ResearchThemeHomepagePort = {
      list: vi
        .fn()
        .mockResolvedValueOnce(first)
        .mockReturnValueOnce(oldPage.promise)
        .mockResolvedValueOnce(refreshed),
      getDetail: vi.fn()
    };
    const session = new ResearchThemeHomeSession(port, { period: 'history' });
    await session.start();
    const loadingMore = session.loadMore();

    await session.refreshFeed();
    oldPage.reject(new Error('stale failure'));

    await expect(loadingMore).resolves.toBe('ignored');
    expect(session.getState().pagination).toBe('exhausted');
  });

  it('keeps a late response keyed to its Theme when the user opens another Theme', async () => {
    const otherTheme = {
      ...mockResearchThemeFeed.items[0],
      id: 'bbbbbbbb-1111-4111-8111-111111111111',
      title: '另一条推理主线',
      evidenceEventCount: 1
    };
    const otherDetail = {
      id: otherTheme.id,
      title: otherTheme.title,
      events: [mockResearchThemeDetail.events[0]]
    };
    const firstRequest = deferred<typeof mockResearchThemeDetail>();
    const secondRequest = deferred<typeof otherDetail>();
    const port: ResearchThemeHomepagePort = {
      list: vi.fn().mockResolvedValue({
        ...mockResearchThemeFeed,
        themeCount: 2,
        items: [mockResearchThemeFeed.items[0], otherTheme]
      }),
      getDetail: vi.fn((themeId: string) =>
        themeId === mockResearchThemeDetail.id ? firstRequest.promise : secondRequest.promise
      )
    };
    const session = new ResearchThemeHomeSession(port);
    await session.start();

    session.openThemeEvents(mockResearchThemeDetail.id);
    session.openThemeEvents(otherTheme.id);
    firstRequest.resolve(mockResearchThemeDetail);
    await flushPromises();

    expect(session.getState().selectedThemeId).toBe(otherTheme.id);
    expect(session.getState().detailsByThemeId[otherTheme.id]).toEqual({ status: 'loading' });

    secondRequest.resolve(otherDetail);
    await flushPromises();
    expect(session.getState().detailsByThemeId[otherTheme.id]).toEqual({
      status: 'ready',
      value: otherDetail
    });
  });
});

function deferred<T>() {
  let resolve!: (value: T) => void;
  let reject!: (error: unknown) => void;
  const promise = new Promise<T>((next, fail) => {
    resolve = next;
    reject = fail;
  });
  return { promise, resolve, reject };
}

async function flushPromises() {
  await Promise.resolve();
  await Promise.resolve();
}
