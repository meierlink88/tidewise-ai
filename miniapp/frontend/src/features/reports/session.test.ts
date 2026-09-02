import { describe, expect, it, vi } from 'vitest';
import { ReportError } from './contract';
import { ReportResourceSession } from './session';

describe('Report resource session', () => {
  it('ignores an older request that resolves after a newer request', async () => {
    const session = new ReportResourceSession<string>();
    let resolveFirst: (value: string) => void = () => undefined;
    const first = new Promise<string>((resolve) => {
      resolveFirst = resolve;
    });

    const firstLoad = session.load(() => first);
    const secondLoad = session.load(async () => 'new');
    await secondLoad;
    resolveFirst('old');
    await firstLoad;

    expect(session.snapshot()).toMatchObject({ status: 'ready', data: 'new' });
  });

  it('keeps successful content when a pull-to-refresh fails', async () => {
    const session = new ReportResourceSession<string>();
    const listener = vi.fn();
    session.subscribe(listener);
    await session.load(async () => 'cached');
    await session.load(async () => {
      throw new ReportError('serviceUnavailable');
    }, true);

    expect(session.snapshot()).toEqual({
      status: 'ready',
      data: 'cached',
      refreshing: false,
      refreshFailed: true
    });
    expect(listener).toHaveBeenCalled();
  });

  it('models a valid empty response separately from an error', async () => {
    const session = new ReportResourceSession<string[]>((items) => items.length === 0);
    await session.load(async () => []);
    expect(session.snapshot()).toMatchObject({ status: 'empty' });

    await session.load(async () => {
      throw new Error('network detail must not leak');
    });
    expect(session.snapshot()).toMatchObject({
      status: 'error',
      error: { kind: 'serviceUnavailable' }
    });
  });
});
