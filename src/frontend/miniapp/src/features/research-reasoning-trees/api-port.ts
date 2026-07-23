import { ResearchReasoningTreeError, type ResearchReasoningTreePort } from './contract';
import {
  parseResearchReasoningTreeDetail,
  parseResearchReasoningTreeIndex
} from './wire-contract';
import {
  normalizeMiniappAPIBaseURL,
  unwrapMiniappAPIEnvelope
} from '../../platform/miniapp-api';

const themesPath = '/api/miniapp/v1/research/themes';

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
  const normalizedBaseUrl = normalizeMiniappAPIBaseURL(baseUrl);

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
  let response: ResearchReasoningTreeRequestResult<unknown>;
  try {
    response = await request<unknown>({ url, method: 'GET', dataType: 'json' });
  } catch {
    throw new ResearchReasoningTreeError('serviceUnavailable');
  }
  if (response.statusCode < 200 || response.statusCode >= 300) {
    throw errorFromResponse(response.statusCode, response.data);
  }
  const result = unwrapMiniappAPIEnvelope<T>(response.data);
  if (result === undefined) {
    throw new ResearchReasoningTreeError('serviceUnavailable');
  }
  return result;
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
