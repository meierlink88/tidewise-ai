import type {
  HomeResearchChainNode,
  HomeResearchIndex,
  HomeResearchThemeFeed,
  HomeResearchThemeItem,
  ResearchImpactLevel,
  ResearchThemeFeedPort,
  ResearchTransmissionStage
} from './contract';
import { formatResearchUpdateLabel } from './presentation';

const themesPath = '/api/v1/miniapp/research/themes';

export interface ResearchThemeRequestOptions {
  url: string;
  method: 'GET';
  data: { window_hours: number; limit: number };
  dataType: 'json';
}

export interface ResearchThemeRequestResult<T> {
  statusCode: number;
  data: T;
}

export type ResearchThemeRequest = <T>(options: ResearchThemeRequestOptions) => Promise<ResearchThemeRequestResult<T>>;

interface APIResearchChainNode {
  id: string;
  name: string;
  relation_role: HomeResearchChainNode['relationRole'];
  impact_summary: string;
}

interface APIResearchIndex {
  id: string;
  name: string;
  impact_direction: HomeResearchIndex['impactDirection'];
  impact_summary: string;
}

interface APIResearchTheme {
  id: string;
  name: string;
  one_line_conclusion: string;
  impact_level: ResearchImpactLevel;
  transmission_path: string;
  trading_direction: string;
  transmission_stage: ResearchTransmissionStage;
  next_checkpoint: string;
  index_impact_summary: string;
  published_at: string;
  affected_chain_nodes: APIResearchChainNode[];
  related_indices: APIResearchIndex[];
  supporting_event_count: number;
  contradicting_event_count: number;
}

interface APIResearchThemeFeed {
  window_start: string;
  window_end: string;
  as_of: string;
  theme_count: number;
  event_count: number;
  items: APIResearchTheme[];
  next_cursor: string | null;
}

interface APIOptions {
  baseUrl: string;
  request: ResearchThemeRequest;
}

export function createResearchThemeApiPort({ baseUrl, request }: APIOptions): ResearchThemeFeedPort {
  const normalizedBaseUrl = normalizeBaseUrl(baseUrl);
  return {
    async list() {
      const response = await request<APIResearchThemeFeed>({
        url: normalizedBaseUrl + themesPath,
        method: 'GET',
        data: { window_hours: 24, limit: 20 },
        dataType: 'json'
      });
      if (response.statusCode < 200 || response.statusCode >= 300) {
        throw new Error(`Miniapp research API returned HTTP ${response.statusCode}`);
      }
      if (!isThemeFeed(response.data)) {
        throw new Error('Miniapp research API returned an invalid response');
      }
      return mapFeed(response.data);
    }
  };
}

function normalizeBaseUrl(value: string): string {
  const normalized = value.trim().replace(/\/+$/, '');
  if (!/^https?:\/\/[^/]+/i.test(normalized)) {
    throw new Error('TARO_APP_MINIAPP_API_BASE_URL must be an absolute HTTP(S) URL in api mode');
  }
  return normalized;
}

function isThemeFeed(value: unknown): value is APIResearchThemeFeed {
  if (typeof value !== 'object' || value === null) return false;
  const feed = value as Partial<APIResearchThemeFeed>;
  return (
    typeof feed.window_start === 'string' &&
    typeof feed.window_end === 'string' &&
    typeof feed.as_of === 'string' &&
    typeof feed.theme_count === 'number' &&
    typeof feed.event_count === 'number' &&
    Array.isArray(feed.items) &&
    (typeof feed.next_cursor === 'string' || feed.next_cursor === null)
  );
}

function mapFeed(feed: APIResearchThemeFeed): HomeResearchThemeFeed {
  return {
    windowStart: feed.window_start,
    windowEnd: feed.window_end,
    asOf: feed.as_of,
    themeCount: feed.theme_count,
    eventCount: feed.event_count,
    trackingCount: 17,
    items: feed.items.map((item) => mapTheme(item, feed.as_of)),
    nextCursor: feed.next_cursor
  };
}

function mapTheme(theme: APIResearchTheme, asOf: string): HomeResearchThemeItem {
  return {
    id: theme.id,
    name: theme.name,
    oneLineConclusion: theme.one_line_conclusion,
    impactLevel: theme.impact_level,
    transmissionPath: theme.transmission_path,
    tradingDirection: theme.trading_direction,
    transmissionStage: theme.transmission_stage,
    nextCheckpoint: theme.next_checkpoint,
    indexImpactSummary: theme.index_impact_summary,
    publishedAt: theme.published_at,
    updateLabel: formatResearchUpdateLabel(theme.published_at, asOf),
    categories: categoriesForTheme(theme.name),
    affectedChainNodes: theme.affected_chain_nodes.map((node) => ({
      id: node.id,
      name: node.name,
      relationRole: node.relation_role,
      impactSummary: node.impact_summary
    })),
    relatedIndices: theme.related_indices.map((index) => ({
      id: index.id,
      name: index.name,
      impactDirection: index.impact_direction,
      impactSummary: index.impact_summary
    })),
    supportingEventCount: theme.supporting_event_count,
    contradictingEventCount: theme.contradicting_event_count
  };
}

function categoriesForTheme(name: string): string[] {
  if (name.includes('中东') || name.includes('冲突')) return ['地缘政治'];
  if (name.includes('AI') || name.includes('人工智能') || name.includes('半导体')) return ['算力基建'];
  return ['货币政策'];
}
