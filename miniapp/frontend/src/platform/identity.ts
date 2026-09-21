import Taro from '@tarojs/taro';
import { isSession, type Session } from '../features/identity/session';

const storageKey = 'tidewise.identity.session.v1';
export const supportsWechatLogin = process.env.TARO_ENV === 'weapp';
export async function openProfile() {
  try {
    await Taro.navigateTo({ url: '/pages/profile/index' });
  } catch {
    void Taro.showToast({ title: '打开失败，请重试', icon: 'none' });
  }
}
export async function leaveProfile() {
  try {
    if (Taro.getCurrentPages().length > 1) await Taro.navigateBack({ delta: 1 });
    else await Taro.reLaunch({ url: '/pages/index/index' });
  } catch {
    void Taro.showToast({ title: '返回失败，请重试', icon: 'none' });
  }
}
export async function requestWechatCode(): Promise<string> {
  if (!supportsWechatLogin) throw new Error('请在微信小程序中登录');
  const result = await Taro.login({ timeout: 10000 });
  if (!result.code) throw new Error('微信登录未完成，请重试');
  return result.code;
}
export function readSession(): Session | null {
  try {
    const value: unknown = Taro.getStorageSync(storageKey);
    if (isSession(value)) return value;
    Taro.removeStorageSync(storageKey);
  } catch {
    /* Unavailable storage is treated as a guest session. */
  }
  return null;
}
export function saveSession(session: Session): void {
  Taro.setStorageSync(storageKey, session);
}
export function clearSession(): void {
  Taro.removeStorageSync(storageKey);
}

export async function confirmLogout(): Promise<boolean> {
  const result = await Taro.showModal({
    title: '退出登录？',
    content: '退出后仍可浏览推理内容，个人资料会保留。',
    confirmText: '退出登录',
    cancelText: '取消',
    confirmColor: '#0b1f33'
  });
  return result.confirm === true;
}

export async function openProfileInformation(section: 'privacy' | 'about') {
  try {
    await Taro.navigateTo({ url: `/pages/profile/information/index?section=${section}` });
  } catch {
    void Taro.showToast({ title: '打开失败，请重试', icon: 'none' });
  }
}
export async function copyPrivacyContact() {
  try {
    await Taro.setClipboardData({ data: 'media22@tidetell.cn' });
  } catch {
    void Taro.showToast({ title: '复制失败，请重试', icon: 'none' });
  }
}

export async function openLogin() {
  try {
    await Taro.navigateTo({ url: '/pages/login/index' });
  } catch {
    void Taro.showToast({ title: '打开失败，请重试', icon: 'none' });
  }
}
export async function leaveLogin() {
  try {
    if (Taro.getCurrentPages().length > 1) await Taro.navigateBack({ delta: 1 });
    else await Taro.redirectTo({ url: '/pages/profile/index' });
  } catch {
    void Taro.showToast({ title: '返回失败，请重试', icon: 'none' });
  }
}
