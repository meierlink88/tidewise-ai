import { useQuery } from '@tanstack/react-query';
import { RefreshCw } from 'lucide-react';
import {
  loadArtifactMonitoring,
  loadCollectorMonitoring,
  loadMonitoringSummary,
  loadSemanticMonitoring,
  type ArtifactMonitoringItem,
  type CollectorMonitoringItem,
  type MonitoringKind,
  type MonitoringPage,
  type MonitoringState,
  type MonitoringSummary,
  type MonitoringWindow,
  type SemanticMonitoringItem
} from '../api/agentManagement';
import QueryError from '../components/admin/query-error';
import { Pagination } from '../components/admin/pagination';
import { Button } from '../components/ui/Button';
import { Card, CardContent } from '../components/ui/Card';
import { Select } from '../components/ui/Select';
import { StatusBadge } from '../components/ui/StatusBadge';
import { Tabs, TabsList, TabsTrigger } from '../components/ui/Tabs';
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow
} from '../components/ui/table';
import { useState } from 'react';

const refreshIntervalMs = 15_000;
const pageSize = 20;
const windows = [
  { label: '最近 1 小时', value: '1h' },
  { label: '最近 6 小时', value: '6h' },
  { label: '最近 12 小时', value: '12h' },
  { label: '最近 1 天', value: '24h' }
] satisfies Array<{ label: string; value: MonitoringWindow }>;
const kinds = [
  { id: 'collector', label: '事件采集' },
  { id: 'artifact', label: 'Event 提取' },
  { id: 'semantic', label: '事件语义' }
] satisfies Array<{ id: MonitoringKind; label: string }>;
const states = [
  { id: 'all', label: '全部' },
  { id: 'success', label: '成功' },
  { id: 'running', label: '执行中' },
  { id: 'failure', label: '失败' }
] satisfies Array<{ id: MonitoringState; label: string }>;

export default function MonitoringCenter({ token }: { token: string }) {
  const [window, setWindow] = useState<MonitoringWindow>('1h');
  const [kind, setKind] = useState<MonitoringKind>('collector');
  const [state, setState] = useState<MonitoringState>('all');
  const [page, setPage] = useState(1);
  const summary = useQuery({
    queryKey: ['admin', 'monitoring', 'summary', window],
    queryFn: () => loadMonitoringSummary(token, window),
    refetchInterval: refreshIntervalMs
  });
  const list = useQuery<MonitoringPage<MonitoringItem>>({
    queryKey: ['admin', 'monitoring', 'list', kind, window, state, page],
    queryFn: () => loadMonitoringList(token, kind, window, state, page),
    refetchInterval: refreshIntervalMs
  });
  const error =
    summary.error instanceof Error
      ? summary.error.message
      : list.error instanceof Error
        ? list.error.message
        : '监控数据加载失败';
  const refresh = () => {
    void summary.refetch();
    void list.refetch();
  };
  return (
    <section className='grid h-full min-w-0 content-start gap-5 overflow-auto pb-6'>
      <div className='flex items-start justify-between gap-4 max-lg:flex-col'>
        <div>
          <span className='page-eyebrow'>Pipeline execution monitor</span>
          <h2 className='page-title'>监控中心</h2>
          <p className='page-description'>
            观察事件采集、Event 提取和事件语义的执行结果与业务产出。
          </p>
        </div>
        <div className='flex flex-wrap items-center gap-2 max-sm:w-full'>
          <Select
            ariaLabel='监控时间范围'
            className='w-36'
            onValueChange={(value) => {
              setWindow(value as MonitoringWindow);
              setPage(1);
            }}
            options={windows}
            value={window}
          />
          <span className='text-xs text-muted-foreground'>自动刷新 15s</span>
          <Button
            disabled={summary.isFetching || list.isFetching}
            onClick={refresh}
            variant='outline'
          >
            <RefreshCw className='size-4' />
            {summary.isFetching || list.isFetching ? '刷新中…' : '刷新状态'}
          </Button>
        </div>
      </div>
      {summary.isError || list.isError ? (
        <QueryError
          message={error}
          onRetry={refresh}
          retrying={summary.isFetching || list.isFetching}
        />
      ) : null}
      {summary.data ? (
        <SummaryCards summary={summary.data} />
      ) : summary.isLoading ? (
        <div className='grid min-h-40 place-items-center text-sm text-muted-foreground'>
          正在加载监控摘要
        </div>
      ) : null}
      <Card className='overflow-hidden py-0'>
        <div className='flex flex-wrap items-center justify-between gap-3 border-b px-4 py-3.5'>
          <div>
            <h3 className='text-sm font-semibold'>执行明细</h3>
            <p className='mt-1 text-xs text-muted-foreground'>
              三类执行对象分开展示；主状态统一，原始枚举保留。
            </p>
          </div>
          <Tabs
            onValueChange={(value) => {
              setState(value as MonitoringState);
              setPage(1);
            }}
            value={state}
          >
            <TabsList>
              {states.map((item) => (
                <TabsTrigger key={item.id} value={item.id}>
                  {item.label}
                </TabsTrigger>
              ))}
            </TabsList>
          </Tabs>
        </div>
        <Tabs
          onValueChange={(value) => {
            setKind(value as MonitoringKind);
            setPage(1);
          }}
          value={kind}
        >
          <TabsList className='h-auto w-full justify-start rounded-none border-b bg-transparent px-4 py-0'>
            {kinds.map((item) => (
              <TabsTrigger
                className='rounded-none border-0 border-b-2 border-transparent px-4 py-3 shadow-none data-[state=active]:border-foreground data-[state=active]:shadow-none'
                key={item.id}
                value={item.id}
              >
                {item.label}
              </TabsTrigger>
            ))}
          </TabsList>
        </Tabs>
        <CardContent className='p-0'>
          {list.isLoading ? (
            <Empty text='正在加载执行明细' />
          ) : list.isError && !list.data ? (
            <Empty text='执行明细加载失败' />
          ) : list.data?.items.length === 0 ? (
            <Empty text='当前范围暂无执行记录' />
          ) : (
            <MonitoringTable kind={kind} items={list.data?.items ?? []} />
          )}
        </CardContent>
        {list.data && list.data.total_items > 0 ? (
          <div className='border-t px-4 pb-4'>
            <Pagination
              onPageChange={setPage}
              page={list.data.page}
              pageSize={list.data.page_size}
              total={list.data.total_items}
            />
          </div>
        ) : null}
      </Card>
    </section>
  );
}

type MonitoringItem = CollectorMonitoringItem | ArtifactMonitoringItem | SemanticMonitoringItem;

async function loadMonitoringList(
  token: string,
  kind: MonitoringKind,
  window: MonitoringWindow,
  state: MonitoringState,
  page: number
): Promise<MonitoringPage<MonitoringItem>> {
  if (kind === 'collector') return loadCollectorMonitoring(token, window, state, page, pageSize);
  if (kind === 'artifact') return loadArtifactMonitoring(token, window, state, page, pageSize);
  return loadSemanticMonitoring(token, window, state, page, pageSize);
}

function SummaryCards({ summary }: { summary: MonitoringSummary }) {
  const cards = [
    {
      title: '事件采集',
      subtitle: '执行对象：Collector Execution',
      counts: summary.collector,
      primaryLabel: 'Accepted Artifact',
      primary: summary.collector_accepted_artifacts,
      firstLabel: 'Raw Results',
      first: summary.collector_raw_results,
      secondLabel: 'Merged Results',
      second: summary.collector_merged_results,
      source: 'executions + candidate_counts'
    },
    {
      title: 'Event 提取',
      subtitle: '执行对象：Accepted Artifact',
      counts: summary.artifact_extraction,
      primaryLabel: 'Published',
      primary: summary.artifact_published,
      firstLabel: '正式 Event',
      first: summary.artifact_formal_events,
      secondLabel: 'No Events',
      second: summary.artifact_no_events,
      source: 'extraction_units + extraction_result + journal'
    },
    {
      title: '事件语义',
      subtitle: '执行对象：Semantic Work Item / Event',
      counts: summary.semantic,
      primaryLabel: 'Submission',
      primary: summary.semantic_submissions,
      firstLabel: 'Accepted',
      first: summary.semantic_accepted_candidates,
      secondLabel: 'Rejected',
      second: summary.semantic_rejected_candidates,
      source: 'work_items + executions'
    }
  ];
  return (
    <div className='grid grid-cols-3 gap-3.5 max-xl:grid-cols-1'>
      {cards.map((card, index) => (
        <Card className='gap-0 overflow-hidden py-0' key={card.title}>
          <div className='flex items-start gap-3 px-4 py-4'>
            <span className='flex size-7 items-center justify-center rounded-md bg-muted text-[0.65rem] font-bold'>
              {String(index + 1).padStart(2, '0')}
            </span>
            <div>
              <h3 className='text-sm font-semibold'>{card.title}</h3>
              <p className='mt-0.5 font-mono text-[0.65rem] text-muted-foreground'>
                {card.subtitle}
              </p>
            </div>
          </div>
          <div className='grid grid-cols-3 border-y'>
            {(['success', 'running', 'failure'] as const).map((key) => (
              <div className='border-r px-4 py-3 last:border-r-0' key={key}>
                <span className='text-xs text-muted-foreground'>
                  {key === 'success' ? '成功' : key === 'running' ? '执行中' : '失败'}
                </span>
                <strong className='mt-1 block text-xl tabular-nums'>{card.counts[key]}</strong>
              </div>
            ))}
          </div>
          <div className='grid gap-2 px-4 py-4 text-xs'>
            <div className='flex justify-between'>
              <span className='font-medium text-foreground'>成功执行的业务结果</span>
              <strong>
                {card.primaryLabel} {card.primary}
              </strong>
            </div>
            <div className='flex justify-between'>
              <span className='text-muted-foreground'>{card.firstLabel}</span>
              <strong>{card.first}</strong>
            </div>
            <div className='flex justify-between'>
              <span className='text-muted-foreground'>{card.secondLabel}</span>
              <strong>{card.second}</strong>
            </div>
            <p className='m-0 border-t pt-2 font-mono text-[0.65rem] text-muted-foreground'>
              数据来源：{card.source}
            </p>
          </div>
        </Card>
      ))}
    </div>
  );
}
function Empty({ text }: { text: string }) {
  return (
    <div className='grid min-h-32 place-items-center text-sm text-muted-foreground'>{text}</div>
  );
}
function StateCell({ state, raw }: { state: string; raw: string }) {
  return (
    <div className='grid justify-items-start gap-1'>
      <StatusBadge
        tone={state === 'success' ? 'success' : state === 'failure' ? 'danger' : 'running'}
      >
        {state === 'success' ? '成功' : state === 'failure' ? '失败' : '执行中'}
      </StatusBadge>
      <span className='font-mono text-[0.68rem] text-muted-foreground'>{raw}</span>
    </div>
  );
}
function MonitoringTable({ kind, items }: { kind: MonitoringKind; items: MonitoringItem[] }) {
  if (kind === 'collector') return <CollectorTable items={items as CollectorMonitoringItem[]} />;
  if (kind === 'artifact') return <ArtifactTable items={items as ArtifactMonitoringItem[]} />;
  return <SemanticTable items={items as SemanticMonitoringItem[]} />;
}
function CollectorTable({ items }: { items: CollectorMonitoringItem[] }) {
  return (
    <Table className='min-w-[920px]'>
      <TableHeader>
        <TableRow>
          <TableHead>Execution ID</TableHead>
          <TableHead>状态 / 原始枚举</TableHead>
          <TableHead>触发来源</TableHead>
          <TableHead>开始</TableHead>
          <TableHead>完成</TableHead>
          <TableHead>耗时</TableHead>
          <TableHead>Raw</TableHead>
          <TableHead>Merged</TableHead>
          <TableHead>Accepted Artifact</TableHead>
          <TableHead>错误码</TableHead>
        </TableRow>
      </TableHeader>
      <TableBody>
        {items.map((item) => (
          <TableRow key={item.execution_id}>
            <TableCell className='font-mono text-xs'>{item.execution_id}</TableCell>
            <TableCell>
              <StateCell raw={item.raw_status} state={item.state} />
            </TableCell>
            <TableCell>{item.trigger_source}</TableCell>
            <TableCell>{formatTime(item.started_at)}</TableCell>
            <TableCell>{formatTime(item.completed_at)}</TableCell>
            <TableCell>{formatDuration(item.started_at, item.completed_at)}</TableCell>
            <TableCell>{item.raw_results}</TableCell>
            <TableCell>{item.merged_results}</TableCell>
            <TableCell>{item.accepted_artifacts}</TableCell>
            <TableCell className='font-mono text-xs'>{item.error_code || '—'}</TableCell>
          </TableRow>
        ))}
      </TableBody>
    </Table>
  );
}
function ArtifactTable({ items }: { items: ArtifactMonitoringItem[] }) {
  return (
    <Table className='min-w-[980px]'>
      <TableHeader>
        <TableRow>
          <TableHead>Extraction Key</TableHead>
          <TableHead>Artifact ID</TableHead>
          <TableHead>Collector Execution</TableHead>
          <TableHead>状态 / 原始枚举</TableHead>
          <TableHead>更新时间</TableHead>
          <TableHead>耗时</TableHead>
          <TableHead>Event Candidate</TableHead>
          <TableHead>Journal 回执</TableHead>
          <TableHead>错误码</TableHead>
        </TableRow>
      </TableHeader>
      <TableBody>
        {items.map((item) => (
          <TableRow key={item.extraction_key}>
            <TableCell className='font-mono text-xs'>{short(item.extraction_key)}</TableCell>
            <TableCell>{item.artifact_id}</TableCell>
            <TableCell className='font-mono text-xs'>
              {short(item.collector_execution_id)}
            </TableCell>
            <TableCell>
              <StateCell raw={item.raw_status} state={item.state} />
            </TableCell>
            <TableCell>{formatTime(item.updated_at)}</TableCell>
            <TableCell>{formatDuration(item.started_at, item.completed_at)}</TableCell>
            <TableCell>{item.event_candidates}</TableCell>
            <TableCell>
              {item.acknowledged_journals} / {item.total_journals}
            </TableCell>
            <TableCell className='font-mono text-xs'>{item.error_code || '—'}</TableCell>
          </TableRow>
        ))}
      </TableBody>
    </Table>
  );
}
function SemanticTable({ items }: { items: SemanticMonitoringItem[] }) {
  return (
    <Table className='min-w-[980px]'>
      <TableHeader>
        <TableRow>
          <TableHead>Work Item ID</TableHead>
          <TableHead>Event ID</TableHead>
          <TableHead>触发来源</TableHead>
          <TableHead>状态 / 原始枚举</TableHead>
          <TableHead>更新时间</TableHead>
          <TableHead>耗时</TableHead>
          <TableHead>尝试</TableHead>
          <TableHead>Accepted</TableHead>
          <TableHead>Rejected</TableHead>
          <TableHead>错误码</TableHead>
        </TableRow>
      </TableHeader>
      <TableBody>
        {items.map((item) => (
          <TableRow key={item.work_item_id}>
            <TableCell className='font-mono text-xs'>{short(item.work_item_id)}</TableCell>
            <TableCell className='font-mono text-xs'>{short(item.event_id)}</TableCell>
            <TableCell>{item.trigger_source}</TableCell>
            <TableCell>
              <StateCell raw={item.raw_status} state={item.state} />
            </TableCell>
            <TableCell>{formatTime(item.updated_at)}</TableCell>
            <TableCell>{formatDuration(item.started_at, item.completed_at)}</TableCell>
            <TableCell>
              {item.attempt_count} / {item.max_attempts}
            </TableCell>
            <TableCell>{item.accepted_candidates}</TableCell>
            <TableCell>{item.rejected_candidates}</TableCell>
            <TableCell className='font-mono text-xs'>{item.error_code || '—'}</TableCell>
          </TableRow>
        ))}
      </TableBody>
    </Table>
  );
}
function formatTime(value?: string) {
  return value
    ? new Date(value).toLocaleString('zh-CN', { hour12: false, timeZone: 'Asia/Shanghai' })
    : '—';
}
function short(value: string) {
  return value.length > 14 ? `${value.slice(0, 8)}…${value.slice(-4)}` : value;
}

function formatDuration(startedAt?: string, completedAt?: string): string {
  if (!startedAt) return '—';
  const started = new Date(startedAt).getTime();
  const completed = completedAt ? new Date(completedAt).getTime() : Date.now();
  if (!Number.isFinite(started) || !Number.isFinite(completed) || completed < started) return '—';
  const seconds = Math.floor((completed - started) / 1000);
  if (seconds < 60) return `${seconds}s`;
  const minutes = Math.floor(seconds / 60);
  if (minutes < 60) return `${minutes}m ${seconds % 60}s`;
  return `${Math.floor(minutes / 60)}h ${minutes % 60}m`;
}
