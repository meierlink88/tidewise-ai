import { beforeEach, describe, expect, it, vi } from 'vitest';
import { APIReportPort } from './api-port';
import normalized from '../../mocks/reports/normalized.json';

const { request } = vi.hoisted(() => ({ request: vi.fn() }));

vi.mock('@tarojs/taro', () => ({ default: { request } }));

describe('APIReportPort', () => {
  beforeEach(() => {
    request.mockReset();
  });

  it('requests plural Report home and accepts today empty without a mock fallback', async () => {
    request.mockResolvedValue({
      statusCode: 200,
      data: {
        request_id: 'req-1',
        result: {
          selection: { mode: 'today', date: '2026-09-01', timezone: 'Asia/Shanghai' },
          reports: []
        }
      }
    });
    const port = new APIReportPort('https://miniapp.example.com');

    await expect(port.getHome()).resolves.toMatchObject({ reports: [] });
    expect(request).toHaveBeenCalledOnce();
    expect(request.mock.calls[0][0].url).toBe(
      'https://miniapp.example.com/api/miniapp/v1/reports/home'
    );
  });

  it('passes the opaque Evidence scope token and maps not-found errors', async () => {
    request.mockResolvedValue({ statusCode: 404, data: {} });
    const port = new APIReportPort('https://miniapp.example.com');
    const reportId = 'RPT11111111-1111-4111-8111-111111111111';

    await expect(
      port.getEvidences(reportId, 'RPE11111111-1111-4111-8111-111111111111')
    ).rejects.toMatchObject({ kind: 'evidenceScopeUnavailable' });
    expect(request.mock.calls[0][0].url).toContain(
      'scope_token=RPE11111111-1111-4111-8111-111111111111'
    );
    expect(request.mock.calls[0][0].url).not.toContain('%2F');
  });

  it('requests report-bound cursor pages for industry cards', async () => {
    request.mockResolvedValue({
      statusCode: 200,
      data: { request_id: 'req-page', result: { items: [], next_cursor: null } }
    });
    const port = new APIReportPort('https://miniapp.example.com');
    const reportId = 'RPT11111111-1111-4111-8111-111111111111';

    await expect(port.getAnalyses(reportId, 'concept_analyses', 'opaque cursor')).resolves.toEqual({
      items: [],
      next_cursor: null
    });
    expect(request.mock.calls[0][0].url).toContain('limit=20&cursor=opaque%20cursor');
  });

  it('fails closed on surplus envelopes and malformed DTOs', async () => {
    const port = new APIReportPort('https://miniapp.example.com');
    request.mockResolvedValueOnce({
      statusCode: 200,
      data: {
        request_id: 'req-1',
        result: {
          selection: { mode: 'today', date: '2026-09-01', timezone: 'Asia/Shanghai' },
          reports: []
        },
        event_count: 1
      }
    });
    await expect(port.getHome()).rejects.toMatchObject({ kind: 'invalidResponse' });

    request.mockResolvedValueOnce({
      statusCode: 200,
      data: { request_id: 'req-2', result: { report: null } }
    });
    await expect(port.getHome()).rejects.toMatchObject({ kind: 'invalidResponse' });
    expect(request).toHaveBeenCalledTimes(2);
  });

  it('reads normalized card, directory and chain projections within the same v1 report scope', async () => {
    const port = new APIReportPort('https://miniapp.example.com');
    const id = 'RPT11111111-1111-4111-8111-111111111111';
    const responses = [
      { items: normalized.groups[0].items, next_cursor: null },
      normalized.details['geopolitical_stories/g1'],
      normalized.chains['geopolitical_stories/g1/g1-2-chain']
    ];
    responses.forEach((result) =>
      request.mockResolvedValueOnce({ statusCode: 200, data: { request_id: 'test', result } })
    );
    await expect(port.getAnalyses(id, 'geopolitical_stories', 'opaque cursor')).resolves.toEqual(
      responses[0]
    );
    await expect(port.getAnalysis(id, 'geopolitical_stories', 'g1')).resolves.toEqual(responses[1]);
    await expect(
      port.getAnalysisChain(id, 'geopolitical_stories', 'g1', 'g1-2-chain')
    ).resolves.toEqual(responses[2]);
    expect(request.mock.calls[0][0].url).toContain(
      '/api/miniapp/v1/reports/' +
        id +
        '/analyses/geopolitical_stories?limit=20&cursor=opaque%20cursor'
    );
    expect(request.mock.calls[2][0].url).toContain(
      '/analyses/geopolitical_stories/g1/industry-chains/g1-2-chain'
    );
  });

  it('maps transport failure without retrying against mock data', async () => {
    request.mockRejectedValue(new Error('network down'));
    const port = new APIReportPort('https://miniapp.example.com');

    await expect(port.getHome()).rejects.toMatchObject({ kind: 'serviceUnavailable' });
    expect(request).toHaveBeenCalledOnce();
  });
});
