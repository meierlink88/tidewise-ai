import { keepPreviousData, useQuery } from '@tanstack/react-query';
import { type ChangeEvent, type FormEvent, useMemo, useState } from 'react';
import { loadSources, type SourceItem, type SourceQuery } from '../../api/dataIngestion';
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
  sourceLevels,
  toRFC3339
} from './shared';

export default function SourceTab({ token }: { token: string }) {
  const [form, setForm] = useState({
    query: '',
    ownershipType: '',
    channelType: '',
    enabled: '',
    priority: '',
    defaultSourceLevel: '',
    updatedFrom: '',
    updatedTo: ''
  });
  const [query, setQuery] = useState<SourceQuery>({ page: 1 });
  const result = useQuery({
    queryKey: ['collection-center', 'sources', query],
    queryFn: () => loadSources(token, query),
    placeholderData: keepPreviousData
  });
  const page = result.data ?? { items: [], total: 0, page: query.page, page_size: pageSize };
  const columns = useMemo<DataTableColumn<SourceItem>[]>(
    () => [
      {
        cellClassName: 'max-w-0',
        headerClassName: 'w-[24%]',
        key: 'name',
        header: '名称 / 编码',
        render: (item) => (
          <div className='min-w-0'>
            <OverflowTooltip className='font-semibold' value={item.name} />
            <div className='truncate text-muted-foreground'>{item.code}</div>
          </div>
        )
      },
      { key: 'ownership_type', header: '归属', render: (item) => item.ownership_type },
      { key: 'channel_type', header: '渠道', render: (item) => item.channel_type },
      {
        key: 'enabled',
        header: '启用状态',
        render: (item) => (
          <StatusBadge tone={item.enabled ? 'success' : 'neutral'}>
            {item.enabled ? '已启用' : '已停用'}
          </StatusBadge>
        )
      },
      { key: 'priority', header: '优先级', render: (item) => String(item.priority) },
      {
        key: 'default_source_level',
        header: '默认信源等级',
        render: (item) => item.default_source_level
      },
      { key: 'updated_at', header: '更新时间', render: (item) => formatDateTime(item.updated_at) }
    ],
    []
  );
  const field = (name: keyof typeof form) => ({
    value: form[name],
    onChange: (event: ChangeEvent<HTMLInputElement>) =>
      setForm((current) => ({ ...current, [name]: event.target.value }))
  });
  const select = (name: keyof typeof form) => (value: string) =>
    setForm((current) => ({ ...current, [name]: value === 'all' ? '' : value }));
  const submit = (event: FormEvent) => {
    event.preventDefault();
    setQuery({
      page: 1,
      query: form.query || undefined,
      ownership_type: form.ownershipType || undefined,
      channel_type: form.channelType || undefined,
      enabled: form.enabled || undefined,
      priority: form.priority || undefined,
      default_source_level: form.defaultSourceLevel || undefined,
      updated_from: toRFC3339(form.updatedFrom),
      updated_to: toRFC3339(form.updatedTo)
    });
  };
  return (
    <CollectionPanel>
      <QueryFailure
        error={result.error}
        fetching={result.isFetching}
        retry={() => void result.refetch()}
      />
      <form className={filterGridClass} onSubmit={submit}>
        <Field label='名称 / 编码'>
          <Input aria-label='信源名称或编码' {...field('query')} />
        </Field>
        <Field label='归属'>
          <Select
            ariaLabel='信源归属'
            onValueChange={select('ownershipType')}
            options={[
              { label: '全部', value: 'all' },
              { label: '固定', value: 'fixed' },
              { label: '动态', value: 'dynamic' }
            ]}
            value={form.ownershipType || 'all'}
          />
        </Field>
        <Field label='渠道'>
          <Select
            ariaLabel='信源渠道'
            onValueChange={select('channelType')}
            options={[
              { label: '全部', value: 'all' },
              { label: 'Web Search', value: 'web_search' },
              { label: 'API', value: 'api' },
              { label: 'RSS', value: 'rss' }
            ]}
            value={form.channelType || 'all'}
          />
        </Field>
        <Field label='启用状态'>
          <Select
            ariaLabel='信源启用状态'
            onValueChange={select('enabled')}
            options={[
              { label: '全部', value: 'all' },
              { label: '已启用', value: 'true' },
              { label: '已停用', value: 'false' }
            ]}
            value={form.enabled || 'all'}
          />
        </Field>
        <Field label='优先级'>
          <Select
            ariaLabel='信源优先级'
            onValueChange={select('priority')}
            options={[
              { label: '全部', value: 'all' },
              ...['1', '2', '3', '4', '5'].map((value) => ({ label: value, value }))
            ]}
            value={form.priority || 'all'}
          />
        </Field>
        <Field label='默认信源等级'>
          <Select
            ariaLabel='默认信源等级'
            onValueChange={select('defaultSourceLevel')}
            options={sourceLevels}
            value={form.defaultSourceLevel || 'all'}
          />
        </Field>
        <Field label='更新时间开始'>
          <Input aria-label='信源更新时间开始' type='datetime-local' {...field('updatedFrom')} />
        </Field>
        <Field label='更新时间结束'>
          <Input aria-label='信源更新时间结束' type='datetime-local' {...field('updatedTo')} />
        </Field>
        <Button className='text-xs' size='sm' type='submit'>
          搜索信源
        </Button>
      </form>
      <DataTable
        className='h-full'
        columns={columns}
        emptyText={result.isPending ? '正在加载信源' : '暂无信源'}
        getRowKey={(item) => item.id}
        items={page.items}
        scrollAreaLabel='信源表格滚动区域'
        tableClassName='min-w-[980px] table-fixed text-xs [&_td]:py-2.5 [&_th]:h-9'
      />
      <Pagination
        page={page.page}
        pageSize={page.page_size}
        total={page.total}
        onPageChange={(next) => setQuery((current) => ({ ...current, page: next }))}
      />
    </CollectionPanel>
  );
}
