export interface MiniappAPIEnvelope<T> {
  request_id: string;
  result: T;
}

export function normalizeMiniappAPIBaseURL(value: string): string {
  const normalized = value.trim().replace(/\/+$/, '');
  if (!/^https?:\/\/[^/]+/i.test(normalized)) {
    throw new Error('TARO_APP_MINIAPP_API_BASE_URL must be an absolute HTTP(S) URL in api mode');
  }
  return normalized;
}

export function unwrapMiniappAPIEnvelope<T>(value: unknown): T | undefined {
  if (typeof value !== 'object' || value === null) return undefined;
  const envelope = value as Partial<MiniappAPIEnvelope<T>>;
  if (typeof envelope.request_id !== 'string' || envelope.request_id.length === 0 || !('result' in envelope)) {
    return undefined;
  }
  return envelope.result;
}
