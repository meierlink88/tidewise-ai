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

export function CollectionPanel({
  actions,
  children,
  description,
  title
}: {
  actions: ReactNode;
  children: ReactNode;
  description: string;
  title: string;
}) {
  return (
    <Card className='flex h-full min-h-0 flex-col gap-3 overflow-y-auto p-4 shadow-xs'>
      <div className='flex flex-col gap-3 sm:flex-row sm:items-start sm:justify-between'>
        <div className='min-w-0'>
          <h3 className='text-sm font-semibold tracking-tight'>{title}</h3>
          <p className='mt-1 text-xs text-muted-foreground'>{description}</p>
        </div>
        <div
          aria-label={`${title}操作`}
          className='flex shrink-0 flex-wrap items-center justify-end gap-2'
          role='toolbar'
        >
          {actions}
        </div>
      </div>
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
  'grid items-end gap-3 [&>div]:gap-1.5 [&_input]:text-xs [&_label]:text-xs [&_label]:text-muted-foreground [&_[role=combobox]]:text-xs sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-6';

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
