import { expect, it, vi } from 'vitest';
import { confirmLogout, openProfile, leaveProfile } from './identity';

const { showModal, switchTab, navigateBack, reLaunch, getCurrentPages, showToast } = vi.hoisted(
  () => ({
    showModal: vi.fn(),
    switchTab: vi.fn(),
    navigateBack: vi.fn(),
    reLaunch: vi.fn(),
    getCurrentPages: vi.fn(),
    showToast: vi.fn()
  })
);
vi.mock('@tarojs/taro', () => ({
  default: { showModal, switchTab, navigateBack, reLaunch, getCurrentPages, showToast }
}));
it('requires an explicit confirmation and treats cancellation as no logout', async () => {
  showModal.mockResolvedValueOnce({ confirm: false, cancel: true });
  expect(await confirmLogout()).toBe(false);
  showModal.mockResolvedValueOnce({ confirm: true, cancel: false });
  expect(await confirmLogout()).toBe(true);
  expect(showModal).toHaveBeenLastCalledWith(
    expect.objectContaining({ title: '退出登录？', confirmText: '退出登录', cancelText: '取消' })
  );
});

it('opens profile as a tab and returns through the existing stack', async () => {
  await openProfile();
  expect(switchTab).toHaveBeenCalledWith({ url: '/pages/profile/index' });
  getCurrentPages.mockReturnValue([{}, {}]);
  await leaveProfile();
  expect(navigateBack).toHaveBeenCalledWith({ delta: 1 });
  expect(reLaunch).not.toHaveBeenCalled();
});
it('returns directly opened profile to home and reports navigation failure', async () => {
  getCurrentPages.mockReturnValue([{}]);
  await leaveProfile();
  expect(reLaunch).toHaveBeenCalledWith({ url: '/pages/index/index' });
  switchTab.mockRejectedValueOnce(new Error('navigation failed'));
  await openProfile();
  expect(showToast).toHaveBeenCalledWith({ title: '打开失败，请重试', icon: 'none' });
});
