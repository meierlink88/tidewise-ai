import { adminAPIURL } from './dataIngestion';

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

export interface AgentExecution {
  execution_id: string;
  agent_key: string;
  agent_version: string;
  trigger_source: string;
  schedule_id?: string;
  status: string;
  error_code?: string;
  error_summary?: string;
  stop_reason?: string;
  blocked_by_execution_id?: string;
  created_at: string;
  triggered_at: string;
  started_at?: string;
  completed_at?: string;
}

export interface AgentExecutionPage {
  items: AgentExecution[];
  page: number;
  page_size: 20;
  total_items: number;
  total_pages: number;
}

export interface AgentStatus {
  agent_key: string;
  display_name: string;
  current_version: string;
  is_working: boolean;
  current_execution_status: string;
  updated_at: string;
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

export function loadAgentExecutions(token: string, page: number): Promise<AgentExecutionPage> {
  const params = new URLSearchParams({ page: String(page) });
  return request<AgentExecutionPage>(token, `/api/admin/v1/agent-executions?${params.toString()}`);
}

export async function loadAgentStatuses(token: string): Promise<AgentStatus[]> {
  const result = await request<{ items: AgentStatus[] }>(token, '/api/admin/v1/agent-statuses');
  return result.items;
}

export function loadMonitoringSummary(
  token: string,
  window: MonitoringWindow
): Promise<MonitoringSummary> {
  return request(token, `/api/admin/v1/monitoring/summary?window=${window}`);
}
export function loadCollectorMonitoring(
  token: string,
  window: MonitoringWindow,
  state: MonitoringState,
  page = 1,
  pageSize = 20
): Promise<MonitoringPage<CollectorMonitoringItem>> {
  return request(token, monitoringPath('collector-executions', window, state, page, pageSize));
}
export function loadArtifactMonitoring(
  token: string,
  window: MonitoringWindow,
  state: MonitoringState,
  page = 1,
  pageSize = 20
): Promise<MonitoringPage<ArtifactMonitoringItem>> {
  return request(token, monitoringPath('artifact-extractions', window, state, page, pageSize));
}
export function loadSemanticMonitoring(
  token: string,
  window: MonitoringWindow,
  state: MonitoringState,
  page = 1,
  pageSize = 20
): Promise<MonitoringPage<SemanticMonitoringItem>> {
  return request(token, monitoringPath('semantic-work-items', window, state, page, pageSize));
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
  const response = await fetch(adminAPIURL(path), { ...init, headers });
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
