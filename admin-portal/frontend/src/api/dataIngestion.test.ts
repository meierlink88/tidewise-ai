import { describe, expect, it, vi } from 'vitest';
import {
  loadCollectionDocument,
  loadEvidenceCategories,
  loadEvidences,
  loadEvents,
  loadSources
} from './dataIngestion';

const evidenceSemantic = {
  actors: ['商务部'],
  action: '发布公告',
  objects: ['公告'],
  stage: 'ANNOUNCED',
  modality: 'FACT',
  time: { raw: null, start_at: null, end_at: null, precision: 'UNKNOWN' },
  jurisdictions: ['中国'],
  reason: null,
  method: null,
  metrics: [],
  attribution: { reported_by: null, claimed_by: '商务部' }
};

describe('data ingestion api client', () => {
  it('validates the complete seven-field Event semantic projection', async () => {
    vi.stubGlobal(
      'fetch',
      vi.fn().mockResolvedValue({
        ok: true,
        json: async () => ({
          request_id: 'admin-event-test',
          result: {
            items: [
              {
                id: 'EVT00000000-0000-5000-8000-000000000001',
                title: '美联储维持利率不变',
                summary: '美联储宣布维持联邦基金利率目标区间不变。',
                semantic: {
                  actors: ['Federal Reserve'],
                  action: 'holds target rate',
                  objects: ['federal funds rate'],
                  stage: 'ANNOUNCED',
                  jurisdictions: ['United States'],
                  effective_at: null,
                  time_precision: 'DAY'
                },
                modality: 'FACT',
                occurred_at: '2026-07-09T08:00:00Z',
                announced_at: null,
                status: 'ACTIVE'
              }
            ],
            total: 1,
            page: 1,
            page_size: 50
          }
        })
      })
    );

    const page = await loadEvents('secret-token', { page: 1, title: '' });

    expect(page.items[0]?.semantic).toEqual({
      actors: ['Federal Reserve'],
      action: 'holds target rate',
      objects: ['federal funds rate'],
      stage: 'ANNOUNCED',
      jurisdictions: ['United States'],
      effective_at: null,
      time_precision: 'DAY'
    });
  });

  it.each([
    [
      'missing field',
      {
        actors: ['Actor'],
        action: 'acts',
        objects: ['Object'],
        stage: 'OCCURRED',
        jurisdictions: [],
        effective_at: null
      }
    ],
    [
      'extra field',
      {
        actors: ['Actor'],
        action: 'acts',
        objects: ['Object'],
        stage: 'OCCURRED',
        jurisdictions: [],
        effective_at: null,
        time_precision: 'DAY',
        who: null
      }
    ],
    [
      'invalid stage',
      {
        actors: ['Actor'],
        action: 'acts',
        objects: ['Object'],
        stage: 'INVALID',
        jurisdictions: [],
        effective_at: null,
        time_precision: 'DAY'
      }
    ],
    [
      'blank actor',
      {
        actors: [' '],
        action: 'acts',
        objects: ['Object'],
        stage: 'OCCURRED',
        jurisdictions: [],
        effective_at: null,
        time_precision: 'DAY'
      }
    ],
    [
      'blank action',
      {
        actors: ['Actor'],
        action: ' ',
        objects: ['Object'],
        stage: 'OCCURRED',
        jurisdictions: [],
        effective_at: null,
        time_precision: 'DAY'
      }
    ]
  ])('rejects Event semantic contract drift: %s', async (_name, semantic) => {
    vi.stubGlobal(
      'fetch',
      vi.fn().mockResolvedValue({
        ok: true,
        json: async () => ({
          request_id: 'admin-event-test',
          result: {
            items: [
              {
                id: 'EVT00000000-0000-5000-8000-000000000001',
                title: 'Event',
                summary: 'Summary',
                semantic,
                modality: 'FACT',
                occurred_at: null,
                announced_at: null,
                status: 'ACTIVE'
              }
            ],
            total: 1,
            page: 1,
            page_size: 50
          }
        })
      })
    );

    await expect(loadEvents('secret-token', { page: 1, title: '' })).rejects.toThrow(
      'Admin API returned an invalid response'
    );
  });

  it('loads current events with the frozen filters', async () => {
    const fetchMock = vi.fn().mockResolvedValue({
      ok: true,
      json: async () => ({
        request_id: 'admin-event-test',
        result: { items: [], total: 0, page: 1, page_size: 50 }
      })
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

  it('maps Evidence and Source filters without exposing retired fields', async () => {
    const fetchMock = vi.fn().mockResolvedValue({
      ok: true,
      json: async () => ({
        request_id: 'request',
        result: { items: [], total: 0, page: 1, page_size: 50 }
      })
    });
    vi.stubGlobal('fetch', fetchMock);
    await loadEvidences('token', {
      page: 1,
      title: '标题',
      category_id: 'category',
      source_id: 'SRC_example_00000000000000000000',
      is_split: 'true'
    });
    await loadSources('token', { page: 2, query: 'official', enabled: 'true', priority: '1' });
    expect(fetchMock).toHaveBeenNthCalledWith(
      1,
      '/api/admin/v1/evidences?page=1&page_size=50&title=%E6%A0%87%E9%A2%98&category_id=category&source_id=SRC_example_00000000000000000000&is_split=true',
      { headers: { Authorization: 'Bearer token' } }
    );
    expect(fetchMock).toHaveBeenNthCalledWith(
      2,
      '/api/admin/v1/sources?page=2&page_size=50&query=official&enabled=true&priority=1',
      { headers: { Authorization: 'Bearer token' } }
    );
    expect(fetchMock.mock.calls.flat().join(' ')).not.toMatch(/adapter|endpoint/);
  });

  it('loads the Evidence Category catalog', async () => {
    const fetchMock = vi.fn().mockResolvedValue({
      ok: true,
      json: async () => ({ request_id: 'request', result: { categories: [] } })
    });
    vi.stubGlobal('fetch', fetchMock);
    await expect(loadEvidenceCategories('token')).resolves.toEqual([]);
    expect(fetchMock).toHaveBeenCalledWith('/api/admin/v1/evidence-categories', {
      headers: { Authorization: 'Bearer token' }
    });
  });

  it('loads a validated collection document link on demand', async () => {
    const fetchMock = vi.fn().mockResolvedValue({
      ok: true,
      json: async () => ({
        request_id: 'collection-document',
        result: {
          available: true,
          url: 'https://tideai.tripwise.cn/raw-evidence/documents/2026/08/17/11f0864fc4078b47a4cc758149a2b0b7923654d2c7c8a694ad5b2d5ced4fc998.md'
        }
      })
    });
    vi.stubGlobal('fetch', fetchMock);

    await expect(
      loadCollectionDocument('token', 'RAW00000000-0000-5000-8000-000000000001')
    ).resolves.toMatchObject({ available: true });
    expect(fetchMock).toHaveBeenCalledWith(
      '/api/admin/v1/raw-evidences/RAW00000000-0000-5000-8000-000000000001/collection-document',
      { headers: { Authorization: 'Bearer token' } }
    );
  });

  it('validates the complete Evidence detail projection', async () => {
    vi.stubGlobal(
      'fetch',
      vi.fn().mockResolvedValue({
        ok: true,
        json: async () => ({
          request_id: 'evidence-detail',
          result: {
            items: [
              {
                id: 'EVD00000000-0000-5000-8000-000000000001',
                raw_evidence_id: 'RAW00000000-0000-5000-8000-000000000001',
                title: '材料标题',
                summary: '原子证据摘要',
                semantic: evidenceSemantic,
                categories: [],
                source_id: 'SRC_example_00000000000000000000',
                source_name: '商务部',
                source_level: 'L1_OFFICIAL',
                source_url: 'https://example.com/report',
                is_original: true,
                quoted_source_name: null,
                keywords: ['公告'],
                is_split: false,
                published_at: null,
                collected_at: '2026-08-19T02:00:00Z'
              }
            ],
            total: 1,
            page: 1,
            page_size: 50
          }
        })
      })
    );

    const page = await loadEvidences('token', { page: 1 });
    expect(page.items[0]).toMatchObject({
      semantic: { action: '发布公告' },
      source_id: 'SRC_example_00000000000000000000',
      source_url: 'https://example.com/report',
      is_original: true,
      quoted_source_name: null,
      keywords: ['公告']
    });
  });

  it('rejects inconsistent Evidence source attribution', async () => {
    vi.stubGlobal(
      'fetch',
      vi.fn().mockResolvedValue({
        ok: true,
        json: async () => ({
          request_id: 'evidence-detail',
          result: {
            items: [
              {
                id: 'EVD00000000-0000-5000-8000-000000000001',
                raw_evidence_id: 'RAW00000000-0000-5000-8000-000000000001',
                title: null,
                summary: '摘要',
                semantic: evidenceSemantic,
                categories: [],
                source_id: 'SRC_example_00000000000000000000',
                source_name: '官方信源',
                source_level: 'L1_OFFICIAL',
                source_url: 'https://example.com/report',
                is_original: false,
                quoted_source_name: null,
                keywords: ['公告'],
                is_split: false,
                published_at: null,
                collected_at: '2026-08-19T02:00:00Z'
              }
            ],
            total: 1,
            page: 1,
            page_size: 50
          }
        })
      })
    );

    await expect(loadEvidences('token', { page: 1 })).rejects.toThrow(
      'Admin API returned an invalid response'
    );
  });

  it('rejects untrusted list items with contract drift', async () => {
    vi.stubGlobal(
      'fetch',
      vi.fn().mockResolvedValue({
        ok: true,
        json: async () => ({
          request_id: 'request',
          result: {
            items: [
              {
                id: 'SRC00000000-0000-5000-8000-000000000001',
                code: 'official',
                name: 'Official',
                ownership_type: 'fixed',
                channel_type: 'api',
                enabled: true,
                priority: 1,
                default_source_level: 'L1_OFFICIAL',
                updated_at: '2026-08-19T02:00:00Z',
                endpoint: 'https://must-not-cross.example.test'
              }
            ],
            total: 1,
            page: 1,
            page_size: 50
          }
        })
      })
    );
    await expect(loadSources('token', { page: 1 })).rejects.toThrow(
      'Admin API returned an invalid response'
    );
  });
});
