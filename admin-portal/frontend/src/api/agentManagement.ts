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
  return request<AgentSchedule>(token, `/api/admin/v1/agent-schedules/${encodeURIComponent(agentKey)}`);
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

export async function loadModelProviders(
  token: string
): Promise<ModelProviderConfiguration[]> {
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
