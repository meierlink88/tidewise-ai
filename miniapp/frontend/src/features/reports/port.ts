import { mockReportPort } from '../../mocks/reports/mock-port';
import type { ReportPort } from './contract';
import { APIReportPort, normalizeReportAPIBaseURL } from './api-port';

let selectedPort: ReportPort | undefined;

export function createReportPort(
  source: string | undefined,
  apiBaseURL: string | undefined,
  platform: string | undefined
): ReportPort {
  if (source === 'mock') return mockReportPort;
  if (source === 'api') {
    return new APIReportPort(normalizeReportAPIBaseURL(apiBaseURL ?? '', platform ?? ''));
  }
  throw new Error('TARO_APP_REPORT_SOURCE must explicitly be mock or api');
}

export function getReportPort(): ReportPort {
  selectedPort ??= createReportPort(
    process.env.TARO_APP_REPORT_SOURCE,
    process.env.TARO_APP_MINIAPP_API_BASE_URL,
    process.env.TARO_ENV
  );
  return selectedPort;
}
