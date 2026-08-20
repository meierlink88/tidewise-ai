import { keepPreviousData, useQuery } from '@tanstack/react-query';
import { Search } from 'lucide-react';
import { type ChangeEvent, type FormEvent, useMemo, useRef, useState } from 'react';
import {
  loadEvidenceCategories,
  loadEvidences,
  type EvidenceItem,
  type EvidenceQuery
} from '../../api/dataIngestion';
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
import { EvidenceDetailSheet } from './EvidenceDetailSheet';

const emptyQuery: EvidenceQuery = { page: 1 };

export default function EvidenceTab({ token }: { token: string }) {
  const [selectedEvidenceId, setSelectedEvidenceId] = useState<string | null>(null);
  const [detailOpen, setDetailOpen] = useState(false);
  const detailTrigger = useRef<HTMLTableRowElement | null>(null);
  const [form, setForm] = useState({
    title: '',
    summary: '',
    categoryId: '',
    sourceId: '',
    sourceName: '',
    sourceLevel: '',
    isSplit: '',
    publishedFrom: '',
    publishedTo: '',
    collectedFrom: '',
    collectedTo: ''
  });
  const filterFormId = 'evidence-filters';
  const [query, setQuery] = useState<EvidenceQuery>(emptyQuery);
  const result = useQuery({
    queryKey: ['collection-center', 'evidences', query],
    queryFn: () => loadEvidences(token, query),
    placeholderData: keepPreviousData
  });
  const categories = useQuery({
    queryKey: ['collection-center', 'evidence-categories'],
    queryFn: () => loadEvidenceCategories(token)
  });
  const page = result.data ?? { items: [], total: 0, page: query.page, page_size: pageSize };
  const selectedEvidence = page.items.find((item) => item.id === selectedEvidenceId) ?? null;
  const columns = useMemo<DataTableColumn<EvidenceItem>[]>(
    () => [
      {
        cellClassName: 'max-w-0',
        headerClassName: 'w-[18%]',
        key: 'title',
        header: '标题',
        render: (item) => <OverflowTooltip className='font-semibold' value={item.title || '—'} />
      },
      {
        cellClassName: 'max-w-0',
        headerClassName: 'w-[18%]',
        key: 'summary',
        header: '摘要',
        render: (item) => <OverflowTooltip value={item.summary} />
      },
      {
        cellClassName: 'max-w-0',
        headerClassName: 'w-[15%]',
        key: 'categories',
        header: '内容分类',
        render: (item) => (
          <OverflowTooltip
            value={item.categories.map((category) => category.name).join('、') || '-'}
          />
        )
      },
      {
        cellClassName: 'max-w-0',
        headerClassName: 'w-[11%]',
        key: 'source_name',
        header: '信源名称',
        render: (item) => <OverflowTooltip value={item.source_name} />
      },
      {
        cellClassName: 'max-w-0',
        headerClassName: 'w-[16%]',
        key: 'source_id',
        header: '信源 ID',
        render: (item) => <OverflowTooltip className='font-mono' value={item.source_id} />
      },
      { key: 'source_level', header: '信源等级', render: (item) => item.source_level },
      {
        key: 'is_split',
        header: '拆分状态',
        render: (item) => (
          <StatusBadge tone={item.is_split ? 'running' : 'neutral'}>
            {item.is_split ? '已拆分' : '未拆分'}
          </StatusBadge>
        )
      },
      {
        key: 'published_at',
        header: '发布时间',
        render: (item) => (item.published_at ? formatDateTime(item.published_at) : '—')
      },
      {
        key: 'collected_at',
        header: '采集时间',
        render: (item) => formatDateTime(item.collected_at)
      }
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
      title: form.title || undefined,
      summary: form.summary || undefined,
      category_id: form.categoryId || undefined,
      source_id: form.sourceId.trim() || undefined,
      source_name: form.sourceName || undefined,
      source_level: form.sourceLevel || undefined,
      is_split: form.isSplit || undefined,
      published_from: toRFC3339(form.publishedFrom),
      published_to: toRFC3339(form.publishedTo),
      collected_from: toRFC3339(form.collectedFrom),
      collected_to: toRFC3339(form.collectedTo)
    });
  };
  const categoryOptions = [
    { label: '全部', value: 'all' },
    ...(categories.data ?? []).map((category) => ({ label: category.name, value: category.id }))
  ];
  return (
    <CollectionPanel
      actions={
        <Button className='text-xs' form={filterFormId} size='sm' type='submit'>
          <Search aria-hidden='true' className='size-4' />
          搜索证据
        </Button>
      }
      description='以证据为主视角，关联原始资讯展示完整证据内容。'
      title='证据中心'
    >
      <QueryFailure
        error={result.error ?? categories.error}
        fetching={result.isFetching || categories.isFetching}
        retry={() => {
          void result.refetch();
          void categories.refetch();
        }}
      />
      <form
        aria-label='证据筛选条件'
        className={filterGridClass}
        id={filterFormId}
        onSubmit={submit}
      >
        <Field label='标题'>
          <Input aria-label='证据标题' {...field('title')} />
        </Field>
        <Field label='摘要'>
          <Input aria-label='证据摘要' {...field('summary')} />
        </Field>
        <Field label='内容分类'>
          <Select
            ariaLabel='内容分类'
            onValueChange={select('categoryId')}
            options={categoryOptions}
            value={form.categoryId || 'all'}
          />
        </Field>
        <Field label='信源 ID'>
          <Input aria-label='证据信源 ID' {...field('sourceId')} />
        </Field>
        <Field label='信源名称'>
          <Input aria-label='证据信源名称' {...field('sourceName')} />
        </Field>
        <Field label='信源等级'>
          <Select
            ariaLabel='证据信源等级'
            onValueChange={select('sourceLevel')}
            options={sourceLevels}
            value={form.sourceLevel || 'all'}
          />
        </Field>
        <Field label='拆分状态'>
          <Select
            ariaLabel='拆分状态'
            onValueChange={select('isSplit')}
            options={[
              { label: '全部', value: 'all' },
              { label: '已拆分', value: 'true' },
              { label: '未拆分', value: 'false' }
            ]}
            value={form.isSplit || 'all'}
          />
        </Field>
        <Field label='发布时间开始'>
          <Input aria-label='证据发布时间开始' type='datetime-local' {...field('publishedFrom')} />
        </Field>
        <Field label='发布时间结束'>
          <Input aria-label='证据发布时间结束' type='datetime-local' {...field('publishedTo')} />
        </Field>
        <Field label='采集时间开始'>
          <Input aria-label='证据采集时间开始' type='datetime-local' {...field('collectedFrom')} />
        </Field>
        <Field label='采集时间结束'>
          <Input aria-label='证据采集时间结束' type='datetime-local' {...field('collectedTo')} />
        </Field>
      </form>
      <DataTable
        className='h-full'
        columns={columns}
        emptyText={result.isPending ? '正在加载证据' : '暂无证据'}
        getRowKey={(item) => item.id}
        items={page.items}
        onRowActivate={(item, trigger) => {
          detailTrigger.current = trigger;
          setSelectedEvidenceId(item.id);
          setDetailOpen(true);
        }}
        rowAccessibleName={(item) => `查看${item.title ?? '无标题证据'}详情`}
        scrollAreaLabel='证据表格滚动区域'
        selectedRowKey={detailOpen ? (selectedEvidenceId ?? undefined) : undefined}
        tableClassName='min-w-[1480px] table-fixed text-xs [&_td]:py-2.5 [&_th]:h-9'
      />
      <Pagination
        page={page.page}
        pageSize={page.page_size}
        total={page.total}
        onPageChange={(next) => setQuery((current) => ({ ...current, page: next }))}
      />
      <EvidenceDetailSheet
        evidence={selectedEvidence}
        open={detailOpen}
        token={token}
        onOpenChange={(open) => {
          if (open) return;
          setDetailOpen(false);
          window.setTimeout(() => detailTrigger.current?.focus(), 0);
        }}
      />
    </CollectionPanel>
  );
}
