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

export type SourceLevel = 'L1_OFFICIAL' | 'L2_WIRE' | 'L3_MEDIA' | 'L4_SOCIAL';

export interface EvidenceCategory {
  id: string;
  code: string;
  name: string;
  description: string;
}

export interface EvidenceItem {
  id: string;
  raw_evidence_id: string;
  title: string | null;
  summary: string;
  semantic: EvidenceSemantic;
  categories: EvidenceCategory[];
  source_id: string;
  source_name: string;
  source_level: SourceLevel;
  source_url: string;
  is_original: boolean;
  quoted_source_name: string | null;
  keywords: string[];
  is_split: boolean;
  published_at: string | null;
  collected_at: string;
}

export interface EvidenceSemantic {
  who: string | null;
  what: string;
  when: string | null;
  where: string | null;
  why: string | null;
  how: string | null;
}

export interface EvidenceQuery {
  page: number;
  title?: string;
  summary?: string;
  category_id?: string;
  source_id?: string;
  source_name?: string;
  source_level?: string;
  is_split?: string;
  published_from?: string;
  published_to?: string;
  collected_from?: string;
  collected_to?: string;
}

export type CollectionDocument = { available: true; url: string } | { available: false; url: null };

export interface SourceItem {
  id: string;
  code: string;
  name: string;
  ownership_type: 'fixed' | 'dynamic';
  channel_type: 'web_search' | 'api' | 'rss';
  enabled: boolean;
  priority: number;
  default_source_level: SourceLevel;
  updated_at: string;
}

export interface SourceQuery {
  page: number;
  query?: string;
  ownership_type?: string;
  channel_type?: string;
  enabled?: string;
  priority?: string;
  default_source_level?: string;
  updated_from?: string;
  updated_to?: string;
}

const defaultPageSize = 50;

const utcTimestamp = z
  .string()
  .min(1)
  .refine((value) => value.endsWith('Z') && !Number.isNaN(Date.parse(value)));
const domainUUID = '[0-9a-f]{8}-[0-9a-f]{4}-[1-5][0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}';
const sourceLevelSchema = z.enum(['L1_OFFICIAL', 'L2_WIRE', 'L3_MEDIA', 'L4_SOCIAL']);
const evidenceCategorySchema: z.ZodType<EvidenceCategory> = z
  .object({
    id: z.string().regex(new RegExp(`^EVC${domainUUID}$`)),
    code: z.string().regex(/^[A-Z][A-Z0-9_]*$/),
    name: z.string().min(1),
    description: z.string().min(1)
  })
  .strict();
const eventItemSchema: z.ZodType<EventItem> = z
  .object({
    id: z.string().regex(new RegExp(`^EVT${domainUUID}$`)),
    title: z.string().min(1),
    summary: z.string().min(1),
    semantic: z
      .object({
        who: z.string().nullable(),
        what: z.string().nullable(),
        when: z.string().nullable(),
        where: z.string().nullable(),
        why: z.string().nullable(),
        how: z.string().nullable()
      })
      .strict(),
    modality: z.enum(['FACT', 'PLAN', 'SPEC']),
    occurred_at: utcTimestamp.nullable(),
    announced_at: utcTimestamp.nullable(),
    status: z.enum(['ACTIVE', 'DEPRECATED', 'ARCHIVED'])
  })
  .strict();
const evidenceItemSchema: z.ZodType<EvidenceItem> = z
  .object({
    id: z.string().regex(new RegExp(`^EVD${domainUUID}$`)),
    raw_evidence_id: z.string().regex(new RegExp(`^RAW${domainUUID}$`)),
    title: z.string().nullable(),
    summary: z.string().min(1),
    semantic: z
      .object({
        who: z.string().min(1).nullable(),
        what: z.string().min(1),
        when: z.string().min(1).nullable(),
        where: z.string().min(1).nullable(),
        why: z.string().min(1).nullable(),
        how: z.string().min(1).nullable()
      })
      .strict(),
    categories: z.array(evidenceCategorySchema),
    source_id: z.string().min(1).max(32),
    source_name: z.string().min(1),
    source_level: sourceLevelSchema,
    source_url: z
      .string()
      .url()
      .refine((value) => /^https?:\/\//.test(value)),
    is_original: z.boolean(),
    quoted_source_name: z.string().min(1).nullable(),
    keywords: z.array(z.string()),
    is_split: z.boolean(),
    published_at: utcTimestamp.nullable(),
    collected_at: utcTimestamp
  })
  .strict()
  .superRefine((item, context) => {
    if (item.is_original && item.quoted_source_name !== null) {
      context.addIssue({
        code: 'custom',
        message: 'original Evidence cannot declare quoted source'
      });
    }
    if (!item.is_original && item.quoted_source_name === null) {
      context.addIssue({ code: 'custom', message: 'reposted Evidence requires quoted source' });
    }
  });
const collectionDocumentSchema: z.ZodType<CollectionDocument> = z.discriminatedUnion('available', [
  z
    .object({
      available: z.literal(true),
      url: z
        .string()
        .url()
        .refine((value) => /^https?:\/\//.test(value))
    })
    .strict(),
  z.object({ available: z.literal(false), url: z.null() }).strict()
]);
const sourceItemSchema: z.ZodType<SourceItem> = z
  .object({
    id: z.string().regex(new RegExp(`^SRC${domainUUID}$`)),
    code: z.string().regex(/^[a-z0-9][a-z0-9_-]{0,63}$/),
    name: z.string().min(1),
    ownership_type: z.enum(['fixed', 'dynamic']),
    channel_type: z.enum(['web_search', 'api', 'rss']),
    enabled: z.boolean(),
    priority: z.number().int().min(1).max(5),
    default_source_level: sourceLevelSchema,
    updated_at: utcTimestamp
  })
  .strict();

function pagedResponseSchema<T>(item: z.ZodType<T>): z.ZodType<PagedResponse<T>> {
  return z
    .object({
      items: z.array(item),
      total: z.number().int().nonnegative(),
      page: z.number().int().positive(),
      page_size: z.number().int().min(1).max(100)
    })
    .strict()
    .refine((page) => page.items.length <= page.page_size && page.items.length <= page.total);
}

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
  return parseContract(pagedResponseSchema(eventItemSchema), await readJSON(response));
}

export async function loadEvidences(token: string, query: EvidenceQuery) {
  return loadPage(token, '/api/admin/v1/evidences', query, evidenceItemSchema);
}

export async function loadCollectionDocument(
  token: string,
  rawEvidenceID: string
): Promise<CollectionDocument> {
  const response = await fetch(
    `/api/admin/v1/raw-evidences/${encodeURIComponent(rawEvidenceID)}/collection-document`,
    { headers: authHeaders(token) }
  );
  return parseContract(collectionDocumentSchema, await readJSON(response));
}

export async function loadSources(token: string, query: SourceQuery) {
  return loadPage(token, '/api/admin/v1/sources', query, sourceItemSchema);
}

export async function loadEvidenceCategories(token: string): Promise<EvidenceCategory[]> {
  const response = await fetch('/api/admin/v1/evidence-categories', {
    headers: authHeaders(token)
  });
  const result = parseContract(
    z.object({ categories: z.array(evidenceCategorySchema) }).strict(),
    await readJSON(response)
  );
  return result.categories;
}

async function loadPage<T>(
  token: string,
  path: string,
  query: { page: number },
  itemSchema: z.ZodType<T>
) {
  const params = new URLSearchParams();
  params.set('page', String(query.page));
  params.set('page_size', String(defaultPageSize));
  Object.entries(query).forEach(([key, value]) => {
    if (key !== 'page' && typeof value === 'string') appendParam(params, key, value);
  });
  const response = await fetch(`${path}?${params.toString()}`, { headers: authHeaders(token) });
  const result = parseContract(pagedResponseSchema(itemSchema), await readJSON(response));
  if (result.page_size !== defaultPageSize) {
    throw new Error('Admin API returned an invalid response');
  }
  return result;
}

function appendParam(params: URLSearchParams, key: string, value?: string) {
  if (value?.trim()) {
    params.set(key, value.trim());
  }
}

function authHeaders(token: string): Record<string, string> {
  return { Authorization: `Bearer ${token}` };
}

async function readJSON(response: Response): Promise<unknown> {
  if (!response.ok) {
    throw new Error(await responseErrorMessage(response));
  }
  let payload: unknown;
  try {
    payload = await response.json();
  } catch {
    throw new Error('Admin API returned an invalid response');
  }
  return parseContract(
    z.object({ request_id: z.string().min(1), result: z.unknown() }).strict(),
    payload
  ).result;
}

async function responseErrorMessage(response: Response): Promise<string> {
  try {
    const parsed = z
      .object({ error: z.object({ message: z.string().min(1) }).passthrough() })
      .passthrough()
      .safeParse(await response.json());
    if (parsed.success) {
      return parsed.data.error.message;
    }
  } catch {}
  return `request failed with status ${response.status}`;
}

function parseContract<T>(schema: z.ZodType<T>, value: unknown): T {
  const parsed = schema.safeParse(value);
  if (!parsed.success) throw new Error('Admin API returned an invalid response');
  return parsed.data;
}
import { z } from 'zod';
