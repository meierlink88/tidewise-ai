import { describe, expect, it, vi } from 'vitest';
import { APIReportPort } from './api-port';
import { createReportPort } from './port';

vi.mock('@tarojs/taro', () => ({ default: { request: vi.fn() } }));

describe('Report port selection', () => {
  it('selects mock only when explicitly configured', () => {
    expect(createReportPort('mock', undefined, 'weapp').constructor.name).toBe('MockReportPort');
  });

  it('selects production API without a mock fallback', () => {
    expect(createReportPort('api', 'https://miniapp.example.com/', 'weapp')).toBeInstanceOf(
      APIReportPort
    );
    expect(() => createReportPort('api', '', 'weapp')).toThrow(
      'TARO_APP_MINIAPP_API_BASE_URL'
    );
    expect(createReportPort('api', '', 'h5')).toBeInstanceOf(APIReportPort);
  });

  it('fails closed when the source was not selected', () => {
    expect(() => createReportPort(undefined, undefined, 'h5')).toThrow(
      'TARO_APP_REPORT_SOURCE'
    );
  });
});
