import { ReportError } from './contract';

export type ReportResourceState<T> =
  | { status: 'idle' }
  | { status: 'loading' }
  | { status: 'empty'; refreshing: boolean; refreshFailed: boolean }
  | { status: 'ready'; data: T; refreshing: boolean; refreshFailed: boolean }
  | { status: 'error'; error: ReportError };

export class ReportResourceSession<T> {
  private generation = 0;
  private state: ReportResourceState<T> = { status: 'idle' };
  private readonly listeners = new Set<(state: ReportResourceState<T>) => void>();

  constructor(
    private readonly isEmpty: (value: T) => boolean = () => false,
    public readonly resourceKey = 'report-resource'
  ) {}

  snapshot(): ReportResourceState<T> {
    return this.state;
  }

  subscribe(listener: (state: ReportResourceState<T>) => void): () => void {
    this.listeners.add(listener);
    return () => this.listeners.delete(listener);
  }

  async load(loader: () => Promise<T>, refresh = false): Promise<void> {
    const request = ++this.generation;
    const previous = this.state;
    if (refresh && previous.status === 'ready') {
      this.publish({ ...previous, refreshing: true, refreshFailed: false });
    } else if (refresh && previous.status === 'empty') {
      this.publish({ ...previous, refreshing: true, refreshFailed: false });
    } else {
      this.publish({ status: 'loading' });
    }

    try {
      const value = await loader();
      if (request !== this.generation) return;
      this.publish(
        this.isEmpty(value)
          ? { status: 'empty', refreshing: false, refreshFailed: false }
          : { status: 'ready', data: value, refreshing: false, refreshFailed: false }
      );
    } catch (error) {
      if (request !== this.generation) return;
      if (refresh && previous.status === 'ready') {
        this.publish({ ...previous, refreshing: false, refreshFailed: true });
        return;
      }
      if (refresh && previous.status === 'empty') {
        this.publish({ ...previous, refreshing: false, refreshFailed: true });
        return;
      }
      this.publish({ status: 'error', error: normalizeReportError(error) });
    }
  }

  reset(): void {
    this.generation += 1;
    this.publish({ status: 'idle' });
  }

  dispose(): void {
    this.generation += 1;
    this.listeners.clear();
  }

  private publish(state: ReportResourceState<T>): void {
    this.state = state;
    this.listeners.forEach((listener) => listener(state));
  }
}

function normalizeReportError(error: unknown): ReportError {
  return error instanceof ReportError ? error : new ReportError('serviceUnavailable');
}
