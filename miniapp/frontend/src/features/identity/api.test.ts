import { beforeEach, describe, expect, it, vi } from 'vitest';
import { login, me, logout, updateNickname } from './api';

const { request } = vi.hoisted(() => ({ request: vi.fn() }));
vi.mock('@tarojs/taro', () => ({ default: { request } }));
const profile = {
  user_id: 'f466d548-4ac3-4d3f-b994-cbeeb6eb3ef2',
  nickname: '观潮用户',
  status: 'active',
  expires_at: '2099-01-01T00:00:00Z'
};
const token = 'A'.repeat(43);
describe('identity API', () => {
  beforeEach(() => {
    request.mockReset();
    process.env.TARO_APP_MINIAPP_API_BASE_URL = 'https://miniapp.example.com';
  });
  it('sends the one-use code once and validates the login envelope', async () => {
    request.mockResolvedValue({
      statusCode: 200,
      data: { request_id: 'r', result: { ...profile, session_token: token } }
    });
    await expect(login('code')).resolves.toMatchObject({ session_token: token });
    expect(request).toHaveBeenCalledOnce();
    expect(request.mock.calls[0][0]).toMatchObject({ method: 'POST', data: { code: 'code' } });
    expect(request.mock.calls[0][0].header.Authorization).toBeUndefined();
  });
  it('puts the user token in the header and never in the URL', async () => {
    request.mockResolvedValue({ statusCode: 200, data: { request_id: 'r', result: profile } });
    await updateNickname(token, '新昵称');
    expect(request.mock.calls[0][0]).toMatchObject({
      method: 'PATCH',
      data: { nickname: '新昵称' },
      header: { Authorization: `Bearer ${token}` }
    });
    expect(request.mock.calls[0][0].url).not.toContain(token);
  });
  it('only invalidates a session on authentication errors', async () => {
    request.mockResolvedValue({ statusCode: 401, data: {} });
    await expect(me(token)).rejects.toMatchObject({ expired: true });
    request.mockResolvedValue({ statusCode: 503, data: {} });
    await expect(me(token)).rejects.toMatchObject({ expired: false });
    request.mockRejectedValue(new Error('private transport detail'));
    await expect(me(token)).rejects.toMatchObject({
      expired: false,
      message: '暂时无法连接，请稍后重试'
    });
  });
  it('requires confirmed revocation and never retries mutations', async () => {
    request.mockResolvedValue({
      statusCode: 200,
      data: { request_id: 'r', result: { revoked: false } }
    });
    await expect(logout(token)).rejects.toThrow('退出未完成');
    expect(request).toHaveBeenCalledOnce();
  });
});
