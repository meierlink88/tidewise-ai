import { expect, it, vi } from 'vitest';
import { parsePage, trackingAPI } from './api';

const request = vi.hoisted(() => vi.fn());
vi.mock('@tarojs/taro', () => ({ default: { request } }));
it('rejects invalid companies and duplicate IDs at the wire boundary', () => {
  const item = {
    id: 'STKf4a8eb61-c352-5980-91b1-9da6eba8f8af',
    title: '公司',
    stock_name: '简称',
    symbol: '000001.SZ',
    industry_label: '',
    industry_path: '',
    concepts: [],
    is_followed: false
  };
  const page = { items: [item], total: 1, next_cursor: '', has_more: false };
  expect(parsePage(page).items).toHaveLength(1);
  expect(() => parsePage({ ...page, items: [item, item] })).toThrow();
  expect(() => parsePage({ ...page, items: [{ ...item, concepts: [5] }] })).toThrow();
});
it('sends mutations to BFF using current bearer without a user ID or body', async () => {
  process.env.TARO_APP_MINIAPP_API_BASE_URL = 'http://localhost:9012';
  const id = 'STKf4a8eb61-c352-5980-91b1-9da6eba8f8af';
  request.mockResolvedValueOnce({
    statusCode: 200,
    data: { request_id: 'r', result: { id, is_followed: true } }
  });
  await trackingAPI.change('session', id, true);
  expect(request).toHaveBeenLastCalledWith(
    expect.objectContaining({
      method: 'PUT',
      header: { Authorization: 'Bearer session' },
      url: 'http://localhost:9012/api/miniapp/v1/tracking/' + id
    })
  );
  expect(request.mock.calls.at(-1)?.[0]).not.toHaveProperty('data');
  request.mockResolvedValueOnce({ statusCode: 401, data: {} });
  await expect(trackingAPI.list('expired', '')).rejects.toMatchObject({ expired: true });
});
