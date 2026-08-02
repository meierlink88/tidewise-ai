import {
  ResearchThemeDetailError,
  type HomeResearchThemeFeed,
  type ResearchThemeDetail,
  type ResearchThemeDetailErrorKind,
  type ResearchThemeFeedPort
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
  selectedThemeId: string | null;
  detailsByThemeId: Record<string, ResearchThemeDetailState>;
}

type Listener = (state: ResearchThemeHomeSessionState) => void;

export class ResearchThemeHomeSession {
  private state: ResearchThemeHomeSessionState = {
    feed: { status: 'idle' },
    selectedThemeId: null,
    detailsByThemeId: {}
  };
  private readonly listeners = new Set<Listener>();
  private disposed = false;
  private refreshInFlight = false;
  private detailGeneration = 0;
  private readonly detailRequests = new Set<string>();

  constructor(private readonly port: ResearchThemeFeedPort) {}

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
      const value = await this.port.list();
      if (this.disposed) return;
      this.update({ ...this.state, feed: { status: 'ready', value } });
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
      const value = await this.port.list();
      if (this.disposed) return 'ignored';
      const selectedThemeId =
        this.state.selectedThemeId !== null &&
        value.items.some(
          (theme) => theme.id === this.state.selectedThemeId && theme.evidenceEventCount > 0
        )
          ? this.state.selectedThemeId
          : null;
      this.detailGeneration += 1;
      this.update({
        ...this.state,
        feed: { status: 'ready', value },
        selectedThemeId,
        detailsByThemeId: {}
      });
      if (selectedThemeId !== null) void this.ensureDetail(selectedThemeId);
      return 'updated';
    } catch {
      return this.disposed ? 'ignored' : 'failed';
    } finally {
      this.refreshInFlight = false;
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
