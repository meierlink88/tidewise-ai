import { keepPreviousData, useQuery } from '@tanstack/react-query';
import { Search } from 'lucide-react';
import { type ChangeEvent, type FormEvent, useMemo, useRef, useState } from 'react';
import { loadEvents, type EventItem, type EventQuery } from '../../api/dataIngestion';
import { DataTable, type DataTableColumn } from '../../components/admin/data-table';
import { OverflowTooltip } from '../../components/admin/overflow-tooltip';
import { Pagination } from '../../components/admin/pagination';
import { Button } from '../../components/ui/Button';
import { Field } from '../../components/ui/Field';
import { Input } from '../../components/ui/Input';
import { Select } from '../../components/ui/Select';
import { StatusBadge } from '../../components/ui/StatusBadge';
import {
  CollectionPanel,
  filterGridClass,
  formatDateTime,
  pageSize,
  QueryFailure,
  toRFC3339
} from './shared';
import { EventDetailSheet } from './EventDetailSheet';

export default function EventTab({ token }: { token: string }) {
  const [selectedEventId, setSelectedEventId] = useState<string | null>(null);
  const [detailOpen, setDetailOpen] = useState(false);
  const detailTrigger = useRef<HTMLTableRowElement | null>(null);
  const [form, setForm] = useState({
    title: '',
    modality: '',
    status: '',
    occurredFrom: '',
    occurredTo: '',
    announcedFrom: '',
    announcedTo: ''
  });
  const filterFormId = 'event-filters';
  const [query, setQuery] = useState<EventQuery>({ page: 1, title: '' });
  const result = useQuery({
    queryKey: ['collection-center', 'events', query],
    queryFn: () => loadEvents(token, query),
    placeholderData: keepPreviousData
  });
  const page = result.data ?? { items: [], total: 0, page: query.page, page_size: pageSize };
  const selectedEvent = page.items.find((item) => item.id === selectedEventId) ?? null;
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
        render: (item) => item.semantic.modality
      },
      {
        headerClassName: 'w-[14%]',
        key: 'status',
        header: '状态',
        render: (item) => (
          <StatusBadge
            tone={
              item.status === 'ACTIVE'
                ? 'success'
                : item.status === 'DEPRECATED'
                  ? 'danger'
                  : 'neutral'
            }
          >
            {item.status}
          </StatusBadge>
        )
      },
      {
        headerClassName: 'w-[20%]',
        key: 'occurred_at',
        header: '发生时间',
        render: (item) =>
          item.semantic.time.occurred_at ? formatDateTime(item.semantic.time.occurred_at) : '-'
      },
      {
        headerClassName: 'w-[20%]',
        key: 'announced_at',
        header: '公布时间',
        render: (item) =>
          item.semantic.time.announced_at ? formatDateTime(item.semantic.time.announced_at) : '-'
      }
    ],
    []
  );
  const submit = (event: FormEvent) => {
    event.preventDefault();
    setQuery({
      page: 1,
      title: form.title,
      modality: form.modality || undefined,
      status: form.status || undefined,
      occurred_from: toRFC3339(form.occurredFrom),
      occurred_to: toRFC3339(form.occurredTo),
      announced_from: toRFC3339(form.announcedFrom),
      announced_to: toRFC3339(form.announcedTo)
    });
  };
  const field = (name: keyof typeof form) => ({
    value: form[name],
    onChange: (event: ChangeEvent<HTMLInputElement>) =>
      setForm((current) => ({ ...current, [name]: event.target.value }))
  });
  return (
    <CollectionPanel
      actions={
        <Button className='text-xs' form={filterFormId} size='sm' type='submit'>
          <Search aria-hidden='true' className='size-4' />
          搜索事件
        </Button>
      }
      description='查询标准化事件及其生命周期状态。'
      title='事件中心'
    >
      <QueryFailure
        error={result.error}
        fetching={result.isFetching}
        retry={() => void result.refetch()}
      />
      <form
        aria-label='事件筛选条件'
        className={filterGridClass}
        id={filterFormId}
        onSubmit={submit}
      >
        <Field label='事件标题搜索'>
          <Input aria-label='事件标题搜索' {...field('title')} />
        </Field>
        <Field label='模态'>
          <Select
            ariaLabel='模态'
            onValueChange={(value) =>
              setForm((v) => ({ ...v, modality: value === 'all' ? '' : value }))
            }
            options={[
              { label: '全部', value: 'all' },
              { label: '事实', value: 'FACT' },
              { label: '计划', value: 'PLAN' },
              { label: '推测', value: 'SPEC' }
            ]}
            value={form.modality || 'all'}
          />
        </Field>
        <Field label='状态'>
          <Select
            ariaLabel='状态'
            onValueChange={(value) =>
              setForm((v) => ({ ...v, status: value === 'all' ? '' : value }))
            }
            options={[
              { label: '全部', value: 'all' },
              { label: '活跃', value: 'ACTIVE' },
              { label: '已废弃', value: 'DEPRECATED' },
              { label: '已归档', value: 'ARCHIVED' }
            ]}
            value={form.status || 'all'}
          />
        </Field>
        <Field label='发生时间开始'>
          <Input aria-label='发生时间开始' type='datetime-local' {...field('occurredFrom')} />
        </Field>
        <Field label='发生时间结束'>
          <Input aria-label='发生时间结束' type='datetime-local' {...field('occurredTo')} />
        </Field>
        <Field label='公布时间开始'>
          <Input aria-label='公布时间开始' type='datetime-local' {...field('announcedFrom')} />
        </Field>
        <Field label='公布时间结束'>
          <Input aria-label='公布时间结束' type='datetime-local' {...field('announcedTo')} />
        </Field>
      </form>
      <DataTable
        className='h-full'
        columns={columns}
        emptyText={result.isPending ? '正在加载事件' : '暂无事件'}
        getRowKey={(item) => item.id}
        items={page.items}
        onRowActivate={(item, trigger) => {
          detailTrigger.current = trigger;
          setSelectedEventId(item.id);
          setDetailOpen(true);
        }}
        rowAccessibleName={(item) => `查看${item.title}详情`}
        scrollAreaLabel='事件表格滚动区域'
        selectedRowKey={detailOpen ? (selectedEventId ?? undefined) : undefined}
        tableClassName='min-w-[720px] table-fixed text-xs [&_td]:py-2.5 [&_th]:h-9'
      />
      <Pagination
        page={page.page}
        pageSize={page.page_size}
        total={page.total}
        onPageChange={(next) => setQuery((current) => ({ ...current, page: next }))}
      />
      <EventDetailSheet
        event={selectedEvent}
        open={detailOpen}
        onOpenChange={(open) => {
          if (open) return;
          setDetailOpen(false);
          window.setTimeout(() => detailTrigger.current?.focus(), 0);
        }}
      />
    </CollectionPanel>
  );
}
