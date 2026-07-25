import Taro from '@tarojs/taro';
import { createMockResearchReasoningTreePort } from '../../mocks/research-reasoning-trees/mock-port';
import {
  createResearchReasoningTreeApiPort,
  type ResearchReasoningTreeRequestOptions,
  type ResearchReasoningTreeRequestResult
} from './api-port';
import type { ResearchReasoningTreePort } from './contract';

export function createResearchReasoningTreePort(): ResearchReasoningTreePort {
  const source = process.env.TARO_APP_RESEARCH_SOURCE;
  if (source === 'mock') return createMockResearchReasoningTreePort();
  if (source === 'api') {
    return createResearchReasoningTreeApiPort({
      baseUrl: process.env.TARO_APP_MINIAPP_API_BASE_URL ?? '',
      request: taroRequest
    });
  }
  throw new Error(`Unsupported TARO_APP_RESEARCH_SOURCE: ${source}`);
}

async function taroRequest<T>(
  options: ResearchReasoningTreeRequestOptions
): Promise<ResearchReasoningTreeRequestResult<T>> {
  const response = await Taro.request<T>(options);
  return { statusCode: response.statusCode, data: response.data };
}
