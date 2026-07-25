import { afterEach, describe, expect, it, vi } from 'vitest';
import { createResearchReasoningTreePort } from './port';

vi.mock('@tarojs/taro', () => ({ default: { request: vi.fn() } }));

afterEach(() => {
  vi.unstubAllEnvs();
});

describe('research reasoning tree Port selection', () => {
  it('uses the shared fixture Mock Adapter in mock mode', async () => {
    vi.stubEnv('TARO_APP_RESEARCH_SOURCE', 'mock');
    const port = createResearchReasoningTreePort();

    const result = await port.list('11111111-1111-4111-8111-111111111111');

    expect(result.reasoningTrees).toHaveLength(2);
  });

  it('requires an absolute BFF URL in api mode instead of falling back to mock', () => {
    vi.stubEnv('TARO_APP_RESEARCH_SOURCE', 'api');
    vi.stubEnv('TARO_APP_MINIAPP_API_BASE_URL', '');

    expect(() => createResearchReasoningTreePort()).toThrow('absolute HTTP(S) URL');
  });

  it('rejects an unsupported source', () => {
    vi.stubEnv('TARO_APP_RESEARCH_SOURCE', 'unexpected');

    expect(() => createResearchReasoningTreePort()).toThrow('Unsupported TARO_APP_RESEARCH_SOURCE');
  });
});
