import Taro from '@tarojs/taro';

// The native tab bar sits above page content on both supported mini-program platforms.
export function hideOverlayNavigation(): Promise<unknown> {
  return Taro.hideTabBar({ animation: false });
}
export async function restoreOverlayNavigation(): Promise<void> {
  try {
    await Taro.showTabBar({ animation: false });
  } catch {
    await Taro.showToast({ title: '导航恢复失败，请重新进入页面', icon: 'none' });
  }
}
