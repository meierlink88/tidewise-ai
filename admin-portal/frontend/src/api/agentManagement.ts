import { z } from 'zod';

export type ScheduleType = 'daily' | 'cron';

export interface AgentSchedule {
  schedule_id: string;
  agent_key: string;
  agent_version: string;
  schedule_type: ScheduleType;
  cron_expression?: string;
  daily_times?: string[];
  input: Record<string, unknown>;
  enabled: boolean;
  last_triggered_at?: string;
  next_run_at?: string;
  created_at: string;
  updated_at: string;
}

export interface AgentScheduleSaveInput {
  agent_version: 'collector.v1';
  schedule_type: ScheduleType;
  cron_expression?: string;
  daily_times?: string[];
  input: {
    prompt: string;
  };
}

export interface AgentStatus {
  agent_key: string;
  display_name: string;
  current_version: string;
  is_working: boolean;
  current_execution_status: string;
  updated_at: string;
}

export type RuntimeHealthStatus = 'ready' | 'degraded' | 'down' | 'unknown';
export type RuntimeHealthReasonCode =
  | 'timeout'
  | 'unreachable'
  | 'not_ready'
  | 'collection_unhealthy'
  | 'authentication_failed'
  | 'invalid_response';
export interface RuntimeHealthService {
  key: 'data' | 'agentrun' | 'qdrant' | 'neo4j';
  display_name: 'Data Service' | 'AgentRun' | 'Qdrant' | 'Neo4j';
  status: RuntimeHealthStatus;
  checked_at: string;
  latency_ms?: number | null;
  reason_code?: RuntimeHealthReasonCode | null;
}
export interface RuntimeHealth {
  status: 'ready' | 'degraded';
  checked_at: string;
  services: RuntimeHealthService[];
}

export type MonitoringWindow = '1h' | '6h' | '12h' | '24h';
export type MonitoringState = 'all' | 'success' | 'running' | 'failure';
export type MonitoringKind = 'collector' | 'artifact' | 'semantic';
export interface MonitoringCounts {
  success: number;
  running: number;
  failure: number;
}
export interface MonitoringSummary {
  window: MonitoringWindow;
  generated_at: string;
  collector: MonitoringCounts;
  artifact_extraction: MonitoringCounts;
  semantic: MonitoringCounts;
  collector_raw_results: number;
  collector_merged_results: number;
  collector_accepted_artifacts: number;
  artifact_published: number;
  artifact_no_events: number;
  artifact_formal_events: number;
  semantic_submissions: number;
  semantic_accepted_candidates: number;
  semantic_rejected_candidates: number;
}
export interface MonitoringPage<T> {
  items: T[];
  window: MonitoringWindow;
  generated_at: string;
  page: number;
  page_size: number;
  total_items: number;
  total_pages: number;
}
export interface CollectorMonitoringItem {
  execution_id: string;
  state: Exclude<MonitoringState, 'all'>;
  raw_status: string;
  trigger_source: string;
  started_at?: string;
  completed_at?: string;
  duration_ms?: number;
  raw_results: number;
  merged_results: number;
  accepted_artifacts: number;
  error_code?: string;
}
export interface ArtifactMonitoringItem {
  extraction_key: string;
  artifact_id: string;
  collector_execution_id: string;
  state: Exclude<MonitoringState, 'all'>;
  raw_status: string;
  updated_at: string;
  started_at?: string;
  completed_at?: string;
  duration_ms?: number;
  event_candidates: number;
  acknowledged_journals: number;
  total_journals: number;
  error_code?: string;
}
export interface SemanticMonitoringItem {
  work_item_id: string;
  event_id: string;
  trigger_source: string;
  state: Exclude<MonitoringState, 'all'>;
  raw_status: string;
  updated_at: string;
  started_at?: string;
  completed_at?: string;
  duration_ms?: number;
  attempt_count: number;
  max_attempts: number;
  accepted_candidates: number;
  rejected_candidates: number;
  error_code?: string;
}

export interface ModelProviderConfiguration {
  provider_key: string;
  base_url: string;
  model: string;
  configured: boolean;
  key_configured: boolean;
  masked_key?: string;
  updated_at?: string;
}

export interface ModelProviderPatch {
  base_url?: string;
  model?: string;
  api_key?: string;
}

export interface ConnectorConfiguration {
  connector_key: string;
  base_url: string;
  configured: boolean;
  key_configured: boolean;
  masked_key?: string;
  updated_at?: string;
}

export interface ConnectorPatch {
  base_url?: string;
  api_key?: string;
}

export class AdminAgentRunAPIError extends Error {
  readonly status: number;
  readonly code: string;

  constructor(status: number, code: string, message: string) {
    super(message);
    this.name = 'AdminAgentRunAPIError';
    this.status = status;
    this.code = code;
  }
}

export function loadAgentSchedule(token: string, agentKey: string): Promise<AgentSchedule> {
  return request<AgentSchedule>(
    token,
    `/api/admin/v1/agent-schedules/${encodeURIComponent(agentKey)}`
  );
}

export function saveAgentSchedule(
  token: string,
  agentKey: string,
  input: AgentScheduleSaveInput
): Promise<AgentSchedule> {
  return request<AgentSchedule>(
    token,
    `/api/admin/v1/agent-schedules/${encodeURIComponent(agentKey)}`,
    {
      method: 'PUT',
      body: JSON.stringify(input)
    }
  );
}

export function setAgentScheduleEnabled(
  token: string,
  agentKey: string,
  enabled: boolean
): Promise<AgentSchedule> {
  return request<AgentSchedule>(
    token,
    `/api/admin/v1/agent-schedules/${encodeURIComponent(agentKey)}`,
    {
      method: 'PATCH',
      body: JSON.stringify({ enabled })
    }
  );
}

export async function loadAgentStatuses(token: string): Promise<AgentStatus[]> {
  const result = await request<{ items: AgentStatus[] }>(token, '/api/admin/v1/agent-statuses');
  return result.items;
}

const monitoringWindowSchema = z.enum(['1h', '6h', '12h', '24h']);
const monitoringStateSchema = z.enum(['success', 'running', 'failure']);
const timestampSchema = z.iso.datetime({ offset: true });
const nonNegativeIntegerSchema = z.number().int().nonnegative();
const runtimeHealthStatusSchema = z.enum(['ready', 'degraded', 'down', 'unknown']);
const runtimeHealthReasonSchema = z.enum([
  'timeout',
  'unreachable',
  'not_ready',
  'collection_unhealthy',
  'authentication_failed',
  'invalid_response'
]);
const runtimeHealthServiceSchema: z.ZodType<RuntimeHealthService> = z
  .strictObject({
    key: z.enum(['data', 'agentrun', 'qdrant', 'neo4j']),
    display_name: z.enum(['Data Service', 'AgentRun', 'Qdrant', 'Neo4j']),
    status: runtimeHealthStatusSchema,
    checked_at: timestampSchema,
    latency_ms: nonNegativeIntegerSchema.nullish(),
    reason_code: runtimeHealthReasonSchema.nullish()
  })
  .refine(
    (service) =>
      (service.status === 'ready' && service.reason_code == null) ||
      (service.status !== 'ready' && service.reason_code != null)
  );
const runtimeHealthSchema: z.ZodType<RuntimeHealth> = z
  .strictObject({
    status: z.enum(['ready', 'degraded']),
    checked_at: timestampSchema,
    services: z.array(runtimeHealthServiceSchema).length(4)
  })
  .refine(
    (health) =>
      health.services.map((service) => service.key).join(',') === 'data,agentrun,qdrant,neo4j' &&
      (health.status === 'ready'
        ? health.services.every((service) => service.status === 'ready')
        : health.services.some((service) => service.status !== 'ready'))
  );
const monitoringCountsSchema = z.strictObject({
  success: nonNegativeIntegerSchema,
  running: nonNegativeIntegerSchema,
  failure: nonNegativeIntegerSchema
});
const monitoringSummarySchema: z.ZodType<MonitoringSummary> = z.strictObject({
  window: monitoringWindowSchema,
  generated_at: timestampSchema,
  collector: monitoringCountsSchema,
  artifact_extraction: monitoringCountsSchema,
  semantic: monitoringCountsSchema,
  collector_raw_results: nonNegativeIntegerSchema,
  collector_merged_results: nonNegativeIntegerSchema,
  collector_accepted_artifacts: nonNegativeIntegerSchema,
  artifact_published: nonNegativeIntegerSchema,
  artifact_no_events: nonNegativeIntegerSchema,
  artifact_formal_events: nonNegativeIntegerSchema,
  semantic_submissions: nonNegativeIntegerSchema,
  semantic_accepted_candidates: nonNegativeIntegerSchema,
  semantic_rejected_candidates: nonNegativeIntegerSchema
});
const monitoringPageFields = {
  window: monitoringWindowSchema,
  generated_at: timestampSchema,
  page: z.number().int().positive(),
  page_size: z.number().int().min(1).max(100),
  total_items: nonNegativeIntegerSchema,
  total_pages: nonNegativeIntegerSchema
};
const collectorMonitoringItemSchema: z.ZodType<CollectorMonitoringItem> = z.strictObject({
  execution_id: z.string().min(1),
  state: monitoringStateSchema,
  raw_status: z.string().min(1),
  trigger_source: z.string().min(1),
  started_at: timestampSchema.optional(),
  completed_at: timestampSchema.optional(),
  duration_ms: nonNegativeIntegerSchema.optional(),
  raw_results: nonNegativeIntegerSchema,
  merged_results: nonNegativeIntegerSchema,
  accepted_artifacts: nonNegativeIntegerSchema,
  error_code: z.string().optional()
});
const artifactMonitoringItemSchema: z.ZodType<ArtifactMonitoringItem> = z.strictObject({
  extraction_key: z.string().min(1),
  artifact_id: z.string().min(1),
  collector_execution_id: z.string().min(1),
  state: monitoringStateSchema,
  raw_status: z.string().min(1),
  updated_at: timestampSchema,
  started_at: timestampSchema.optional(),
  completed_at: timestampSchema.optional(),
  duration_ms: nonNegativeIntegerSchema.optional(),
  event_candidates: nonNegativeIntegerSchema,
  acknowledged_journals: nonNegativeIntegerSchema,
  total_journals: nonNegativeIntegerSchema,
  error_code: z.string().optional()
});
const semanticMonitoringItemSchema: z.ZodType<SemanticMonitoringItem> = z.strictObject({
  work_item_id: z.string().min(1),
  event_id: z.string().min(1),
  trigger_source: z.string().min(1),
  state: monitoringStateSchema,
  raw_status: z.string().min(1),
  updated_at: timestampSchema,
  started_at: timestampSchema.optional(),
  completed_at: timestampSchema.optional(),
  duration_ms: nonNegativeIntegerSchema.optional(),
  attempt_count: nonNegativeIntegerSchema,
  max_attempts: z.number().int().positive(),
  accepted_candidates: nonNegativeIntegerSchema,
  rejected_candidates: nonNegativeIntegerSchema,
  error_code: z.string().optional()
});
const collectorMonitoringPageSchema: z.ZodType<MonitoringPage<CollectorMonitoringItem>> =
  z.strictObject({ ...monitoringPageFields, items: z.array(collectorMonitoringItemSchema) });
const artifactMonitoringPageSchema: z.ZodType<MonitoringPage<ArtifactMonitoringItem>> =
  z.strictObject({ ...monitoringPageFields, items: z.array(artifactMonitoringItemSchema) });
const semanticMonitoringPageSchema: z.ZodType<MonitoringPage<SemanticMonitoringItem>> =
  z.strictObject({ ...monitoringPageFields, items: z.array(semanticMonitoringItemSchema) });

export async function loadMonitoringSummary(
  token: string,
  window: MonitoringWindow
): Promise<MonitoringSummary> {
  const result = parseMonitoringResponse(
    monitoringSummarySchema,
    await request<unknown>(token, `/api/admin/v1/monitoring/summary?window=${window}`)
  );
  return requireMonitoringWindow(result, window);
}

export async function loadRuntimeHealth(token: string): Promise<RuntimeHealth> {
  const parsed = runtimeHealthSchema.safeParse(
    await request<unknown>(token, '/api/admin/v1/runtime-health')
  );
  if (!parsed.success) {
    throw new Error('Admin API returned invalid runtime health data');
  }
  return parsed.data;
}
export async function loadCollectorMonitoring(
  token: string,
  window: MonitoringWindow,
  state: MonitoringState,
  page = 1,
  pageSize = 20
): Promise<MonitoringPage<CollectorMonitoringItem>> {
  const result = parseMonitoringResponse(
    collectorMonitoringPageSchema,
    await request<unknown>(
      token,
      monitoringPath('collector-executions', window, state, page, pageSize)
    )
  );
  return requireMonitoringWindow(result, window);
}
export async function loadArtifactMonitoring(
  token: string,
  window: MonitoringWindow,
  state: MonitoringState,
  page = 1,
  pageSize = 20
): Promise<MonitoringPage<ArtifactMonitoringItem>> {
  const result = parseMonitoringResponse(
    artifactMonitoringPageSchema,
    await request<unknown>(
      token,
      monitoringPath('artifact-extractions', window, state, page, pageSize)
    )
  );
  return requireMonitoringWindow(result, window);
}
export async function loadSemanticMonitoring(
  token: string,
  window: MonitoringWindow,
  state: MonitoringState,
  page = 1,
  pageSize = 20
): Promise<MonitoringPage<SemanticMonitoringItem>> {
  const result = parseMonitoringResponse(
    semanticMonitoringPageSchema,
    await request<unknown>(
      token,
      monitoringPath('semantic-work-items', window, state, page, pageSize)
    )
  );
  return requireMonitoringWindow(result, window);
}

function parseMonitoringResponse<T>(schema: z.ZodType<T>, value: unknown): T {
  const parsed = schema.safeParse(value);
  if (!parsed.success) {
    throw new Error('Admin API returned invalid monitoring data');
  }
  return parsed.data;
}

function requireMonitoringWindow<T extends { window: MonitoringWindow }>(
  value: T,
  expected: MonitoringWindow
): T {
  if (value.window !== expected) {
    throw new Error('Admin API returned monitoring data for a different time range');
  }
  return value;
}
function monitoringPath(
  resource: string,
  window: MonitoringWindow,
  state: MonitoringState,
  page: number,
  pageSize: number
): string {
  const params = new URLSearchParams({
    window,
    state,
    page: String(page),
    page_size: String(pageSize)
  });
  return `/api/admin/v1/monitoring/${resource}?${params}`;
}

export async function loadModelProviders(token: string): Promise<ModelProviderConfiguration[]> {
  const result = await request<{ items: ModelProviderConfiguration[] }>(
    token,
    '/api/admin/v1/model-providers'
  );
  return result.items;
}

export function loadModelProvider(
  token: string,
  providerKey: string
): Promise<ModelProviderConfiguration> {
  return request<ModelProviderConfiguration>(
    token,
    `/api/admin/v1/model-providers/${encodeURIComponent(providerKey)}`
  );
}

export function updateModelProvider(
  token: string,
  providerKey: string,
  patch: ModelProviderPatch
): Promise<ModelProviderConfiguration> {
  return request<ModelProviderConfiguration>(
    token,
    `/api/admin/v1/model-providers/${encodeURIComponent(providerKey)}`,
    {
      method: 'PATCH',
      body: JSON.stringify(patch)
    }
  );
}

export async function loadConnectors(token: string): Promise<ConnectorConfiguration[]> {
  const result = await request<{ items: ConnectorConfiguration[] }>(
    token,
    '/api/admin/v1/connectors'
  );
  return result.items;
}

export function loadConnector(
  token: string,
  connectorKey: string
): Promise<ConnectorConfiguration> {
  return request<ConnectorConfiguration>(
    token,
    `/api/admin/v1/connectors/${encodeURIComponent(connectorKey)}`
  );
}

export function updateConnector(
  token: string,
  connectorKey: string,
  patch: ConnectorPatch
): Promise<ConnectorConfiguration> {
  return request<ConnectorConfiguration>(
    token,
    `/api/admin/v1/connectors/${encodeURIComponent(connectorKey)}`,
    {
      method: 'PATCH',
      body: JSON.stringify(patch)
    }
  );
}

async function request<T>(token: string, path: string, init: RequestInit = {}): Promise<T> {
  const headers = new Headers(init.headers);
  headers.set('Authorization', `Bearer ${token}`);
  headers.set('Accept', 'application/json');
  if (init.body) {
    headers.set('Content-Type', 'application/json');
  }
  const response = await fetch(path, { ...init, headers });
  const payload = await readPayload(response);
  if (!response.ok) {
    const error = payload as {
      error?: { code?: string; message?: string };
    };
    throw new AdminAgentRunAPIError(
      response.status,
      error.error?.code ?? 'ADMIN_API_ERROR',
      error.error?.message ?? `request failed with status ${response.status}`
    );
  }
  const success = payload as {
    request_id?: unknown;
    result?: T;
  };
  if (
    typeof success.request_id !== 'string' ||
    success.request_id.length === 0 ||
    !Object.prototype.hasOwnProperty.call(success, 'result')
  ) {
    throw new Error('Admin API returned an invalid response');
  }
  return success.result as T;
}

async function readPayload(response: Response): Promise<unknown> {
  try {
    return await response.json();
  } catch {
    if (response.ok) {
      throw new Error('Admin API returned an invalid response');
    }
    return {};
  }
}
