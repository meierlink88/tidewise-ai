import Taro from '@tarojs/taro';
import { isSession, type Session } from '../features/identity/session';

const storageKey = 'tidewise.identity.session.v1';
export const supportsWechatLogin = process.env.TARO_ENV === 'weapp';
export function openProfile() {
  return Taro.switchTab({ url: '/pages/profile/index' });
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
