export interface PagedResponse<T> {
  items: T[];
  total: number;
  page: number;
  page_size: number;
}

export interface RawDocumentItem {
  id: string;
  source_id?: string;
  ingest_channel?: string;
  source_type?: string;
  source_name: string;
  source_url?: string;
  source_external_id?: string;
  title: string;
  content_text: string;
  raw_object_uri?: string;
  raw_mime_type?: string;
  language?: string;
  published_at?: string;
  collected_at: string;
  ingest_status: string;
}

export interface EventItem {
  id: string;
  title: string;
  summary: string;
  event_time?: string;
  first_seen_at: string;
  knowable_at?: string;
  event_status: string;
  fact_status: string;
  dedupe_key?: string;
  primary_source_id?: string;
}

export interface SourceCatalogItem {
  id: string;
  ingest_channel: string;
  provider_key: string;
  connector_key?: string;
  source_type: string;
  source_name: string;
  source_url?: string;
  source_level?: string;
  topic_hint?: string;
  usage_policy?: string;
  status: string;
}

export interface RawDocumentQuery {
  page: number;
  title: string;
}

export interface EventQuery {
  page: number;
  title: string;
  event_status?: string;
  fact_status?: string;
  event_time_from?: string;
  event_time_to?: string;
  first_seen_from?: string;
  first_seen_to?: string;
}

export interface SourceCatalogQuery {
  status?: string;
}

const defaultPageSize = 50;

export async function loadRawDocuments(token: string, query: RawDocumentQuery): Promise<PagedResponse<RawDocumentItem>> {
  const params = new URLSearchParams();
  params.set('page', String(query.page));
  params.set('page_size', String(defaultPageSize));
  if (query.title.trim()) {
    params.set('title', query.title.trim());
  }
  const response = await fetch(`/admin/raw-documents?${params.toString()}`, {
    headers: authHeaders(token)
  });
  return readJSON(response);
}

export async function loadEvents(token: string, query: EventQuery): Promise<PagedResponse<EventItem>> {
  const params = new URLSearchParams();
  params.set('page', String(query.page));
  params.set('page_size', String(defaultPageSize));
  appendParam(params, 'title', query.title);
  appendParam(params, 'event_status', query.event_status);
  appendParam(params, 'fact_status', query.fact_status);
  appendParam(params, 'event_time_from', query.event_time_from);
  appendParam(params, 'event_time_to', query.event_time_to);
  appendParam(params, 'first_seen_from', query.first_seen_from);
  appendParam(params, 'first_seen_to', query.first_seen_to);
  const response = await fetch(`/admin/events?${params.toString()}`, {
    headers: authHeaders(token)
  });
  return readJSON(response);
}

export async function loadSourceCatalogs(token: string, query: SourceCatalogQuery = {}): Promise<{ items: SourceCatalogItem[] }> {
  const params = new URLSearchParams();
  appendParam(params, 'status', query.status);
  const suffix = params.toString();
  const response = await fetch(`/admin/source-catalogs${suffix ? `?${suffix}` : ''}`, {
    headers: authHeaders(token)
  });
  return readJSON(response);
}

function appendParam(params: URLSearchParams, key: string, value?: string) {
  if (value && value.trim()) {
    params.set(key, value.trim());
  }
}

function authHeaders(token: string): Record<string, string> {
  return {
    Authorization: `Bearer ${token}`
  };
}

async function readJSON<T>(response: Response): Promise<T> {
  if (!response.ok) {
    throw new Error(await responseErrorMessage(response));
  }
  return response.json() as Promise<T>;
}

async function responseErrorMessage(response: Response): Promise<string> {
  try {
    const payload = await response.json() as { error?: string };
    if (payload.error) {
      return payload.error;
    }
  } catch {
  }
  return `request failed with status ${response.status}`;
}
