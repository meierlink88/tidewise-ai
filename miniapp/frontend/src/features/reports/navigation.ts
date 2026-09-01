import type {
  ReportDetailTargetType,
  ReportEvidenceScopeType,
  ReportLayerKey
} from './contract';

const reportIDPattern =
  /^RPT[0-9a-f]{8}-[0-9a-f]{4}-[1-5][0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$/;
const localKeyPattern = /^[a-z0-9][a-z0-9._-]{0,127}$/;
const evidenceRouteTitleMaxCodePoints = 80;
const layerKeys = ['geopolitics', 'macroeconomics'] as const;
const targetTypes = ['layer', 'industry_chain'] as const;
const scopeTypes = [
  'report_card',
  'layer',
  'anchor',
  'reasoning_step',
  'transmission_path',
  'candidate_mechanism',
  'industry_chain',
  'industry_chain_node'
] as const;

export interface ReportDetailRoute {
  reportId: string;
  targetType: ReportDetailTargetType;
  targetKey: string;
}

export interface ReportEvidenceRoute {
  reportId: string;
  scopeType: ReportEvidenceScopeType;
  scopeKey: string;
  title: string;
}

export interface ReportNavigator {
  navigateTo(options: { url: string }): Promise<unknown> | unknown;
}

export function parseReportDetailRoute(value: unknown): ReportDetailRoute {
  return readReportDetailRoute(value, true);
}

function readReportDetailRoute(
  value: unknown,
  normalizeInboundValues: boolean
): ReportDetailRoute {
  const params = routeRecord(value, ['reportId', 'targetType', 'targetKey']);
  const reportId = reportID(routeParam(params.reportId, normalizeInboundValues));
  const targetType = enumParam<ReportDetailTargetType>(
    routeParam(params.targetType, normalizeInboundValues),
    targetTypes
  );
  const targetKey = localKey(routeParam(params.targetKey, normalizeInboundValues));
  if (targetType === 'layer' && !layerKeys.includes(targetKey as ReportLayerKey)) invalidRoute();
  return { reportId, targetType, targetKey };
}

export function parseReportEvidenceRoute(
  value: unknown,
  decodeInboundValues = true
): ReportEvidenceRoute {
  return readReportEvidenceRoute(value, decodeInboundValues);
}

function readReportEvidenceRoute(
  value: unknown,
  normalizeInboundValues: boolean
): ReportEvidenceRoute {
  const params = routeRecord(value, ['reportId', 'scopeType', 'scopeKey', 'title']);
  return {
    reportId: reportID(routeParam(params.reportId, normalizeInboundValues)),
    scopeType: enumParam<ReportEvidenceScopeType>(
      routeParam(params.scopeType, normalizeInboundValues),
      scopeTypes
    ),
    scopeKey: localKey(routeParam(params.scopeKey, normalizeInboundValues)),
    title: textParam(
      routeParam(params.title, normalizeInboundValues),
      evidenceRouteTitleMaxCodePoints
    )
  };
}

export function buildReportDetailURL(route: ReportDetailRoute): string {
  const parsed = readReportDetailRoute(route, false);
  return `/pages/report/detail/index?${query({
    reportId: parsed.reportId,
    targetType: parsed.targetType,
    targetKey: parsed.targetKey
  })}`;
}

export function buildReportEvidenceURL(route: ReportEvidenceRoute): string {
  const parsed = readReportEvidenceRoute({
    ...route,
    title: evidenceRouteTitle(route.title)
  }, false);
  return `/pages/report/evidences/index?${query({
    reportId: parsed.reportId,
    scopeType: parsed.scopeType,
    scopeKey: parsed.scopeKey,
    title: parsed.title
  })}`;
}

function evidenceRouteTitle(value: unknown): string {
  if (
    typeof value !== 'string' ||
    value.trim() !== value ||
    value.length === 0 ||
    /[\u0000-\u001f\u007f]/.test(value)
  ) {
    invalidRoute();
  }
  const title = value;
  const codePoints = Array.from(title);
  if (codePoints.length <= evidenceRouteTitleMaxCodePoints) return title;
  return `${codePoints.slice(0, evidenceRouteTitleMaxCodePoints - 1).join('')}…`;
}

export function navigateToReportDetail(
  navigator: ReportNavigator,
  route: ReportDetailRoute
): void {
  void navigator.navigateTo({ url: buildReportDetailURL(route) });
}

export function navigateToReportEvidences(
  navigator: ReportNavigator,
  route: ReportEvidenceRoute
): void {
  void navigator.navigateTo({ url: buildReportEvidenceURL(route) });
}

function routeRecord(value: unknown, keys: readonly string[]): Record<string, unknown> {
  if (typeof value !== 'object' || value === null || Array.isArray(value)) invalidRoute();
  const result = value as Record<string, unknown>;
  const actualKeys = Object.keys(result);
  const expected = new Set(keys);
  if (keys.some((key) => !Object.prototype.hasOwnProperty.call(result, key))) {
    invalidRoute();
  }
  if (
    actualKeys.some(
      (key) => !expected.has(key) && key !== 'stamp' && key !== '$taroTimestamp'
    )
  ) {
    invalidRoute();
  }
  validateTaroRouteInternals(result);
  return result;
}

function validateTaroRouteInternals(params: Record<string, unknown>): void {
  if (Object.prototype.hasOwnProperty.call(params, 'stamp')) {
    const stamp = params.stamp;
    if (typeof stamp !== 'string' || !/^[A-Za-z]{1,16}$/.test(stamp)) invalidRoute();
  }
  if (Object.prototype.hasOwnProperty.call(params, '$taroTimestamp')) {
    const timestamp = params.$taroTimestamp;
    if (typeof timestamp !== 'number' || !Number.isSafeInteger(timestamp) || timestamp < 0) {
      invalidRoute();
    }
  }
}

function reportID(value: unknown): string {
  const id = textParam(value, 39);
  if (!reportIDPattern.test(id)) invalidRoute();
  return id;
}

function localKey(value: unknown): string {
  const key = textParam(value, 128);
  if (!localKeyPattern.test(key)) invalidRoute();
  return key;
}

function textParam(value: unknown, maxLength: number): string {
  if (
    typeof value !== 'string' ||
    value.trim() !== value ||
    value.length === 0 ||
    Array.from(value).length > maxLength ||
    /[\u0000-\u001f\u007f]/.test(value)
  ) {
    invalidRoute();
  }
  return value;
}

function routeParam(value: unknown, normalizeInboundValue: boolean): unknown {
  if (!normalizeInboundValue || typeof value !== 'string' || !value.includes('%')) {
    return value;
  }
  if (!/%[0-9a-fA-F]{2}/.test(value)) return value;
  if (/%(?![0-9a-fA-F]{2})/.test(value)) invalidRoute();

  let decoded: string;
  try {
    decoded = decodeURIComponent(value);
  } catch {
    invalidRoute();
  }
  return decoded;
}

function enumParam<T extends string>(value: unknown, values: readonly T[]): T {
  const parsed = textParam(value, 64);
  if (!values.includes(parsed as T)) invalidRoute();
  return parsed as T;
}

function query(values: Record<string, string>): string {
  return Object.entries(values)
    .map(([key, value]) => `${encodeURIComponent(key)}=${encodeURIComponent(value)}`)
    .join('&');
}

function invalidRoute(): never {
  throw new Error('invalid Report route');
}
