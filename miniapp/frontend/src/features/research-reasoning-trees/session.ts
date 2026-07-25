import {
  ResearchReasoningTreeError,
  type ResearchReasoningTreeErrorKind,
  type ResearchReasoningTreeDetail,
  type ResearchReasoningTreeIndex,
  type ResearchReasoningTreePort
} from './contract';

const uuidPattern = /^[0-9a-f]{8}-[0-9a-f]{4}-[1-5][0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$/;

export type ResearchReasoningTreeIndexState =
  | { status: 'idle' }
  | { status: 'loading' }
  | { status: 'ready'; value: ResearchReasoningTreeIndex }
  | { status: 'themeUnavailable' }
  | { status: 'treesNotPublished' }
  | { status: 'error' };

export type ResearchReasoningTreeDetailState =
  | { status: 'idle' }
  | { status: 'loading' }
  | { status: 'ready'; value: ResearchReasoningTreeDetail }
  | { status: 'error'; errorKind: ResearchReasoningTreeErrorKind };

export interface ResearchReasoningTreeSessionState {
  routeStatus: 'valid' | 'invalid';
  index: ResearchReasoningTreeIndexState;
  selectedAnchorId: string | null;
  detailsByAnchorId: Record<string, ResearchReasoningTreeDetailState>;
}

type Listener = (state: ResearchReasoningTreeSessionState) => void;

export class ResearchReasoningTreeSession {
  private state: ResearchReasoningTreeSessionState;
  private readonly listeners = new Set<Listener>();
  private disposed = false;

  constructor(
    private readonly themeId: string,
    private readonly port: ResearchReasoningTreePort
  ) {
    this.state = {
      routeStatus: isLowercaseUUID(themeId) ? 'valid' : 'invalid',
      index: { status: 'idle' },
      selectedAnchorId: null,
      detailsByAnchorId: {}
    };
  }

  getState(): ResearchReasoningTreeSessionState {
    return this.state;
  }

  subscribe(listener: Listener): () => void {
    this.listeners.add(listener);
    listener(this.state);
    return () => this.listeners.delete(listener);
  }

  async start(): Promise<void> {
    if (this.state.routeStatus === 'invalid' || this.state.index.status !== 'idle') return;
    await this.loadIndex();
  }

  retryIndex(): void {
    if (this.state.routeStatus === 'valid' && this.state.index.status === 'error') {
      void this.loadIndex();
    }
  }

  selectAnchor(anchorId: string): void {
    if (!this.hasAnchor(anchorId)) return;
    this.update({ ...this.state, selectedAnchorId: anchorId });
    void this.ensureDetail(anchorId);
  }

  retryAnchor(anchorId: string): void {
    if (!this.hasAnchor(anchorId) || this.state.detailsByAnchorId[anchorId]?.status !== 'error')
      return;
    this.setDetail(anchorId, { status: 'idle' });
    void this.ensureDetail(anchorId);
  }

  dispose(): void {
    this.disposed = true;
    this.listeners.clear();
  }

  private async loadIndex(): Promise<void> {
    this.update({
      ...this.state,
      index: { status: 'loading' },
      selectedAnchorId: null,
      detailsByAnchorId: {}
    });
    try {
      const value = await this.port.list(this.themeId);
      if (this.disposed) return;
      if (value.theme.id !== this.themeId || value.reasoningTrees.length === 0) {
        throw new ResearchReasoningTreeError('serviceUnavailable');
      }
      this.update({ ...this.state, index: { status: 'ready', value } });
      this.selectAnchor(value.reasoningTrees[0].anchorId);
    } catch (error) {
      if (this.disposed) return;
      const kind = errorKind(error);
      const status =
        kind === 'themeUnavailable'
          ? 'themeUnavailable'
          : kind === 'treesNotPublished'
            ? 'treesNotPublished'
            : 'error';
      this.update({
        ...this.state,
        index: { status },
        selectedAnchorId: null,
        detailsByAnchorId: {}
      });
    }
  }

  private async ensureDetail(anchorId: string): Promise<void> {
    const current = this.state.detailsByAnchorId[anchorId];
    if (current?.status === 'loading' || current?.status === 'ready') return;
    this.setDetail(anchorId, { status: 'loading' });
    try {
      const value = await this.port.get(this.themeId, anchorId);
      if (this.disposed) return;
      this.setDetail(anchorId, { status: 'ready', value });
    } catch (error) {
      if (this.disposed) return;
      const kind = errorKind(error);
      if (kind === 'themeUnavailable' || kind === 'treesNotPublished') {
        this.update({
          ...this.state,
          index: { status: kind },
          selectedAnchorId: null,
          detailsByAnchorId: {}
        });
        return;
      }
      this.setDetail(anchorId, { status: 'error', errorKind: kind });
    }
  }

  private hasAnchor(anchorId: string): boolean {
    return (
      this.state.index.status === 'ready' &&
      this.state.index.value.reasoningTrees.some((tree) => tree.anchorId === anchorId)
    );
  }

  private setDetail(anchorId: string, detail: ResearchReasoningTreeDetailState): void {
    this.update({
      ...this.state,
      detailsByAnchorId: { ...this.state.detailsByAnchorId, [anchorId]: detail }
    });
  }

  private update(next: ResearchReasoningTreeSessionState): void {
    if (this.disposed) return;
    this.state = next;
    for (const listener of this.listeners) listener(next);
  }
}

export function isLowercaseUUID(value: string): boolean {
  return uuidPattern.test(value);
}

function errorKind(error: unknown): ResearchReasoningTreeErrorKind {
  return error instanceof ResearchReasoningTreeError ? error.kind : 'serviceUnavailable';
}
