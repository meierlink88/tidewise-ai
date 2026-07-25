import Taro from '@tarojs/taro';
import type { ResearchThemeFeedPort } from './contract';
import {
  createResearchThemeApiPort,
  type ResearchThemeRequestOptions,
  type ResearchThemeRequestResult
} from './api-port';
import { createMockResearchThemeFeedPort } from '../../mocks/research-themes/mock-port';

export function createResearchThemeFeedPort(): ResearchThemeFeedPort {
  const source = process.env.TARO_APP_RESEARCH_SOURCE;
  if (source === 'mock') {
    return createMockResearchThemeFeedPort();
  }
  if (source === 'api') {
    return createResearchThemeApiPort({
      baseUrl: process.env.TARO_APP_MINIAPP_API_BASE_URL ?? '',
      request: taroRequest,
      windowHours: Number(process.env.TARO_APP_RESEARCH_WINDOW_HOURS ?? '24')
    });
  }
  throw new Error(`Unsupported TARO_APP_RESEARCH_SOURCE: ${source}`);
}

async function taroRequest<T>(options: ResearchThemeRequestOptions): Promise<ResearchThemeRequestResult<T>> {
  const response = await Taro.request<T, typeof options.data>(options);
  return { statusCode: response.statusCode, data: response.data };
}
