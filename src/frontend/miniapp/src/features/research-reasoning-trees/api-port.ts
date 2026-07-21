import { ResearchReasoningTreeError, type ResearchReasoningTreePort } from './contract';
import {
  parseResearchReasoningTreeDetail,
  parseResearchReasoningTreeIndex
} from './wire-contract';

const themesPath = '/api/v1/miniapp/research/themes';

export interface ResearchReasoningTreeRequestOptions {
  url: string;
  method: 'GET';
  dataType: 'json';
}

export interface ResearchReasoningTreeRequestResult<T> {
  statusCode: number;
  data: T;
}

export type ResearchReasoningTreeRequest = <T>(
  options: ResearchReasoningTreeRequestOptions
) => Promise<ResearchReasoningTreeRequestResult<T>>;

interface APIOptions {
  baseUrl: string;
  request: ResearchReasoningTreeRequest;
}

export function createResearchReasoningTreeApiPort({
  baseUrl,
  request
}: APIOptions): ResearchReasoningTreePort {
  const normalizedBaseUrl = normalizeBaseUrl(baseUrl);

  return {
    async list(themeId) {
      const response = await get<unknown>(
        request,
        normalizedBaseUrl + reasoningTreeListPath(themeId)
      );
      return parseResearchReasoningTreeIndex(response);
    },
    async get(themeId, anchorId) {
      const response = await get<unknown>(
        request,
        normalizedBaseUrl + reasoningTreeDetailPath(themeId, anchorId)
      );
      return parseResearchReasoningTreeDetail(response, themeId, anchorId);
    }
  };
}

async function get<T>(request: ResearchReasoningTreeRequest, url: string): Promise<T> {
  let response: ResearchReasoningTreeRequestResult<T>;
  try {
    response = await request<T>({ url, method: 'GET', dataType: 'json' });
  } catch {
    throw new ResearchReasoningTreeError('serviceUnavailable');
  }
  if (response.statusCode < 200 || response.statusCode >= 300) {
    throw errorFromResponse(response.statusCode, response.data);
  }
  return response.data;
}

function errorFromResponse(statusCode: number, data: unknown): ResearchReasoningTreeError {
  const code = errorCode(data);
  if (statusCode === 400 && code === 'INVALID_REQUEST')
    return new ResearchReasoningTreeError('invalidRequest');
  if (statusCode === 404 && code === 'RESEARCH_THEME_NOT_FOUND') {
    return new ResearchReasoningTreeError('themeUnavailable');
  }
  if (statusCode === 404 && code === 'RESEARCH_REASONING_TREES_NOT_FOUND') {
    return new ResearchReasoningTreeError('treesNotPublished');
  }
  if (statusCode === 404 && code === 'RESEARCH_REASONING_TREE_NOT_FOUND') {
    return new ResearchReasoningTreeError('treeUnavailable');
  }
  return new ResearchReasoningTreeError('serviceUnavailable');
}

function errorCode(data: unknown): string | null {
  if (!isRecord(data) || !isRecord(data.error) || typeof data.error.code !== 'string') return null;
  return data.error.code;
}

function isRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === 'object' && value !== null;
}

function reasoningTreeListPath(themeId: string): string {
  return `${themesPath}/${encodeURIComponent(themeId)}/reasoning-trees`;
}

function reasoningTreeDetailPath(themeId: string, anchorId: string): string {
  return `${reasoningTreeListPath(themeId)}/${encodeURIComponent(anchorId)}`;
}

function normalizeBaseUrl(value: string): string {
  const normalized = value.trim().replace(/\/+$/, '');
  if (!/^https?:\/\/[^/]+/i.test(normalized)) {
    throw new Error('TARO_APP_MINIAPP_API_BASE_URL must be an absolute HTTP(S) URL in api mode');
  }
  return normalized;
}
