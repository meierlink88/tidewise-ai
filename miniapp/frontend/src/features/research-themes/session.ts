import {
  ResearchThemeDetailError,
  type HomeResearchThemeFeed,
  type ResearchThemeDetail,
  type ResearchThemeDetailErrorKind,
  type ResearchThemeHomepagePort,
  type ResearchThemePeriod
} from './contract';

export type ResearchThemeFeedState =
  | { status: 'idle' }
  | { status: 'loading' }
  | { status: 'ready'; value: HomeResearchThemeFeed }
  | { status: 'error' };

export type ResearchThemeDetailState =
  | { status: 'loading' }
  | { status: 'ready'; value: ResearchThemeDetail }
  | { status: 'error'; errorKind: ResearchThemeDetailErrorKind };

export interface ResearchThemeHomeSessionState {
  feed: ResearchThemeFeedState;
  pagination: 'idle' | 'loading' | 'error' | 'exhausted';
  selectedThemeId: string | null;
  detailsByThemeId: Record<string, ResearchThemeDetailState>;
}

type Listener = (state: ResearchThemeHomeSessionState) => void;

export class ResearchThemeHomeSession {
  private state: ResearchThemeHomeSessionState = {
    feed: { status: 'idle' },
    pagination: 'idle',
    selectedThemeId: null,
    detailsByThemeId: {}
  };
  private readonly listeners = new Set<Listener>();
  private disposed = false;
  private refreshInFlight = false;
  private loadMoreInFlight = false;
  private feedGeneration = 0;
  private detailGeneration = 0;
  private readonly detailRequests = new Set<string>();

  private readonly period: ResearchThemePeriod;
  private readonly pageSize: number;

  constructor(
    private readonly port: ResearchThemeHomepagePort,
    options: { period?: ResearchThemePeriod; pageSize?: number } = {}
  ) {
    this.period = options.period ?? 'today';
    this.pageSize = options.pageSize ?? (this.period === 'history' ? 5 : 20);
  }

  getState(): ResearchThemeHomeSessionState {
    return this.state;
  }

  subscribe(listener: Listener): () => void {
    this.listeners.add(listener);
    listener(this.state);
    return () => this.listeners.delete(listener);
  }

  async start(): Promise<void> {
    if (this.state.feed.status !== 'idle') return;
    this.update({ ...this.state, feed: { status: 'loading' } });
    try {
      const value = await this.port.list({ period: this.period, limit: this.pageSize });
      if (this.disposed) return;
      this.update({
        ...this.state,
        feed: { status: 'ready', value },
        pagination: value.nextCursor === null ? 'exhausted' : 'idle'
      });
    } catch {
      if (this.disposed) return;
      this.update({ ...this.state, feed: { status: 'error' } });
    }
  }

  async retryFeed(): Promise<void> {
    if (this.disposed || this.state.feed.status !== 'error') return;
    this.update({ ...this.state, feed: { status: 'loading' } });
    try {
      const value = await this.port.list({ period: this.period, limit: this.pageSize });
      if (this.disposed) return;
      this.update({
        ...this.state,
        feed: { status: 'ready', value },
        pagination: value.nextCursor === null ? 'exhausted' : 'idle'
      });
    } catch {
      if (this.disposed) return;
      this.update({ ...this.state, feed: { status: 'error' } });
    }
  }

  async refreshFeed(): Promise<'updated' | 'failed' | 'ignored'> {
    if (this.disposed || this.state.feed.status !== 'ready' || this.refreshInFlight) {
      return 'ignored';
    }
    this.refreshInFlight = true;
    try {
      const value = await this.port.list({ period: this.period, limit: this.pageSize });
      if (this.disposed) return 'ignored';
      this.feedGeneration += 1;
      this.detailGeneration += 1;
      this.update({
        ...this.state,
        feed: { status: 'ready', value },
        pagination: value.nextCursor === null ? 'exhausted' : 'idle',
        selectedThemeId: null,
        detailsByThemeId: {}
      });
      return 'updated';
    } catch {
      return this.disposed ? 'ignored' : 'failed';
    } finally {
      this.refreshInFlight = false;
    }
  }

  async loadMore(): Promise<'updated' | 'failed' | 'ignored' | 'exhausted'> {
    if (
      this.disposed ||
      this.state.feed.status !== 'ready' ||
      this.loadMoreInFlight ||
      this.state.pagination === 'exhausted'
    ) {
      return this.state.pagination === 'exhausted' ? 'exhausted' : 'ignored';
    }
    const cursor = this.state.feed.value.nextCursor;
    if (cursor === null) {
      this.update({ ...this.state, pagination: 'exhausted' });
      return 'exhausted';
    }
    this.loadMoreInFlight = true;
    const generation = this.feedGeneration;
    this.update({ ...this.state, pagination: 'loading' });
    try {
      const page = await this.port.list({
        period: this.period,
        limit: this.pageSize,
        cursor
      });
      if (this.disposed || generation !== this.feedGeneration || this.state.feed.status !== 'ready')
        return 'ignored';
      const existingIds = new Set(this.state.feed.value.items.map((item) => item.id));
      const items = [
        ...this.state.feed.value.items,
        ...page.items.filter((item) => !existingIds.has(item.id))
      ];
      this.update({
        ...this.state,
        feed: { status: 'ready', value: { ...page, items } },
        pagination: page.nextCursor === null ? 'exhausted' : 'idle'
      });
      return 'updated';
    } catch {
      if (this.disposed || generation !== this.feedGeneration) return 'ignored';
      this.update({ ...this.state, pagination: 'error' });
      return 'failed';
    } finally {
      this.loadMoreInFlight = false;
    }
  }

  openThemeEvents(themeId: string): void {
    if (
      this.state.feed.status !== 'ready' ||
      !this.state.feed.value.items.some(
        (theme) => theme.id === themeId && theme.evidenceEventCount > 0
      )
    ) {
      return;
    }
    this.update({ ...this.state, selectedThemeId: themeId });
    void this.ensureDetail(themeId);
  }

  closeThemeEvents(): void {
    if (this.state.selectedThemeId === null) return;
    this.update({ ...this.state, selectedThemeId: null });
  }

  retryThemeEvents(): void {
    const themeId = this.state.selectedThemeId;
    if (themeId === null || this.state.detailsByThemeId[themeId]?.status !== 'error') return;
    void this.ensureDetail(themeId);
  }

  dispose(): void {
    this.disposed = true;
    this.listeners.clear();
  }

  private async ensureDetail(themeId: string): Promise<void> {
    const current = this.state.detailsByThemeId[themeId];
    const generation = this.detailGeneration;
    const requestKey = `${generation}:${themeId}`;
    if (
      current?.status === 'loading' ||
      current?.status === 'ready' ||
      this.detailRequests.has(requestKey)
    ) {
      return;
    }
    this.detailRequests.add(requestKey);
    this.setDetail(themeId, { status: 'loading' });
    try {
      const value = await this.port.getDetail(themeId);
      if (this.disposed || generation !== this.detailGeneration) return;
      if (value.id !== themeId) throw new ResearchThemeDetailError('serviceUnavailable');
      this.setDetail(themeId, { status: 'ready', value });
    } catch (error) {
      if (this.disposed || generation !== this.detailGeneration) return;
      const errorKind =
        error instanceof ResearchThemeDetailError ? error.kind : 'serviceUnavailable';
      this.setDetail(themeId, { status: 'error', errorKind });
    } finally {
      this.detailRequests.delete(requestKey);
    }
  }

  private setDetail(themeId: string, detail: ResearchThemeDetailState): void {
    this.update({
      ...this.state,
      detailsByThemeId: { ...this.state.detailsByThemeId, [themeId]: detail }
    });
  }

  private update(next: ResearchThemeHomeSessionState): void {
    if (this.disposed) return;
    this.state = next;
    for (const listener of this.listeners) listener(next);
  }
}
