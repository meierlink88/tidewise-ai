import { describe, expect, it, vi } from 'vitest';
import { loadEvents } from './dataIngestion';

describe('data ingestion api client', () => {
  it('loads current events with the frozen filters', async () => {
    const fetchMock = vi.fn().mockResolvedValue({
      ok: true,
      json: async () => ({ request_id: 'admin-event-test', result: { items: [], total: 0, page: 1, page_size: 50 } })
    });
    vi.stubGlobal('fetch', fetchMock);

    await loadEvents('secret-token', {
      page: 1,
      title: '美联储',
      modality: 'FACT',
      status: 'ACTIVE',
      occurred_from: '2026-07-09T00:00:00Z',
      occurred_to: '2026-07-10T00:00:00Z',
      announced_from: '2026-07-09T00:00:00Z',
      announced_to: '2026-07-10T00:00:00Z'
    });

    expect(fetchMock).toHaveBeenCalledWith(
      '/api/admin/v1/events?page=1&page_size=50&title=%E7%BE%8E%E8%81%94%E5%82%A8&modality=FACT&status=ACTIVE&occurred_from=2026-07-09T00%3A00%3A00Z&occurred_to=2026-07-10T00%3A00%3A00Z&announced_from=2026-07-09T00%3A00%3A00Z&announced_to=2026-07-10T00%3A00%3A00Z',
      { headers: { Authorization: 'Bearer secret-token' } }
    );
  });
});
