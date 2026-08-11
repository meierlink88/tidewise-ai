import { afterEach, describe, expect, it, vi } from 'vitest';
import {
  AdminAgentRunAPIError,
  loadAgentStatuses,
  loadCollectorMonitoring,
  loadMonitoringSummary,
  loadConnector,
  loadModelProvider,
  loadRuntimeHealth,
  saveAgentSchedule,
  updateConnector,
  updateModelProvider
} from './agentManagement';

describe('AgentRun management Admin API client', () => {
  afterEach(() => {
    vi.unstubAllGlobals();
  });

  it('saves schedule configuration without sending enabled state', async () => {
    const fetchMock = successFetch(scheduleResult());
    vi.stubGlobal('fetch', fetchMock);

    await saveAgentSchedule('browser-token', 'collector', {
      agent_version: 'collector.v1',
      schedule_type: 'daily',
      daily_times: ['08:30'],
      input: { prompt: '采集事实' }
    });

    const [url, init] = fetchMock.mock.calls[0] as [string, RequestInit];
    expect(url).toBe('/api/admin/v1/agent-schedules/collector');
    expect(init.method).toBe('PUT');
    expect(JSON.parse(String(init.body))).toEqual({
      agent_version: 'collector.v1',
      schedule_type: 'daily',
      daily_times: ['08:30'],
      input: { prompt: '采集事实' }
    });
    expect(JSON.parse(String(init.body))).not.toHaveProperty('enabled');
    expect(new Headers(init.headers).get('Authorization')).toBe('Bearer browser-token');
  });

	it('strictly loads the three runtime services through the Admin BFF', async () => {
    const fetchMock = successFetch(runtimeHealthResult());
    vi.stubGlobal('fetch', fetchMock);

    const result = await loadRuntimeHealth('browser-token');

    expect(fetchMock.mock.calls[0][0]).toBe('/api/admin/v1/runtime-health');
		expect(result.services.map((service) => service.key)).toEqual(['data', 'agentrun', 'qdrant']);
  });

  it('loads the safe Agent status projection through the Admin BFF', async () => {
    const fetchMock = successFetch({
      items: [
        {
          agent_key: 'event-semantic-enricher',
          display_name: 'Event Semantic Enricher',
          current_version: 'event-semantic-enricher.v1',
          is_working: true,
          current_execution_status: 'running',
          updated_at: '2026-07-29T08:30:00Z'
        }
      ]
    });
    vi.stubGlobal('fetch', fetchMock);

    const result = await loadAgentStatuses('browser-token');

    expect(fetchMock.mock.calls[0][0]).toBe('/api/admin/v1/agent-statuses');
    expect(result).toHaveLength(1);
    expect(result[0].current_execution_status).toBe('running');
  });

  it('loads monitoring only through versioned Admin BFF resources', async () => {
    const fetchMock = vi
      .fn()
      .mockResolvedValueOnce(successResponse(monitoringSummaryResult('6h')))
      .mockResolvedValueOnce(successResponse(monitoringPageResult('6h')));
    vi.stubGlobal('fetch', fetchMock);

    await loadMonitoringSummary('browser-token', '6h');
    await loadCollectorMonitoring('browser-token', '6h', 'failure');

    expect(fetchMock.mock.calls[0][0]).toBe('/api/admin/v1/monitoring/summary?window=6h');
    expect(fetchMock.mock.calls[1][0]).toBe(
      '/api/admin/v1/monitoring/collector-executions?window=6h&state=failure&page=1&page_size=20'
    );
    expect(String(fetchMock.mock.calls[0][0])).not.toContain('agentrun');
  });

  it('rejects malformed successful monitoring responses at the API boundary', async () => {
    vi.stubGlobal('fetch', successFetch({ ...monitoringSummaryResult('1h'), collector: null }));

    await expect(loadMonitoringSummary('browser-token', '1h')).rejects.toThrow(
      'Admin API returned invalid monitoring data'
    );
  });

  it('preserves model keys when omitted and explicitly clears connector keys', async () => {
    const fetchMock = vi
      .fn()
      .mockResolvedValueOnce(successResponse(modelResult()))
      .mockResolvedValueOnce(successResponse(connectorResult()));
    vi.stubGlobal('fetch', fetchMock);

    await updateModelProvider('browser-token', 'deepseek', {
      base_url: 'https://api.deepseek.com',
      model: 'deepseek-chat'
    });
    await updateConnector('browser-token', 'parallel_search', {
      base_url: 'https://search.example.com',
      api_key: ''
    });

    expect(JSON.parse(String(fetchMock.mock.calls[0][1]?.body))).not.toHaveProperty('api_key');
    expect(JSON.parse(String(fetchMock.mock.calls[1][1]?.body))).toMatchObject({ api_key: '' });
  });

  it('loads individual registered model and connector configurations through the BFF', async () => {
    const fetchMock = vi
      .fn()
      .mockResolvedValueOnce(successResponse(modelResult()))
      .mockResolvedValueOnce(successResponse(connectorResult()));
    vi.stubGlobal('fetch', fetchMock);

    await loadModelProvider('browser-token', 'deepseek');
    await loadConnector('browser-token', 'parallel_search');

    expect(fetchMock.mock.calls[0][0]).toBe('/api/admin/v1/model-providers/deepseek');
    expect(fetchMock.mock.calls[1][0]).toBe('/api/admin/v1/connectors/parallel_search');
  });

  it('returns a typed safe error from the Admin BFF envelope', async () => {
    vi.stubGlobal(
      'fetch',
      vi.fn().mockResolvedValue({
        ok: false,
        status: 503,
        json: async () => ({
          request_id: 'admin-error',
          error: {
            code: 'AGENTRUN_UNAVAILABLE',
            message: 'AgentRun is unavailable',
            details: {}
          }
        })
      })
    );

    await expect(loadRuntimeHealth('browser-token')).rejects.toEqual(
      expect.objectContaining<Partial<AdminAgentRunAPIError>>({
        status: 503,
        code: 'AGENTRUN_UNAVAILABLE',
        message: 'AgentRun is unavailable'
      })
    );
  });
});

function successFetch(result: unknown) {
  return vi.fn().mockResolvedValue(successResponse(result));
}

function runtimeHealthResult() {
  return {
    status: 'ready',
    checked_at: '2026-08-04T10:00:00Z',
    services: [
      {
        key: 'data',
        display_name: 'Data Service',
        status: 'ready',
        checked_at: '2026-08-04T10:00:00Z',
        latency_ms: null,
        reason_code: null
      },
      {
        key: 'agentrun',
        display_name: 'AgentRun',
        status: 'ready',
        checked_at: '2026-08-04T10:00:00Z',
        latency_ms: null,
        reason_code: null
      },
      {
        key: 'qdrant',
        display_name: 'Qdrant',
        status: 'ready',
        checked_at: '2026-08-04T10:00:00Z',
        latency_ms: null,
        reason_code: null
		}
    ]
  };
}

function successResponse(result: unknown) {
  return {
    ok: true,
    status: 200,
    json: async () => ({
      request_id: 'admin-agent-management',
      result
    })
  };
}

function scheduleResult() {
  return {
    schedule_id: 'schedule-1',
    agent_key: 'collector',
    agent_version: 'collector.v1',
    schedule_type: 'daily',
    daily_times: ['08:30'],
    input: { prompt: '采集事实' },
    enabled: false,
    created_at: '2026-07-24T00:00:00Z',
    updated_at: '2026-07-24T00:00:00Z'
  };
}

function monitoringSummaryResult(window: '1h' | '6h' | '12h' | '24h') {
  return {
    window,
    generated_at: '2026-08-03T08:30:00Z',
    collector: { success: 1, running: 0, failure: 0 },
    artifact_extraction: { success: 1, running: 0, failure: 0 },
    semantic: { success: 1, running: 0, failure: 0 },
    collector_raw_results: 2,
    collector_merged_results: 1,
    collector_accepted_artifacts: 1,
    artifact_published: 1,
    artifact_no_events: 0,
    artifact_formal_events: 1,
    semantic_submissions: 1,
    semantic_accepted_candidates: 1,
    semantic_rejected_candidates: 0
  };
}

function monitoringPageResult(window: '1h' | '6h' | '12h' | '24h') {
  return {
    items: [],
    window,
    generated_at: '2026-08-03T08:30:00Z',
    page: 1,
    page_size: 20,
    total_items: 0,
    total_pages: 0
  };
}

function modelResult() {
  return {
    provider_key: 'deepseek',
    base_url: 'https://api.deepseek.com',
    model: 'deepseek-chat',
    configured: true,
    key_configured: true
  };
}

function connectorResult() {
  return {
    connector_key: 'parallel_search',
    base_url: 'https://search.example.com',
    configured: false,
    key_configured: false
  };
}
