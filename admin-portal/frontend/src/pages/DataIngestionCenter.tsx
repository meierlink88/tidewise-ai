import { type FormEvent, useEffect, useMemo, useState } from 'react';
import { loadEvents, type EventItem, type EventQuery } from '../api/dataIngestion';
import { DataTable, type DataTableColumn } from '../components/admin/data-table';
import { OverflowTooltip } from '../components/admin/overflow-tooltip';
import { Pagination } from '../components/admin/pagination';
import StatusAlert from '../components/admin/status-alert';
import { Button } from '../components/ui/Button';
import { Card } from '../components/ui/Card';
import { Field } from '../components/ui/Field';
import { Input } from '../components/ui/Input';
import { Select } from '../components/ui/Select';
import { StatusBadge } from '../components/ui/StatusBadge';

const pageSize = 50;

export default function DataIngestionCenter({ token }: { token: string }) {
  const [title, setTitle] = useState('');
  const [modality, setModality] = useState('');
  const [status, setStatus] = useState('');
  const [occurredFrom, setOccurredFrom] = useState('');
  const [occurredTo, setOccurredTo] = useState('');
  const [announcedFrom, setAnnouncedFrom] = useState('');
  const [announcedTo, setAnnouncedTo] = useState('');
  const [query, setQuery] = useState<EventQuery>({ page: 1, title: '' });
  const [page, setPage] = useState({
    items: [] as EventItem[],
    total: 0,
    page: 1,
    page_size: pageSize
  });
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState('');

  useEffect(() => {
    let active = true;
    setLoading(true);
    setError('');
    loadEvents(token, query)
      .then((result) => active && setPage(result))
      .catch((loadError) => active && setError(errorText(loadError)))
      .finally(() => active && setLoading(false));
    return () => {
      active = false;
    };
  }, [query, token]);

  const columns = useMemo<DataTableColumn<EventItem>[]>(
    () => [
      {
        cellClassName: 'max-w-0',
        headerClassName: 'w-[34%]',
        key: 'title',
        header: '事件标题',
        render: (item) => <OverflowTooltip className='font-semibold' value={item.title} />
      },
      {
        headerClassName: 'w-[12%]',
        key: 'modality',
        header: '模态',
        render: (item) => item.modality
      },
      {
        headerClassName: 'w-[14%]',
        key: 'status',
        header: '状态',
        render: (item) => <StatusBadge tone={statusTone(item.status)}>{item.status}</StatusBadge>
      },
      {
        headerClassName: 'w-[20%]',
        key: 'occurred_at',
        header: '发生时间',
        render: (item) => (item.occurred_at ? formatDateTime(item.occurred_at) : '-')
      },
      {
        headerClassName: 'w-[20%]',
        key: 'announced_at',
        header: '公布时间',
        render: (item) => (item.announced_at ? formatDateTime(item.announced_at) : '-')
      }
    ],
    []
  );

  const submitSearch = (event: FormEvent) => {
    event.preventDefault();
    setQuery({
      page: 1,
      title,
      modality: modality || undefined,
      status: status || undefined,
      occurred_from: toRFC3339(occurredFrom),
      occurred_to: toRFC3339(occurredTo),
      announced_from: toRFC3339(announcedFrom),
      announced_to: toRFC3339(announcedTo)
    });
  };

  return (
    <div className='grid h-full min-h-0 w-full grid-rows-[auto_minmax(0,1fr)] gap-3'>
      <div>
        <span className='page-eyebrow'>Data operations</span>
        <h2 className='page-title'>事件中心</h2>
        <p className='page-description'>查询标准化事件。证据通过 Event Evidence Link 关联。</p>
      </div>
      <Card className='flex h-full min-h-0 flex-col gap-3 overflow-y-auto p-4 shadow-xs'>
        {error ? (
          <StatusAlert
            actionDisabled={loading}
            actionLabel={loading ? '重试中…' : '重试'}
            onAction={() => setQuery((current) => ({ ...current }))}
            tone='destructive'
          >
            {error}
          </StatusAlert>
        ) : null}
        <form
          className='grid items-end gap-3 [&>div]:gap-1.5 [&_input]:text-xs [&_label]:text-xs [&_label]:text-muted-foreground [&_[role=combobox]]:text-xs sm:grid-cols-2 lg:grid-cols-4'
          onSubmit={submitSearch}
        >
          <Field label='事件标题搜索'>
            <Input
              aria-label='事件标题搜索'
              onChange={(event) => setTitle(event.target.value)}
              value={title}
            />
          </Field>
          <Field label='模态'>
            <Select
              ariaLabel='模态'
              onValueChange={(value) => setModality(value === 'all' ? '' : value)}
              options={[
                { label: '全部', value: 'all' },
                { label: '事实', value: 'FACT' },
                { label: '计划', value: 'PLAN' },
                { label: '推测', value: 'SPEC' }
              ]}
              value={modality || 'all'}
            />
          </Field>
          <Field label='状态'>
            <Select
              ariaLabel='状态'
              onValueChange={(value) => setStatus(value === 'all' ? '' : value)}
              options={[
                { label: '全部', value: 'all' },
                { label: '活跃', value: 'ACTIVE' },
                { label: '已废弃', value: 'DEPRECATED' },
                { label: '已归档', value: 'ARCHIVED' }
              ]}
              value={status || 'all'}
            />
          </Field>
          <Field label='发生时间开始'>
            <Input
              aria-label='发生时间开始'
              onChange={(event) => setOccurredFrom(event.target.value)}
              type='datetime-local'
              value={occurredFrom}
            />
          </Field>
          <Field label='发生时间结束'>
            <Input
              aria-label='发生时间结束'
              onChange={(event) => setOccurredTo(event.target.value)}
              type='datetime-local'
              value={occurredTo}
            />
          </Field>
          <Field label='公布时间开始'>
            <Input
              aria-label='公布时间开始'
              onChange={(event) => setAnnouncedFrom(event.target.value)}
              type='datetime-local'
              value={announcedFrom}
            />
          </Field>
          <Field label='公布时间结束'>
            <Input
              aria-label='公布时间结束'
              onChange={(event) => setAnnouncedTo(event.target.value)}
              type='datetime-local'
              value={announcedTo}
            />
          </Field>
          <Button className='text-xs' size='sm' type='submit'>
            搜索事件
          </Button>
        </form>
        <DataTable
          className='h-full'
          columns={columns}
          emptyText={loading ? '正在加载事件' : '暂无事件'}
          getRowKey={(item) => item.id}
          items={page.items}
          scrollAreaLabel='事件表格滚动区域'
          tableClassName='min-w-[720px] table-fixed text-xs [&_td]:py-2.5 [&_th]:h-9'
        />
        <Pagination
          page={page.page}
          pageSize={page.page_size}
          total={page.total}
          onPageChange={(nextPage) => setQuery((current) => ({ ...current, page: nextPage }))}
        />
      </Card>
    </div>
  );
}

function errorText(error: unknown): string {
  const message = error instanceof Error ? error.message.trim() : '';
  return !message ||
    /internal server error|admin api returned|request failed with status/i.test(message)
    ? '数据加载失败，请稍后重试。'
    : message;
}

function formatDateTime(value: string): string {
  return new Date(value).toLocaleString('zh-CN', { hour12: false, timeZone: 'Asia/Shanghai' });
}

function statusTone(status: string): 'success' | 'danger' | 'neutral' {
  if (status === 'ACTIVE') return 'success';
  if (status === 'DEPRECATED') return 'danger';
  return 'neutral';
}

function toRFC3339(value: string): string | undefined {
  if (!value) return undefined;
  const date = new Date(value);
  return Number.isNaN(date.getTime()) ? undefined : date.toISOString();
}
