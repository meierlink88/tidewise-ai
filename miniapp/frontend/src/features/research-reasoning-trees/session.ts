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
  selectedReasoningTreeId: string | null;
  detailsByReasoningTreeId: Record<string, ResearchReasoningTreeDetailState>;
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
      selectedReasoningTreeId: null,
      detailsByReasoningTreeId: {}
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

  selectReasoningTree(reasoningTreeId: string): void {
    if (!this.hasReasoningTree(reasoningTreeId)) return;
    this.update({ ...this.state, selectedReasoningTreeId: reasoningTreeId });
    void this.ensureDetail(reasoningTreeId);
  }

  retryReasoningTree(reasoningTreeId: string): void {
    if (!this.hasReasoningTree(reasoningTreeId) || this.state.detailsByReasoningTreeId[reasoningTreeId]?.status !== 'error')
      return;
    this.setDetail(reasoningTreeId, { status: 'idle' });
    void this.ensureDetail(reasoningTreeId);
  }

  dispose(): void {
    this.disposed = true;
    this.listeners.clear();
  }

  private async loadIndex(): Promise<void> {
    this.update({
      ...this.state,
      index: { status: 'loading' },
      selectedReasoningTreeId: null,
      detailsByReasoningTreeId: {}
    });
    try {
      const value = await this.port.list(this.themeId);
      if (this.disposed) return;
      if (value.theme.id !== this.themeId) {
        throw new ResearchReasoningTreeError('serviceUnavailable');
      }
      if (value.reasoningTrees.length === 0) {
        throw new ResearchReasoningTreeError('treesNotPublished');
      }
      this.update({ ...this.state, index: { status: 'ready', value } });
      this.selectReasoningTree(value.reasoningTrees[0].reasoningTreeId);
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
        selectedReasoningTreeId: null,
        detailsByReasoningTreeId: {}
      });
    }
  }

  private async ensureDetail(reasoningTreeId: string): Promise<void> {
    const current = this.state.detailsByReasoningTreeId[reasoningTreeId];
    if (current?.status === 'loading' || current?.status === 'ready') return;
    this.setDetail(reasoningTreeId, { status: 'loading' });
    try {
      const value = await this.port.get(this.themeId, reasoningTreeId);
      if (this.disposed) return;
      this.setDetail(reasoningTreeId, { status: 'ready', value });
    } catch (error) {
      if (this.disposed) return;
      const kind = errorKind(error);
      if (kind === 'themeUnavailable' || kind === 'treesNotPublished') {
        this.update({
          ...this.state,
          index: { status: kind },
          selectedReasoningTreeId: null,
          detailsByReasoningTreeId: {}
        });
        return;
      }
      this.setDetail(reasoningTreeId, { status: 'error', errorKind: kind });
    }
  }

  private hasReasoningTree(reasoningTreeId: string): boolean {
    return (
      this.state.index.status === 'ready' &&
      this.state.index.value.reasoningTrees.some((tree) => tree.reasoningTreeId === reasoningTreeId)
    );
  }

  private setDetail(reasoningTreeId: string, detail: ResearchReasoningTreeDetailState): void {
    this.update({
      ...this.state,
      detailsByReasoningTreeId: { ...this.state.detailsByReasoningTreeId, [reasoningTreeId]: detail }
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
