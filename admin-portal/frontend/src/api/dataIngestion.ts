export interface PagedResponse<T> {
  items: T[];
  total: number;
  page: number;
  page_size: number;
}

export interface EventSemantic {
  who: string | null;
  what: string | null;
  when: string | null;
  where: string | null;
  why: string | null;
  how: string | null;
}

export interface EventItem {
  id: string;
  title: string;
  summary: string;
  semantic: EventSemantic;
  modality: 'FACT' | 'PLAN' | 'SPEC';
  occurred_at: string | null;
  announced_at: string | null;
  status: 'ACTIVE' | 'DEPRECATED' | 'ARCHIVED';
}

export interface EventQuery {
  page: number;
  title: string;
  modality?: string;
  status?: string;
  occurred_from?: string;
  occurred_to?: string;
  announced_from?: string;
  announced_to?: string;
}

const defaultPageSize = 50;

export async function loadEvents(
  token: string,
  query: EventQuery
): Promise<PagedResponse<EventItem>> {
  const params = new URLSearchParams();
  params.set('page', String(query.page));
  params.set('page_size', String(defaultPageSize));
  appendParam(params, 'title', query.title);
  appendParam(params, 'modality', query.modality);
  appendParam(params, 'status', query.status);
  appendParam(params, 'occurred_from', query.occurred_from);
  appendParam(params, 'occurred_to', query.occurred_to);
  appendParam(params, 'announced_from', query.announced_from);
  appendParam(params, 'announced_to', query.announced_to);
  const response = await fetch(`/api/admin/v1/events?${params.toString()}`, {
    headers: authHeaders(token)
  });
  return readJSON(response);
}

function appendParam(params: URLSearchParams, key: string, value?: string) {
  if (value?.trim()) {
    params.set(key, value.trim());
  }
}

function authHeaders(token: string): Record<string, string> {
  return { Authorization: `Bearer ${token}` };
}

async function readJSON<T>(response: Response): Promise<T> {
  if (!response.ok) {
    throw new Error(await responseErrorMessage(response));
  }
  const payload = (await response.json()) as { request_id?: unknown; result?: T };
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
    const payload = (await response.json()) as { error?: { message?: string } };
    if (payload.error?.message) {
      return payload.error.message;
    }
  } catch {}
  return `request failed with status ${response.status}`;
}
