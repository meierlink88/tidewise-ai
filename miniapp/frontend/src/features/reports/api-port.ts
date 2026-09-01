import Taro from '@tarojs/taro';
import { unwrapMiniappAPIEnvelope } from '../../platform/miniapp-api';
import type {
  ReportErrorKind,
  ReportEvidenceScope,
  ReportLayerKey,
  ReportPort
} from './contract';
import { ReportError } from './contract';
import {
  parseReportEvidenceListWire,
  parseReportHomeWire,
  parseReportIndustryChainDetailWire,
  parseReportLayerDetailWire
} from './wire-contract';

const requestTimeoutMs = 10_000;

export class APIReportPort implements ReportPort {
  constructor(private readonly baseURL: string) {}

  async getHome() {
    const result = await this.get('/api/miniapp/v1/reports/home', 'reportUnavailable');
    return parseResponse(result, parseReportHomeWire);
  }

  async getLayer(reportId: string, layerKey: ReportLayerKey) {
    const result = await this.get(
      `/api/miniapp/v1/reports/${encodeURIComponent(reportId)}/layers/${encodeURIComponent(layerKey)}`,
      'layerUnavailable'
    );
    return parseResponse(result, (value) =>
      parseReportLayerDetailWire(value, reportId, layerKey)
    );
  }

  async getIndustryChain(reportId: string, chainKey: string) {
    const result = await this.get(
      `/api/miniapp/v1/reports/${encodeURIComponent(reportId)}/industry-chains/${encodeURIComponent(chainKey)}`,
      'chainUnavailable'
    );
    return parseResponse(result, (value) =>
      parseReportIndustryChainDetailWire(value, reportId, chainKey)
    );
  }

  async getEvidences(reportId: string, scope: ReportEvidenceScope) {
    const query = `scope_type=${encodeURIComponent(scope.type)}&scope_key=${encodeURIComponent(scope.key)}`;
    const result = await this.get(
      `/api/miniapp/v1/reports/${encodeURIComponent(reportId)}/evidences?${query}`,
      'evidenceScopeUnavailable'
    );
    return parseResponse(result, (value) =>
      parseReportEvidenceListWire(value, reportId, scope)
    );
  }

  private async get(path: string, missingKind: ReportErrorKind): Promise<unknown> {
    let response: Awaited<ReturnType<typeof Taro.request<unknown>>>;
    try {
      response = await Taro.request<unknown>({
        url: `${this.baseURL}${path}`,
        method: 'GET',
        timeout: requestTimeoutMs,
        header: { accept: 'application/json' }
      });
    } catch {
      throw new ReportError('serviceUnavailable');
    }
    if (response.statusCode < 200 || response.statusCode >= 300) {
      if (response.statusCode === 400) throw new ReportError('invalidRequest');
      if (response.statusCode === 404) throw new ReportError(missingKind);
      throw new ReportError('serviceUnavailable');
    }
    const result = unwrapReportEnvelope(response.data);
    if (result === undefined) throw new ReportError('invalidResponse');
    return result;
  }
}

function unwrapReportEnvelope(value: unknown): unknown | undefined {
  if (typeof value !== 'object' || value === null || Array.isArray(value)) return undefined;
  const keys = Object.keys(value);
  if (keys.length !== 2 || !keys.includes('request_id') || !keys.includes('result')) {
    return undefined;
  }
  return unwrapMiniappAPIEnvelope<unknown>(value);
}

function parseResponse<T>(value: unknown, parser: (wire: unknown) => T): T {
  try {
    return parser(value);
  } catch {
    throw new ReportError('invalidResponse');
  }
}

export function normalizeReportAPIBaseURL(value: string, platform: string): string {
  const normalized = value.trim().replace(/\/+$/, '');
  if (normalized === '' && platform === 'h5') return '';
  if (!/^https?:\/\/[^/]+(?:\/.*)?$/i.test(normalized)) {
    throw new Error(
      'TARO_APP_MINIAPP_API_BASE_URL must be an absolute HTTP(S) URL outside H5 proxy mode'
    );
  }
  return normalized;
}
