export interface PagedResponse<T> {
  items: T[];
  total: number;
  page: number;
  page_size: number;
}

export interface RawDocumentItem {
  id: string;
  contract_version?: number;
  artifact_id?: string;
  source_ref?: string;
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

const defaultPageSize = 50;

declare global {
  interface Window {
    __TIDEWISE_RUNTIME_CONFIG__?: {
      adminApiBaseUrl?: string;
    };
  }
}

export async function loadRawDocuments(
  token: string,
  query: RawDocumentQuery
): Promise<PagedResponse<RawDocumentItem>> {
  const params = new URLSearchParams();
  params.set('page', String(query.page));
  params.set('page_size', String(defaultPageSize));
  if (query.title.trim()) {
    params.set('title', query.title.trim());
  }
  const response = await fetch(adminAPIURL(`/api/admin/v1/raw-documents?${params.toString()}`), {
    headers: authHeaders(token)
  });
  return readJSON(response);
}

export async function loadEvents(
  token: string,
  query: EventQuery
): Promise<PagedResponse<EventItem>> {
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
  const response = await fetch(adminAPIURL(`/api/admin/v1/events?${params.toString()}`), {
    headers: authHeaders(token)
  });
  return readJSON(response);
}

export function adminAPIURL(path: string): string {
  const baseURL =
    window.__TIDEWISE_RUNTIME_CONFIG__?.adminApiBaseUrl?.trim().replace(/\/$/, '') ?? '';
  return `${baseURL}${path}`;
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
  const payload = (await response.json()) as {
    request_id?: unknown;
    result?: T;
  };
  if (
    typeof payload.request_id !== 'string' ||
    payload.request_id.length === 0 ||
    !Object.prototype.hasOwnProperty.call(payload, 'result')
  ) {
    throw new Error('Admin API returned an invalid response');
  }
  return payload.result as T;
}

async function responseErrorMessage(response: Response): Promise<string> {
  try {
    const payload = (await response.json()) as {
      error?: { message?: string };
    };
    if (payload.error?.message) {
      return payload.error.message;
    }
  } catch {}
  return `request failed with status ${response.status}`;
}
