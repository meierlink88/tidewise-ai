import {
  ResearchThemeDetailError,
  type HomeResearchThemeFeed,
  type ResearchThemeHomepagePort
} from './contract';
import {
  normalizeMiniappAPIBaseURL,
  type MiniappAPIEnvelope,
  unwrapMiniappAPIEnvelope
} from '../../platform/miniapp-api';
import { parseResearchThemeDetailWire, parseResearchThemeWire } from './wire-contract';

const themesPath = '/api/miniapp/v1/research/themes';
export interface ResearchThemeRequestOptions {
  url: string;
  method: 'GET';
  data: { window_hours: number; limit?: number };
  dataType: 'json';
  timeout: number;
}
export interface ResearchThemeRequestResult<T> {
  statusCode: number;
  data: T;
}
export type ResearchThemeRequest = <T>(
  options: ResearchThemeRequestOptions
) => Promise<ResearchThemeRequestResult<T>>;

interface APIFeed {
  window_start: string;
  window_end: string;
  as_of: string;
  theme_count: number;
  event_count: number;
  items: unknown[];
  next_cursor: string | null;
}
interface APIOptions {
  baseUrl: string;
  request: ResearchThemeRequest;
  windowHours?: number;
  requestTimeoutMs?: number;
}

export function createResearchThemeApiPort({
  baseUrl,
  request,
  windowHours = 24,
  requestTimeoutMs = 10_000
}: APIOptions): ResearchThemeHomepagePort {
  const base = normalizeMiniappAPIBaseURL(baseUrl);
  if (!Number.isInteger(windowHours) || windowHours < 1 || windowHours > 168)
    throw new Error('Research Theme window hours must be an integer between 1 and 168');
  if (!Number.isInteger(requestTimeoutMs) || requestTimeoutMs < 1)
    throw new Error('Research Theme request timeout must be a positive integer');
  return {
    async list() {
      const response = await request<MiniappAPIEnvelope<APIFeed>>({
        url: base + themesPath,
        method: 'GET',
        data: { window_hours: windowHours, limit: 20 },
        dataType: 'json',
        timeout: requestTimeoutMs
      });
      if (response.statusCode < 200 || response.statusCode >= 300)
        throw new Error(`Miniapp research API returned HTTP ${response.statusCode}`);
      const feed = unwrapMiniappAPIEnvelope<APIFeed>(response.data);
      if (!isFeed(feed)) throw new Error('Miniapp research API returned an invalid response');
      return mapFeed(feed);
    },
    async getDetail(themeId) {
      let response: ResearchThemeRequestResult<MiniappAPIEnvelope<unknown>>;
      try {
        response = await request<MiniappAPIEnvelope<unknown>>({
          url: `${base}${themesPath}/${encodeURIComponent(themeId)}`,
          method: 'GET',
          data: { window_hours: windowHours },
          dataType: 'json',
          timeout: requestTimeoutMs
        });
      } catch {
        throw new ResearchThemeDetailError('serviceUnavailable');
      }
      if (response.statusCode < 200 || response.statusCode >= 300) {
        const kind =
          response.statusCode === 404 && errorCode(response.data) === 'RESEARCH_THEME_NOT_FOUND'
            ? 'themeUnavailable'
            : 'serviceUnavailable';
        throw new ResearchThemeDetailError(kind);
      }
      const detail = unwrapMiniappAPIEnvelope<unknown>(response.data);
      if (detail === undefined)
        throw new Error('Miniapp research API returned an invalid response');
      return parseResearchThemeDetailWire(detail, themeId);
    }
  };
}

function errorCode(value: unknown): string | null {
  if (typeof value !== 'object' || value === null) return null;
  const error = (value as Record<string, unknown>).error;
  if (typeof error !== 'object' || error === null) return null;
  const code = (error as Record<string, unknown>).code;
  return typeof code === 'string' ? code : null;
}

function isFeed(value: unknown): value is APIFeed {
  return (
    typeof value === 'object' &&
    value !== null &&
    Array.isArray((value as APIFeed).items) &&
    typeof (value as APIFeed).as_of === 'string'
  );
}
function mapFeed(feed: APIFeed): HomeResearchThemeFeed {
  return {
    windowStart: feed.window_start,
    windowEnd: feed.window_end,
    asOf: feed.as_of,
    themeCount: feed.theme_count,
    eventCount: feed.event_count,
    items: feed.items.map((item) => parseResearchThemeWire(item, feed.as_of)),
    nextCursor: feed.next_cursor
  };
}
