import type { ReportErrorKind, ReportEvidence } from './contract';

export function formatShanghaiTimestamp(value: string): string {
  const timestamp = Date.parse(value);
  if (!Number.isFinite(timestamp)) return '时间待确认';
  const date = new Date(timestamp + 8 * 60 * 60 * 1000);
  return `${date.getUTCFullYear()}.${two(date.getUTCMonth() + 1)}.${two(date.getUTCDate())} ${two(date.getUTCHours())}:${two(date.getUTCMinutes())}`;
}

export function evidenceStableKey(item: ReportEvidence): string {
  const source = `${item.publishedAt ?? 'pending'}\u0000${item.summary}\u0000${item.keywords.join('\u0000')}`;
  let hash = 2166136261;
  for (let index = 0; index < source.length; index += 1) {
    hash ^= source.charCodeAt(index);
    hash = Math.imul(hash, 16777619);
  }
  return `evidence-${(hash >>> 0).toString(16).padStart(8, '0')}`;
}

export function reportErrorCopy(kind: ReportErrorKind): {
  title: string;
  description: string;
} {
  switch (kind) {
    case 'invalidRequest':
      return { title: '页面参数无效', description: '请返回上一页后重新进入' };
    case 'reportUnavailable':
      return { title: '报告不可用', description: '该报告可能尚未发布或已不可访问' };
    case 'layerUnavailable':
      return { title: '推理层不可用', description: '该报告没有发布此层详情' };
    case 'chainUnavailable':
      return { title: '产业链不可用', description: '该报告没有发布此产业链详情' };
    case 'evidenceScopeUnavailable':
      return { title: '相关证据不可用', description: '该对象没有可读取的证据投影' };
    case 'invalidResponse':
      return { title: '报告数据异常', description: '服务返回内容不完整，请稍后重试' };
    case 'serviceUnavailable':
    default:
      return { title: '暂时无法读取', description: '网络或服务暂时不可用，请稍后重试' };
  }
}

function two(value: number): string {
  return String(value).padStart(2, '0');
}
