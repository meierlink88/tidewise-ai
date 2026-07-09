import { FormEvent, useEffect, useMemo, useState } from 'react';
import {
  loadEvents,
  loadRawDocuments,
  loadSourceCatalogs,
  type EventItem,
  type EventQuery,
  type RawDocumentItem,
  type RawDocumentQuery,
  type SourceCatalogItem
} from '../api/dataIngestion';
import Button from '../components/ui/Button';
import Card from '../components/ui/Card';
import DataTable, { type DataTableColumn } from '../components/ui/DataTable';
import Field from '../components/ui/Field';
import Icon from '../components/ui/Icon';
import Input from '../components/ui/Input';
import Pagination from '../components/ui/Pagination';
import Select from '../components/ui/Select';
import StatusBadge from '../components/ui/StatusBadge';
import Tabs, { TabPanel } from '../components/ui/Tabs';
import SchedulerSettings from './SchedulerSettings';

type ActiveTab = 'raw' | 'events' | 'sources' | 'scheduler';

const pageSize = 50;

const tabItems: { id: ActiveTab; label: string }[] = [
  { id: 'raw', label: '原始数据' },
  { id: 'events', label: '全球事件' },
  { id: 'sources', label: '搜索通道' },
  { id: 'scheduler', label: '调度器' }
];

export default function DataIngestionCenter({ token }: { token: string }) {
  const [activeTab, setActiveTab] = useState<ActiveTab>('raw');
  const [rawTitle, setRawTitle] = useState('');
  const [rawQuery, setRawQuery] = useState<RawDocumentQuery>({ page: 1, title: '' });
  const [rawPage, setRawPage] = useState({ items: [] as RawDocumentItem[], total: 0, page: 1, page_size: pageSize });
  const [eventTitle, setEventTitle] = useState('');
  const [eventStatus, setEventStatus] = useState('');
  const [factStatus, setFactStatus] = useState('');
  const [eventTimeFrom, setEventTimeFrom] = useState('');
  const [eventTimeTo, setEventTimeTo] = useState('');
  const [eventQuery, setEventQuery] = useState<EventQuery>({ page: 1, title: '' });
  const [eventPage, setEventPage] = useState({ items: [] as EventItem[], total: 0, page: 1, page_size: pageSize });
  const [sourceStatus, setSourceStatus] = useState('');
  const [sourceItems, setSourceItems] = useState<SourceCatalogItem[]>([]);
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

  useEffect(() => {
    if (activeTab !== 'sources') {
      return;
    }
    let active = true;
    setLoading(true);
    setError('');
    loadSourceCatalogs(token, { status: sourceStatus || undefined })
      .then((result) => {
        if (active) {
          setSourceItems(result.items);
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
  }, [activeTab, sourceStatus, token]);

  const rawColumns = useMemo<DataTableColumn<RawDocumentItem>[]>(() => [
    { key: 'title', header: '标题', render: (item) => <strong>{item.title || '-'}</strong> },
    { key: 'source', header: '来源', render: (item) => item.source_name || '-' },
    { key: 'channel', header: '通道', render: (item) => item.ingest_channel || '-' },
    { key: 'status', header: '状态', render: (item) => <StatusBadge tone={statusTone(item.ingest_status)}>{item.ingest_status}</StatusBadge> },
    { key: 'collected', header: '采集时间', render: (item) => formatDateTime(item.collected_at) }
  ], []);

  const eventColumns = useMemo<DataTableColumn<EventItem>[]>(() => [
    { key: 'title', header: '事件标题', render: (item) => <strong>{item.title}</strong> },
    { key: 'status', header: '事件状态', render: (item) => <StatusBadge tone={statusTone(item.event_status)}>{item.event_status}</StatusBadge> },
    { key: 'fact', header: '事实状态', render: (item) => item.fact_status },
    { key: 'event_time', header: '事件时间', render: (item) => item.event_time ? formatDateTime(item.event_time) : '-' },
    { key: 'first_seen', header: '首次发现', render: (item) => formatDateTime(item.first_seen_at) }
  ], []);

  const sourceColumns = useMemo<DataTableColumn<SourceCatalogItem>[]>(() => [
    { key: 'name', header: '通道名称', render: (item) => <strong>{item.source_name}</strong> },
    { key: 'provider', header: 'Provider', render: (item) => item.provider_key },
    { key: 'channel', header: 'Channel', render: (item) => item.ingest_channel },
    { key: 'type', header: '类型', render: (item) => item.source_type },
    { key: 'status', header: '状态', render: (item) => <StatusBadge tone={statusTone(item.status)}>{item.status}</StatusBadge> }
  ], []);

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
      event_time_to: toRFC3339(eventTimeTo)
    });
  };

  return (
    <section className="data-ingestion-center">
      <div className="data-ingestion-tabs-bar">
        <Tabs active={activeTab} items={tabItems} onChange={setActiveTab} />
      </div>
      <div className="data-ingestion-scroll-area">
        {error ? <div className="ui-alert danger">{error}</div> : null}

        {activeTab === 'raw' ? (
          <TabPanel label="全球政经原始数据列表">
            <Card>
              <form className="toolbar-form" onSubmit={submitRawSearch}>
                <Field label="原始数据标题搜索">
                  <div className="search-input-row">
                    <Icon name="search" />
                    <Input aria-label="原始数据标题搜索" onChange={(event) => setRawTitle(event.target.value)} value={rawTitle} />
                  </div>
                </Field>
                <Button type="submit">搜索原始数据</Button>
              </form>
              <DataTable columns={rawColumns} emptyText={loading ? '正在加载原始数据' : '暂无原始数据'} getRowKey={(item) => item.id} items={rawPage.items} />
              <Pagination page={rawPage.page} pageSize={rawPage.page_size} total={rawPage.total} onPageChange={(page) => setRawQuery((current) => ({ ...current, page }))} />
            </Card>
          </TabPanel>
        ) : null}

        {activeTab === 'events' ? (
          <TabPanel label="全球事件列表">
            <Card>
              <form className="toolbar-form event-filter-form" onSubmit={submitEventSearch}>
                <Field label="事件标题搜索">
                  <Input aria-label="事件标题搜索" onChange={(event) => setEventTitle(event.target.value)} value={eventTitle} />
                </Field>
                <Field label="事件状态">
                  <Select aria-label="事件状态" onChange={(event) => setEventStatus(event.target.value)} value={eventStatus}>
                    <option value="">全部</option>
                    <option value="candidate">候选</option>
                    <option value="confirmed">已确认</option>
                    <option value="archived">已归档</option>
                  </Select>
                </Field>
                <Field label="事实状态">
                  <Select aria-label="事实状态" onChange={(event) => setFactStatus(event.target.value)} value={factStatus}>
                    <option value="">全部</option>
                    <option value="unverified">未核验</option>
                    <option value="verified">已核验</option>
                    <option value="disputed">有争议</option>
                  </Select>
                </Field>
                <Field label="事件时间开始">
                  <Input aria-label="事件时间开始" onChange={(event) => setEventTimeFrom(event.target.value)} type="datetime-local" value={eventTimeFrom} />
                </Field>
                <Field label="事件时间结束">
                  <Input aria-label="事件时间结束" onChange={(event) => setEventTimeTo(event.target.value)} type="datetime-local" value={eventTimeTo} />
                </Field>
                <Button type="submit">搜索事件</Button>
              </form>
              <DataTable columns={eventColumns} emptyText={loading ? '正在加载全球事件' : '暂无全球事件'} getRowKey={(item) => item.id} items={eventPage.items} />
              <Pagination page={eventPage.page} pageSize={eventPage.page_size} total={eventPage.total} onPageChange={(page) => setEventQuery((current) => ({ ...current, page }))} />
            </Card>
          </TabPanel>
        ) : null}

        {activeTab === 'sources' ? (
          <TabPanel label="搜索通道列表">
            <Card>
              <div className="toolbar-form compact-toolbar">
                <Field label="通道状态">
                  <Select aria-label="通道状态" onChange={(event) => setSourceStatus(event.target.value)} value={sourceStatus}>
                    <option value="">全部</option>
                    <option value="active">active</option>
                    <option value="inactive">inactive</option>
                    <option value="disabled">disabled</option>
                  </Select>
                </Field>
              </div>
              <DataTable columns={sourceColumns} emptyText={loading ? '正在加载搜索通道' : '暂无搜索通道'} getRowKey={(item) => item.id} items={sourceItems} />
            </Card>
          </TabPanel>
        ) : null}

        {activeTab === 'scheduler' ? (
          <TabPanel label="调度器">
            <SchedulerSettings token={token} />
          </TabPanel>
        ) : null}
      </div>
    </section>
  );
}

function errorText(error: unknown): string {
  return error instanceof Error ? error.message : '加载失败';
}

function formatDateTime(value: string): string {
  return new Date(value).toLocaleString('zh-CN', {
    hour12: false,
    timeZone: 'Asia/Shanghai'
  });
}

function statusTone(status: string): 'success' | 'danger' | 'neutral' {
  if (status === 'succeeded' || status === 'active' || status === 'confirmed' || status === 'verified' || status === 'collected') {
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
