import { expect, it, vi } from 'vitest';
import { parsePage, parseFilterOptions, trackingAPI } from './api';

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

it('validates filter catalogs and serializes combined selection to the BFF', async () => {
  const i = 'SIND11111111-1111-5111-8111-111111111111';
  const c = 'SCON11111111-1111-5111-8111-111111111111';
  const h = 'SICH11111111-1111-5111-8111-111111111111';
  const options = {
    industries: [{ id: i, name: '行业', children: [] }],
    concepts: [{ id: c, name: '概念' }],
    industry_chains: [{ id: h, name: '产业链' }]
  };
  expect(parseFilterOptions(options)).toEqual(options);
  expect(() =>
    parseFilterOptions({ ...options, concepts: [{ id: i, name: '错误类型' }] })
  ).toThrow();
  expect(() =>
    parseFilterOptions({
      ...options,
      industries: [{ id: i, name: '行业', children: [{ id: i, name: '重复' }] }]
    })
  ).toThrow();
  request.mockResolvedValueOnce({
    statusCode: 200,
    data: {
      request_id: 'filter',
      result: { items: [], total: 0, next_cursor: '', has_more: false }
    }
  });
  await trackingAPI.search('银行', '', 20, {
    industry_ids: [i],
    concept_ids: [c],
    industry_chain_ids: [h]
  });
  const url = new URL(request.mock.lastCall![0].url);
  expect(url.pathname).toBe('/api/miniapp/v1/tracking/search');
  expect(url.searchParams.get('industry_ids')).toBe(i);
  expect(url.searchParams.get('concept_ids')).toBe(c);
  expect(url.searchParams.get('industry_chain_ids')).toBe(h);
  expect(url.searchParams.get('offset')).toBe('20');
});
