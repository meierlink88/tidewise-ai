import { expect, it, vi } from 'vitest';
import { hideOverlayNavigation, restoreOverlayNavigation } from './overlay-navigation';

const mock = vi.hoisted(() => ({ hideTabBar: vi.fn(), showTabBar: vi.fn(), showToast: vi.fn() }));
vi.mock('@tarojs/taro', () => ({ default: mock }));
it('hides and restores native navigation with failure feedback', async () => {
  mock.hideTabBar.mockResolvedValueOnce({});
  await hideOverlayNavigation();
  expect(mock.hideTabBar).toHaveBeenLastCalledWith({ animation: false });
  mock.showTabBar.mockResolvedValueOnce({});
  await restoreOverlayNavigation();
  expect(mock.showTabBar).toHaveBeenLastCalledWith({ animation: false });
  mock.showTabBar.mockRejectedValueOnce(new Error('unavailable'));
  mock.showToast.mockResolvedValueOnce({});
  await restoreOverlayNavigation();
  expect(mock.showToast).toHaveBeenCalledWith(expect.objectContaining({ icon: 'none' }));
  mock.hideTabBar.mockRejectedValueOnce(new Error('unavailable'));
  await expect(hideOverlayNavigation()).rejects.toThrow('unavailable');
});
