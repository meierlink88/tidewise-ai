export type SchedulerMode = 'interval' | 'fixed_times';

export interface SchedulerSourceFilter {
  provider_key?: string;
  ingest_channel?: string;
  source_type?: string;
}

export interface SchedulerConfig {
  id?: string;
  enabled: boolean;
  mode: SchedulerMode;
  interval_minutes: number;
  fixed_times: string[];
  concurrency: number;
  batch_size: number;
  timeout_seconds: number;
  source_filter: SchedulerSourceFilter;
  timezone: string;
  config_version?: number;
  recent_run?: SchedulerRun;
}

export interface SchedulerRun {
  id: string;
  trigger_type: string;
  status: string;
  started_at: string;
  finished_at?: string;
  total_sources: number;
  succeeded_sources: number;
  failed_sources: number;
  skipped_sources: number;
  error_summary: string;
}

export async function loadSchedulerConfig(token: string): Promise<SchedulerConfig> {
  const response = await fetch('/admin/scheduler/config', {
    headers: authHeaders(token)
  });
  return readJSON(response);
}

export async function saveSchedulerConfig(token: string, config: SchedulerConfig): Promise<SchedulerConfig> {
  const response = await fetch('/admin/scheduler/config', {
    method: 'PUT',
    headers: {
      ...authHeaders(token),
      'Content-Type': 'application/json'
    },
    body: JSON.stringify(config)
  });
  return readJSON(response);
}

function authHeaders(token: string): Record<string, string> {
  return {
    Authorization: `Bearer ${token}`
  };
}

async function readJSON<T>(response: Response): Promise<T> {
  if (!response.ok) {
    throw new Error(`request failed with status ${response.status}`);
  }
  return response.json() as Promise<T>;
}
