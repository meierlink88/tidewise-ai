import { FormEvent, useEffect, useMemo, useState } from 'react';
import {
  loadEvents,
  loadRawDocuments,
  type EventItem,
  type EventQuery,
  type RawDocumentItem,
  type RawDocumentQuery
} from '../api/dataIngestion';
import { Search } from 'lucide-react';
import StatusAlert from '../components/admin/status-alert';
import { Button } from '../components/ui/Button';
import { Card, CardContent } from '../components/ui/Card';
import { DataTable, type DataTableColumn } from '../components/admin/data-table';
import { Field } from '../components/ui/Field';
import { Input } from '../components/ui/Input';
import { Pagination } from '../components/admin/pagination';
import { OverflowTooltip } from '../components/admin/overflow-tooltip';
import { Select } from '../components/ui/Select';
import { StatusBadge } from '../components/ui/StatusBadge';
import { Tabs, TabsContent, TabsList, TabsTrigger } from '../components/ui/Tabs';
import CollectorConfiguration from './CollectorConfiguration';

type ActiveTab = 'raw' | 'events' | 'collector';

const pageSize = 50;

const tabItems: { id: ActiveTab; label: string }[] = [
  { id: 'raw', label: '原始数据' },
  { id: 'events', label: '全球事件' },
  { id: 'collector', label: '采集器配置' }
];

export default function DataIngestionCenter({
  token,
  onOpenMonitoring
}: {
  token: string;
  onOpenMonitoring?: () => void;
}) {
  const [activeTab, setActiveTab] = useState<ActiveTab>('raw');
  const [rawTitle, setRawTitle] = useState('');
  const [rawQuery, setRawQuery] = useState<RawDocumentQuery>({ page: 1, title: '' });
  const [rawPage, setRawPage] = useState({
    items: [] as RawDocumentItem[],
    total: 0,
    page: 1,
    page_size: pageSize
  });
  const [eventTitle, setEventTitle] = useState('');
  const [eventStatus, setEventStatus] = useState('');
  const [factStatus, setFactStatus] = useState('');
  const [eventTimeFrom, setEventTimeFrom] = useState('');
  const [eventTimeTo, setEventTimeTo] = useState('');
  const [firstSeenFrom, setFirstSeenFrom] = useState('');
  const [firstSeenTo, setFirstSeenTo] = useState('');
  const [eventQuery, setEventQuery] = useState<EventQuery>({ page: 1, title: '' });
  const [eventPage, setEventPage] = useState({
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
    loadRawDocuments(token, rawQuery)
      .then((page) => {
        if (active) {
          setRawPage(page);
        }
      })
      .catch((loadError) => {
        if (active) {
          setError(errorText(loadError));
        }
      })
      .finally(() => {
        if (active) {
          setLoading(false);
        }
      });
    return () => {
      active = false;
    };
  }, [rawQuery, token]);

  useEffect(() => {
    if (activeTab !== 'events') {
      return;
    }
    let active = true;
    setLoading(true);
    setError('');
    loadEvents(token, eventQuery)
      .then((page) => {
        if (active) {
          setEventPage(page);
        }
      })
      .catch((loadError) => {
        if (active) {
          setError(errorText(loadError));
        }
      })
      .finally(() => {
        if (active) {
          setLoading(false);
        }
      });
    return () => {
      active = false;
    };
  }, [activeTab, eventQuery, token]);

  const rawColumns = useMemo<DataTableColumn<RawDocumentItem>[]>(
    () => [
      {
        cellClassName: 'max-w-0',
        headerClassName: 'w-[30%]',
        key: 'title',
        header: '标题',
        render: (item) => (
          <OverflowTooltip className='font-semibold' value={item.title || '-'} />
        )
      },
      {
        cellClassName: 'max-w-0',
        headerClassName: 'w-[14%]',
        key: 'source',
        header: '来源',
        render: (item) => <OverflowTooltip value={item.source_name || '-'} />
      },
      {
        cellClassName: 'max-w-0',
        headerClassName: 'w-[28%]',
        key: 'reference',
        header: '证据引用',
        render: (item) => {
          const reference = item.source_ref || item.ingest_channel || '-';
          return <OverflowTooltip className='font-mono text-xs' value={reference} />;
        }
      },
      {
        headerClassName: 'w-[12%]',
        key: 'status',
        header: '状态',
        render: (item) => (
          <StatusBadge tone={statusTone(item.ingest_status)}>{item.ingest_status}</StatusBadge>
        )
      },
      {
        headerClassName: 'w-[16%]',
        key: 'collected',
        header: '采集时间',
        render: (item) => formatDateTime(item.collected_at)
      }
    ],
    []
  );

  const eventColumns = useMemo<DataTableColumn<EventItem>[]>(
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
        key: 'status',
        header: '事件状态',
        render: (item) => (
          <StatusBadge tone={statusTone(item.event_status)}>{item.event_status}</StatusBadge>
        )
      },
      {
        headerClassName: 'w-[12%]',
        key: 'fact',
        header: '事实状态',
        render: (item) => item.fact_status
      },
      {
        headerClassName: 'w-[21%]',
        key: 'event_time',
        header: '事件时间',
        render: (item) => (item.event_time ? formatDateTime(item.event_time) : '-')
      },
      {
        headerClassName: 'w-[21%]',
        key: 'first_seen',
        header: '首次发现',
        render: (item) => formatDateTime(item.first_seen_at)
      }
    ],
    []
  );

  const submitRawSearch = (event: FormEvent) => {
    event.preventDefault();
    setRawQuery({ page: 1, title: rawTitle });
  };

  const submitEventSearch = (event: FormEvent) => {
    event.preventDefault();
    setEventQuery({
      page: 1,
      title: eventTitle,
      event_status: eventStatus || undefined,
      fact_status: factStatus || undefined,
      event_time_from: toRFC3339(eventTimeFrom),
      event_time_to: toRFC3339(eventTimeTo),
      first_seen_from: toRFC3339(firstSeenFrom),
      first_seen_to: toRFC3339(firstSeenTo)
    });
  };

  const retryCurrentData = () => {
    setError('');
    if (activeTab === 'events') {
      setEventQuery((current) => ({ ...current }));
      return;
    }
    setRawQuery((current) => ({ ...current }));
  };

  return (
    <Tabs
      className='grid h-full min-h-0 w-full grid-rows-[auto_auto_minmax(0,1fr)]'
      onValueChange={(value) => isActiveTab(value) && setActiveTab(value)}
      value={activeTab}
    >
      <div className='pb-5'>
        <span className='page-eyebrow'>Data operations</span>
        <h2 className='page-title'>数据采集中心</h2>
        <p className='page-description'>查询原始数据和全球事件，并管理采集 Agent 配置。</p>
      </div>
      <div className='pb-4'>
        <TabsList aria-label='数据采集中心标签'>
          {tabItems.map((item) => (
            <TabsTrigger key={item.id} value={item.id}>
              {item.label}
            </TabsTrigger>
          ))}
        </TabsList>
      </div>
      <div className='flex min-h-0 flex-col gap-4 overflow-hidden'>
        {error && activeTab !== 'collector' ? (
          <StatusAlert
            actionDisabled={loading}
            actionLabel={loading ? '重试中…' : '重试'}
            onAction={retryCurrentData}
            tone='destructive'
          >
            {error}
          </StatusAlert>
        ) : null}

        <TabsContent
          aria-label='全球政经原始数据列表'
          className='min-h-0 flex-1'
          value='raw'
        >
          <Card className='h-full min-h-0 gap-0 overflow-hidden py-0'>
            <CardContent className='grid h-full min-h-0 grid-rows-[auto_minmax(0,1fr)_auto] gap-4 py-5'>
              <form
                className='grid items-end gap-3.5 sm:grid-cols-[minmax(13.75rem,1fr)_auto]'
                onSubmit={submitRawSearch}
              >
                <Field controlId='raw-title-search' label='原始数据标题搜索'>
                  <div className='relative'>
                    <Search
                      aria-hidden='true'
                      className='absolute left-3 top-1/2 size-4 -translate-y-1/2 text-muted-foreground'
                    />
                    <Input
                      aria-label='原始数据标题搜索'
                      className='pl-9'
                      id='raw-title-search'
                      onChange={(event) => setRawTitle(event.target.value)}
                      value={rawTitle}
                    />
                  </div>
                </Field>
                <Button type='submit'>搜索原始数据</Button>
              </form>
              <DataTable
                className='h-full'
                columns={rawColumns}
                emptyText={loading ? '正在加载原始数据' : '暂无原始数据'}
                getRowKey={(item) => item.id}
                items={rawPage.items}
                scrollAreaLabel='原始数据表格滚动区域'
                tableClassName='min-w-[720px] table-fixed'
              />
              <Pagination
                page={rawPage.page}
                pageSize={rawPage.page_size}
                total={rawPage.total}
                onPageChange={(page) => setRawQuery((current) => ({ ...current, page }))}
              />
            </CardContent>
          </Card>
        </TabsContent>

        <TabsContent aria-label='全球事件列表' className='min-h-0 flex-1' value='events'>
          <Card className='h-full min-h-0 gap-0 overflow-hidden py-0'>
            <CardContent className='grid h-full min-h-0 grid-rows-[auto_minmax(0,1fr)_auto] gap-4 py-5'>
              <form
                className='grid items-end gap-3.5 xl:grid-cols-[repeat(auto-fit,minmax(10rem,1fr))]'
                onSubmit={submitEventSearch}
              >
                <Field label='事件标题搜索'>
                  <Input
                    aria-label='事件标题搜索'
                    onChange={(event) => setEventTitle(event.target.value)}
                    value={eventTitle}
                  />
                </Field>
                <Field label='事件状态'>
                  <Select
                    ariaLabel='事件状态'
                    onValueChange={(value) => setEventStatus(value === 'all' ? '' : value)}
                    options={[
                      { label: '全部', value: 'all' },
                      { label: '候选', value: 'candidate' },
                      { label: '已确认', value: 'confirmed' },
                      { label: '已归档', value: 'archived' }
                    ]}
                    value={eventStatus || 'all'}
                  />
                </Field>
                <Field label='事实状态'>
                  <Select
                    ariaLabel='事实状态'
                    onValueChange={(value) => setFactStatus(value === 'all' ? '' : value)}
                    options={[
                      { label: '全部', value: 'all' },
                      { label: '未核验', value: 'unverified' },
                      { label: '已核验', value: 'verified' },
                      { label: '有争议', value: 'disputed' }
                    ]}
                    value={factStatus || 'all'}
                  />
                </Field>
                <Field label='事件时间开始'>
                  <Input
                    aria-label='事件时间开始'
                    onChange={(event) => setEventTimeFrom(event.target.value)}
                    type='datetime-local'
                    value={eventTimeFrom}
                  />
                </Field>
                <Field label='事件时间结束'>
                  <Input
                    aria-label='事件时间结束'
                    onChange={(event) => setEventTimeTo(event.target.value)}
                    type='datetime-local'
                    value={eventTimeTo}
                  />
                </Field>
                <Field label='首次发现开始'>
                  <Input
                    aria-label='首次发现开始'
                    onChange={(event) => setFirstSeenFrom(event.target.value)}
                    type='datetime-local'
                    value={firstSeenFrom}
                  />
                </Field>
                <Field label='首次发现结束'>
                  <Input
                    aria-label='首次发现结束'
                    onChange={(event) => setFirstSeenTo(event.target.value)}
                    type='datetime-local'
                    value={firstSeenTo}
                  />
                </Field>
                <Button type='submit'>搜索事件</Button>
              </form>
              <DataTable
                className='h-full'
                columns={eventColumns}
                emptyText={loading ? '正在加载全球事件' : '暂无全球事件'}
                getRowKey={(item) => item.id}
                items={eventPage.items}
                scrollAreaLabel='全球事件表格滚动区域'
                tableClassName='min-w-[720px] table-fixed'
              />
              <Pagination
                page={eventPage.page}
                pageSize={eventPage.page_size}
                total={eventPage.total}
                onPageChange={(page) => setEventQuery((current) => ({ ...current, page }))}
              />
            </CardContent>
          </Card>
        </TabsContent>

        <TabsContent
          aria-label='采集器配置'
          className='min-h-0 flex-1 overflow-y-auto [scrollbar-gutter:stable]'
          value='collector'
        >
          {activeTab === 'collector' ? (
            <CollectorConfiguration
              onOpenMonitoring={onOpenMonitoring ?? (() => undefined)}
              token={token}
            />
          ) : null}
        </TabsContent>
      </div>
    </Tabs>
  );
}

function errorText(error: unknown): string {
  const message = error instanceof Error ? error.message.trim() : '';
  if (
    !message ||
    /internal server error|admin api returned|request failed with status/i.test(message)
  ) {
    return '数据加载失败，请稍后重试。';
  }
  return message;
}

function formatDateTime(value: string): string {
  return new Date(value).toLocaleString('zh-CN', {
    hour12: false,
    timeZone: 'Asia/Shanghai'
  });
}

function statusTone(status: string): 'success' | 'danger' | 'neutral' {
  if (
    status === 'succeeded' ||
    status === 'active' ||
    status === 'confirmed' ||
    status === 'verified' ||
    status === 'collected'
  ) {
    return 'success';
  }
  if (status === 'failed' || status === 'disabled' || status === 'disputed') {
    return 'danger';
  }
  return 'neutral';
}

function toRFC3339(value: string): string | undefined {
  if (!value) {
    return undefined;
  }
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) {
    return undefined;
  }
  return date.toISOString();
}

function isActiveTab(value: string): value is ActiveTab {
  return tabItems.some((item) => item.id === value);
}
