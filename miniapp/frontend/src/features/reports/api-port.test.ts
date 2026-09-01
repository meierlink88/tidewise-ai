import { beforeEach, describe, expect, it, vi } from 'vitest';
import { APIReportPort } from './api-port';

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

  it('passes direct local Evidence scope keys and maps not-found errors', async () => {
    request.mockResolvedValue({ statusCode: 404, data: {} });
    const port = new APIReportPort('https://miniapp.example.com');
    const reportId = 'RPT11111111-1111-4111-8111-111111111111';

    await expect(
      port.getEvidences(reportId, { type: 'industry_chain_node', key: 'chn-21-n01' })
    ).rejects.toMatchObject({ kind: 'evidenceScopeUnavailable' });
    expect(request.mock.calls[0][0].url).toContain(
      'scope_type=industry_chain_node&scope_key=chn-21-n01'
    );
    expect(request.mock.calls[0][0].url).not.toContain('%2F');
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

  it('maps transport failure without retrying against mock data', async () => {
    request.mockRejectedValue(new Error('network down'));
    const port = new APIReportPort('https://miniapp.example.com');

    await expect(port.getHome()).rejects.toMatchObject({ kind: 'serviceUnavailable' });
    expect(request).toHaveBeenCalledOnce();
  });
});
