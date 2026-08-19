import type { ReactNode } from 'react';
import StatusAlert from '../../components/admin/status-alert';
import { Card } from '../../components/ui/Card';

export const pageSize = 50;
export const sourceLevels = [
  { label: '全部', value: 'all' },
  { label: 'L1 官方', value: 'L1_OFFICIAL' },
  { label: 'L2 通讯社', value: 'L2_WIRE' },
  { label: 'L3 媒体', value: 'L3_MEDIA' },
  { label: 'L4 社交媒体', value: 'L4_SOCIAL' }
];

export function CollectionPanel({ children }: { children: ReactNode }) {
  return (
    <Card className='flex h-full min-h-0 flex-col gap-3 overflow-y-auto p-4 shadow-xs'>
      {children}
    </Card>
  );
}

export function QueryFailure({
  error,
  fetching,
  retry
}: {
  error: unknown;
  fetching: boolean;
  retry: () => void;
}) {
  if (!error) return null;
  return (
    <StatusAlert
      actionDisabled={fetching}
      actionLabel={fetching ? '重试中…' : '重试'}
      onAction={retry}
      tone='destructive'
    >
      {errorText(error)}
    </StatusAlert>
  );
}

export const filterGridClass =
  'grid items-end gap-3 [&>div]:gap-1.5 [&_input]:text-xs [&_label]:text-xs [&_label]:text-muted-foreground [&_[role=combobox]]:text-xs sm:grid-cols-2 lg:grid-cols-4';

export function formatDateTime(value: string): string {
  return new Date(value).toLocaleString('zh-CN', { hour12: false, timeZone: 'Asia/Shanghai' });
}

export function toRFC3339(value: string): string | undefined {
  if (!value) return undefined;
  const date = new Date(value);
  return Number.isNaN(date.getTime()) ? undefined : date.toISOString();
}

function errorText(error: unknown): string {
  const message = error instanceof Error ? error.message.trim() : '';
  return !message ||
    /internal server error|admin api returned|request failed with status/i.test(message)
    ? '数据加载失败，请稍后重试。'
    : message;
}
