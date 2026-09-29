import Taro from '@tarojs/taro';
import { normalizeMiniappAPIBaseURL, unwrapMiniappAPIEnvelope } from '../../platform/miniapp-api';
import {
  TrackingError,
  type Company,
  type Page,
  type TrackingPort,
  type FilterOptions
} from './contract';

const idPattern = /^STK[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$/;
function object(v: unknown): v is Record<string, unknown> {
  return v !== null && typeof v === 'object';
}
export function isCompany(v: unknown): v is Company {
  return (
    object(v) &&
    typeof v.id === 'string' &&
    idPattern.test(v.id) &&
    ['title', 'stock_name', 'symbol', 'industry_label', 'industry_path'].every(
      (k) => typeof v[k] === 'string'
    ) &&
    v.title !== '' &&
    v.stock_name !== '' &&
    typeof v.is_followed === 'boolean' &&
    Array.isArray(v.concepts) &&
    v.concepts.every((x) => typeof x === 'string' && x.length > 0)
  );
}
export function parsePage(v: unknown): Page {
  if (
    !object(v) ||
    !Array.isArray(v.items) ||
    v.items.length > 100 ||
    !v.items.every(isCompany) ||
    new Set(v.items.map((x) => x.id)).size !== v.items.length ||
    !Number.isSafeInteger(v.total) ||
    typeof v.total !== 'number' ||
    v.total < 0 ||
    typeof v.next_cursor !== 'string' ||
    v.next_cursor.length > 512 ||
    typeof v.has_more !== 'boolean'
  )
    throw new TrackingError('服务响应异常，请重试');
  return { items: v.items, total: v.total, next_cursor: v.next_cursor, has_more: v.has_more };
}
async function request(
  path: string,
  token: string,
  method: 'GET' | 'PUT' | 'DELETE' = 'GET'
): Promise<unknown> {
  let response: { statusCode: number; data: unknown };
  try {
    const base = normalizeMiniappAPIBaseURL(process.env.TARO_APP_MINIAPP_API_BASE_URL || '');
    response = await Taro.request({
      url: base + '/api/miniapp/v1/tracking' + path,
      method,
      timeout: 12000,
      header: token ? { Authorization: 'Bearer ' + token } : {}
    });
  } catch {
    throw new TrackingError('暂时无法连接，请稍后重试');
  }
  if (
    response.statusCode === 401 ||
    (response.statusCode === 403 &&
      object(response.data) &&
      object(response.data.error) &&
      response.data.error.code === 'USER_DISABLED')
  )
    throw new TrackingError('登录已失效，请重新登录', true);
  if (response.statusCode !== 200)
    throw new TrackingError(
      response.statusCode === 404 ? '未找到这家公司，请重新搜索' : '跟踪服务暂不可用，请重试'
    );
  const result = unwrapMiniappAPIEnvelope<unknown>(response.data);
  if (result === undefined) throw new TrackingError('服务响应异常，请重试');
  return result;
}
export const trackingAPI: TrackingPort = {
  async search(query, token, offset, filters) {
    const parameters = filters
      ? Object.entries(filters)
          .filter(([, ids]) => ids.length > 0)
          .map(([key, ids]) => '&' + key + '=' + encodeURIComponent(ids.join(',')))
          .join('')
      : '';
    return parsePage(
      await request(
        '/search?q=' + encodeURIComponent(query) + '&page_size=20&offset=' + offset + parameters,
        token
      )
    );
  },
  async list(token, cursor) {
    return parsePage(
      await request(
        '?page_size=20' + (cursor ? '&cursor=' + encodeURIComponent(cursor) : ''),
        token
      )
    );
  },
  async change(token, id, add) {
    if (!idPattern.test(id)) throw new TrackingError('公司信息异常，请重新搜索');
    const result = await request('/' + id, token, add ? 'PUT' : 'DELETE');
    if (!object(result) || result.id !== id || result.is_followed !== add)
      throw new TrackingError('操作结果异常，请刷新确认');
  }
};

export function parseFilterOptions(v: unknown): FilterOptions {
  const valid = (items: unknown, prefix: string): items is { id: string; name: string }[] =>
    Array.isArray(items) &&
    items.length <= 5000 &&
    items.every(
      (x) =>
        object(x) &&
        typeof x.id === 'string' &&
        new RegExp(
          '^' + prefix + '[0-9a-f]{8}-[0-9a-f]{4}-[1-5][0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$'
        ).test(x.id) &&
        typeof x.name === 'string' &&
        x.name.trim() !== ''
    ) &&
    new Set(items.map((x) => x.id)).size === items.length;
  if (
    !object(v) ||
    !valid(v.concepts, 'SCON') ||
    !valid(v.industry_chains, 'SICH') ||
    !valid(v.industries, 'SIND')
  )
    throw new TrackingError('筛选选项暂不可用，请重试');
  const industries = v.industries.map((x) => {
    const item: unknown = x;
    if (!object(item) || !valid(item.children, 'SIND'))
      throw new TrackingError('行业选项异常，请重试');
    return { id: x.id, name: x.name, children: item.children };
  });
  const ids = industries.flatMap((x) => [x.id, ...x.children.map((y) => y.id)]);
  if (new Set(ids).size !== ids.length) throw new TrackingError('行业选项异常，请重试');
  return { industries, concepts: v.concepts, industry_chains: v.industry_chains };
}
export async function loadFilterOptions(): Promise<FilterOptions> {
  return parseFilterOptions(await request('/filters', ''));
}
