export interface HomeChromeMetrics {
  statusBarHeight: number;
  navigationBarHeight: number;
  rightReservedWidth: number;
}

interface WindowSnapshot {
  statusBarHeight?: number;
  windowWidth: number;
}

interface CapsuleSnapshot {
  top: number;
  left: number;
  width: number;
  height: number;
}

export interface SystemUIApi {
  getWindowInfo?: () => WindowSnapshot;
  getSystemInfoSync?: () => WindowSnapshot;
  getMenuButtonBoundingClientRect?: () => CapsuleSnapshot;
}

function readWindowSnapshot(api: SystemUIApi): WindowSnapshot {
  if (typeof api.getWindowInfo === 'function') {
    try {
      return api.getWindowInfo();
    } catch {
      // Continue to the cross-platform fallback.
    }
  }

  if (typeof api.getSystemInfoSync === 'function') {
    try {
      return api.getSystemInfoSync();
    } catch {
      // Use conservative metrics when neither platform API is available.
    }
  }

  return { statusBarHeight: 20, windowWidth: 375 };
}

export function getHomeChromeMetrics(api: SystemUIApi): HomeChromeMetrics {
  const windowInfo = readWindowSnapshot(api);
  const statusBarHeight = windowInfo.statusBarHeight ?? 20;
  let navigationBarHeight = 44;
  let rightReservedWidth = 16;

  if (typeof api.getMenuButtonBoundingClientRect === 'function') {
    try {
      const capsule = api.getMenuButtonBoundingClientRect();
      if (capsule.width > 0 && capsule.height > 0) {
        const capsuleGap = Math.max(capsule.top - statusBarHeight, 0);
        navigationBarHeight = capsule.height + capsuleGap * 2;
        rightReservedWidth = Math.max(windowInfo.windowWidth - capsule.left + 12, 16);
      }
    } catch {
      // ByteDance and preview runtimes may not expose the WeChat capsule API.
    }
  }

  return { statusBarHeight, navigationBarHeight, rightReservedWidth };
}
