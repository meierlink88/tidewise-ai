import Taro from '@tarojs/taro';
import { privacyVersion } from './privacy';
import { normalizeMiniappAPIBaseURL, unwrapMiniappAPIEnvelope } from '../../platform/miniapp-api';
import { isProfile, isSession, type Profile, type Session } from './session';

export class IdentityError extends Error {
  constructor(
    message: string,
    readonly expired = false
  ) {
    super(message);
  }
}
async function request(
  path: string,
  method: 'GET' | 'POST' | 'PATCH',
  token?: string,
  data?: object
): Promise<unknown> {
  let response: { statusCode: number; data: unknown };
  try {
    const base = normalizeMiniappAPIBaseURL(process.env.TARO_APP_MINIAPP_API_BASE_URL || '');
    response = await Taro.request({
      url: `${base}/api/miniapp/v1/auth/${path}`,
      method,
      data,
      timeout: 12000,
      header: {
        'Content-Type': 'application/json',
        ...(token ? { Authorization: `Bearer ${token}` } : {})
      }
    });
  } catch {
    throw new IdentityError('暂时无法连接，请稍后重试');
  }
  if (
    response.statusCode === 401 ||
    (response.statusCode === 403 &&
      typeof response.data === 'object' &&
      response.data !== null &&
      'error' in response.data &&
      (response.data as { error?: { code?: string } }).error?.code === 'USER_DISABLED')
  )
    throw new IdentityError('登录已失效，请重新登录', true);
  if (response.statusCode !== 200) {
    throw new IdentityError(
      response.statusCode === 429
        ? '操作频繁，请稍后再试'
        : response.statusCode === 400
          ? '信息未通过验证，请重试'
          : '登录服务暂不可用，请稍后重试'
    );
  }
  const result = unwrapMiniappAPIEnvelope<unknown>(response.data);
  if (result === undefined) throw new IdentityError('服务响应异常，请稍后重试');
  return result;
}
function profile(value: unknown): Profile {
  if (!isProfile(value)) throw new IdentityError('用户信息异常，请稍后重试');
  return value;
}
export async function login(
  code: string,
  previous?: string,
  phoneCode?: string
): Promise<Profile & Session> {
  const value = await request('wechat/login', 'POST', previous, {
    code,
    privacy_version: privacyVersion,
    ...(phoneCode ? { phone_code: phoneCode } : {})
  });
  if (!isSession(value)) throw new IdentityError('登录响应异常，请重试');
  return { ...profile(value), ...value };
}
export async function me(token: string): Promise<Profile> {
  return profile(await request('me', 'GET', token));
}
export async function updateNickname(token: string, nickname: string): Promise<Profile> {
  return profile(await request('profile', 'PATCH', token, { nickname }));
}
export async function logout(token: string): Promise<void> {
  const result = await request('logout', 'POST', token);
  if (!result || typeof result !== 'object' || !('revoked' in result) || result.revoked !== true)
    throw new IdentityError('退出未完成，请重试');
}
