import { describe, expect, it, vi } from 'vitest';
import { getHomeChromeMetrics } from './system-ui';

describe('home chrome metrics', () => {
  it('uses the modern window API and the native WeChat capsule geometry', () => {
    const metrics = getHomeChromeMetrics({
      getWindowInfo: () => ({ statusBarHeight: 44, windowWidth: 390 }),
      getMenuButtonBoundingClientRect: () => ({ top: 50, left: 300, width: 78, height: 32 })
    });

    expect(metrics).toEqual({
      statusBarHeight: 44,
      navigationBarHeight: 44,
      rightReservedWidth: 102
    });
  });

  it('falls back to the cross-platform system API when getWindowInfo is unavailable', () => {
    const getSystemInfoSync = vi.fn(() => ({ statusBarHeight: 24, windowWidth: 375 }));

    expect(getHomeChromeMetrics({ getSystemInfoSync })).toEqual({
      statusBarHeight: 24,
      navigationBarHeight: 44,
      rightReservedWidth: 16
    });
    expect(getSystemInfoSync).toHaveBeenCalledOnce();
  });

  it('falls back safely when a runtime exposes getWindowInfo but throws', () => {
    const getSystemInfoSync = vi.fn(() => ({ statusBarHeight: 20, windowWidth: 360 }));

    expect(
      getHomeChromeMetrics({
        getWindowInfo: () => {
          throw new Error('unsupported');
        },
        getSystemInfoSync
      })
    ).toMatchObject({ statusBarHeight: 20 });
    expect(getSystemInfoSync).toHaveBeenCalledOnce();
  });
});
