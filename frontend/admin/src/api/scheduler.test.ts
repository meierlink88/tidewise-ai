import { describe, expect, it, vi } from 'vitest';
import { loadSchedulerConfig, saveSchedulerConfig } from './scheduler';

describe('scheduler api client', () => {
  it('adds admin token header when loading config', async () => {
    const fetchMock = vi.fn().mockResolvedValue({
      ok: true,
      json: async () => ({ enabled: false, mode: 'interval' })
    });
    vi.stubGlobal('fetch', fetchMock);

    await loadSchedulerConfig('secret-token');

    expect(fetchMock).toHaveBeenCalledWith('/admin/scheduler/config', {
      headers: { Authorization: 'Bearer secret-token' }
    });
  });

  it('sends scheduler config as json', async () => {
    const fetchMock = vi.fn().mockResolvedValue({
      ok: true,
      json: async () => ({ enabled: true, mode: 'interval' })
    });
    vi.stubGlobal('fetch', fetchMock);

    await saveSchedulerConfig('secret-token', {
      enabled: true,
      mode: 'interval',
      interval_minutes: 60,
      fixed_times: [],
      concurrency: 2,
      batch_size: 20,
      timeout_seconds: 180,
      source_filter: {
        provider_key: 'llm_web_research',
        ingest_channel: 'ai_web_research',
        source_type: 'news'
      },
      timezone: 'Asia/Shanghai'
    });

    expect(fetchMock).toHaveBeenCalledWith('/admin/scheduler/config', expect.objectContaining({
      method: 'PUT',
      headers: {
        Authorization: 'Bearer secret-token',
        'Content-Type': 'application/json'
      }
    }));
  });

  it('includes api error message when request fails', async () => {
    const fetchMock = vi.fn().mockResolvedValue({
      ok: false,
      status: 401,
      json: async () => ({ error: 'unauthorized' })
    });
    vi.stubGlobal('fetch', fetchMock);

    await expect(loadSchedulerConfig('wrong-token')).rejects.toThrow('unauthorized');
  });
});
