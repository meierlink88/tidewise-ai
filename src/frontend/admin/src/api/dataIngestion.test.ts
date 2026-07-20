import { describe, expect, it, vi } from 'vitest';
import {
  adminAPIURL,
  loadEvents,
  loadRawDocuments,
  loadSourceCatalogs
} from './dataIngestion';

describe('data ingestion api client', () => {
  it('uses the runtime Admin API base URL without rebuilding the frontend', () => {
    window.__TIDEWISE_RUNTIME_CONFIG__ = { adminApiBaseUrl: 'http://uat.example.test:9013/' };
    expect(adminAPIURL('/admin/events')).toBe('http://uat.example.test:9013/admin/events');
    window.__TIDEWISE_RUNTIME_CONFIG__ = undefined;
  });

  it('loads raw documents with title search and default page size', async () => {
    const fetchMock = vi.fn().mockResolvedValue({
      ok: true,
      json: async () => ({ items: [], total: 0, page: 1, page_size: 50 })
    });
    vi.stubGlobal('fetch', fetchMock);

    await loadRawDocuments('secret-token', { page: 2, title: '央行' });

    expect(fetchMock).toHaveBeenCalledWith('/admin/raw-documents?page=2&page_size=50&title=%E5%A4%AE%E8%A1%8C', {
      headers: { Authorization: 'Bearer secret-token' }
    });
  });

  it('loads events with title and approved filters', async () => {
    const fetchMock = vi.fn().mockResolvedValue({
      ok: true,
      json: async () => ({ items: [], total: 0, page: 1, page_size: 50 })
    });
    vi.stubGlobal('fetch', fetchMock);

    await loadEvents('secret-token', {
      page: 1,
      title: '美联储',
      event_status: 'confirmed',
      fact_status: 'verified',
      event_time_from: '2026-07-09T00:00:00Z',
      event_time_to: '2026-07-10T00:00:00Z',
      first_seen_from: '2026-07-09T00:00:00Z',
      first_seen_to: '2026-07-10T00:00:00Z'
    });

    expect(fetchMock).toHaveBeenCalledWith('/admin/events?page=1&page_size=50&title=%E7%BE%8E%E8%81%94%E5%82%A8&event_status=confirmed&fact_status=verified&event_time_from=2026-07-09T00%3A00%3A00Z&event_time_to=2026-07-10T00%3A00%3A00Z&first_seen_from=2026-07-09T00%3A00%3A00Z&first_seen_to=2026-07-10T00%3A00%3A00Z', {
      headers: { Authorization: 'Bearer secret-token' }
    });
  });

  it('loads source catalogs by status without pagination', async () => {
    const fetchMock = vi.fn().mockResolvedValue({
      ok: true,
      json: async () => ({ items: [] })
    });
    vi.stubGlobal('fetch', fetchMock);

    await loadSourceCatalogs('secret-token', { status: 'inactive' });

    expect(fetchMock).toHaveBeenCalledWith('/admin/source-catalogs?status=inactive', {
      headers: { Authorization: 'Bearer secret-token' }
    });
  });
});
