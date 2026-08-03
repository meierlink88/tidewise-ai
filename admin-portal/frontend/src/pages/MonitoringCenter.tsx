import { useQuery } from '@tanstack/react-query';
import { Database, RefreshCw } from 'lucide-react';
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
  const kindTotals = summary.data
    ? {
        collector: totalExecutions(summary.data.collector),
        artifact: totalExecutions(summary.data.artifact_extraction),
        semantic: totalExecutions(summary.data.semantic)
      }
    : undefined;
  const selectedCounts = summary.data ? monitoringCountsForKind(summary.data, kind) : undefined;
  const selectDetail = (nextKind: MonitoringKind, nextState: Exclude<MonitoringState, 'all'>) => {
    setKind(nextKind);
    setState(nextState);
    setPage(1);
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
        <SummaryCards onSelectDetail={selectDetail} summary={summary.data} />
      ) : summary.isLoading ? (
        <div className='grid min-h-40 place-items-center text-sm text-muted-foreground'>
          正在加载监控摘要
        </div>
      ) : null}
      <Card className='overflow-hidden py-0'>
        <div className='flex flex-wrap items-center justify-between gap-3 border-b border-border px-5 py-4'>
          <div>
            <h3 className='text-sm font-semibold'>执行明细</h3>
            <p className='mt-1 text-xs text-muted-foreground'>
              {list.data
                ? `${monitoringWindowLabel(window)} · ${monitoringKindLabel(kind)} · ${monitoringStateLabel(state)} · 共 ${list.data.total_items} 条`
                : '三类执行对象分开展示；主状态统一，原始枚举保留。'}
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
                <TabsTrigger aria-label={item.label} key={item.id} value={item.id}>
                  {item.label}
                  {selectedCounts ? (
                    <span
                      aria-hidden='true'
                      className='rounded-full bg-muted-foreground/10 px-1.5 py-0.5 text-[0.65rem] font-semibold leading-none tabular-nums'
                    >
                      {monitoringStateCount(selectedCounts, item.id)}
                    </span>
                  ) : null}
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
          <TabsList className='h-auto w-full justify-start rounded-none border-b border-border bg-transparent px-5 py-0'>
            {kinds.map((item) => (
              <TabsTrigger
                aria-label={item.label}
                className='h-auto flex-none rounded-none border-0 border-b-2 border-transparent px-4 py-3 text-muted-foreground shadow-none data-[state=active]:border-primary data-[state=active]:bg-transparent data-[state=active]:text-primary data-[state=active]:shadow-none'
                key={item.id}
                value={item.id}
              >
                {item.label}
                {kindTotals ? (
                  <span className='rounded-full bg-muted px-1.5 py-0.5 text-[0.65rem] font-semibold leading-none text-muted-foreground tabular-nums'>
                    {kindTotals[item.id]}
                  </span>
                ) : null}
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
          <div className='border-t border-border px-4 pb-4'>
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

function SummaryCards({
  onSelectDetail,
  summary
}: {
  onSelectDetail: (kind: MonitoringKind, state: Exclude<MonitoringState, 'all'>) => void;
  summary: MonitoringSummary;
}) {
  const cards = [
    {
      kind: 'collector' as const,
      title: '事件采集',
      subtitle: '执行对象：每次采集执行',
      counts: summary.collector,
      primaryLabel: 'Accepted Artifact',
      primary: summary.collector_accepted_artifacts,
      firstLabel: 'Raw Results',
      first: summary.collector_raw_results,
      secondLabel: 'Merged Results',
      second: summary.collector_merged_results,
      source: '采集执行与业务结果统计'
    },
    {
      kind: 'artifact' as const,
      title: 'Event 提取',
      subtitle: '执行对象：每个已接收内容',
      counts: summary.artifact_extraction,
      primaryLabel: 'Published',
      primary: summary.artifact_published,
      firstLabel: '正式 Event',
      first: summary.artifact_formal_events,
      secondLabel: 'No Events',
      second: summary.artifact_no_events,
      source: '提取执行、Event 结果与发布记录'
    },
    {
      kind: 'semantic' as const,
      title: '事件语义',
      subtitle: '执行对象：每个 Event 的语义处理',
      counts: summary.semantic,
      primaryLabel: 'Submission',
      primary: summary.semantic_submissions,
      firstLabel: 'Accepted',
      first: summary.semantic_accepted_candidates,
      secondLabel: 'Rejected',
      second: summary.semantic_rejected_candidates,
      source: '语义处理与候选结果统计'
    }
  ];
  return (
    <div className='grid grid-cols-3 gap-4 max-xl:grid-cols-1'>
      {cards.map((card, index) => (
        <Card
          className='gap-0 overflow-hidden py-0 transition-shadow duration-150 hover:shadow-md'
          key={card.title}
        >
          <div className='flex items-start justify-between gap-3 px-4 py-4'>
            <div className='flex min-w-0 items-start gap-3'>
              <span className='flex size-7 shrink-0 items-center justify-center rounded-md bg-primary/10 text-[0.65rem] font-bold text-primary'>
                {String(index + 1).padStart(2, '0')}
              </span>
              <div className='min-w-0'>
                <h3 className='text-sm font-semibold'>{card.title}</h3>
                <p className='mt-0.5 truncate text-[0.7rem] text-muted-foreground'>
                  {card.subtitle}
                </p>
              </div>
            </div>
            <span className='shrink-0 rounded-md border border-border bg-muted/65 px-2 py-1 text-[0.65rem] font-semibold text-muted-foreground tabular-nums'>
              {totalExecutions(card.counts)} 次执行
            </span>
          </div>
          <div className='grid grid-cols-3 border-y border-border bg-muted/20'>
            {(['success', 'running', 'failure'] as const).map((key) => (
              <button
                aria-label={`查看 ${card.title}${monitoringStateLabel(key)}明细，共 ${card.counts[key]} 条`}
                className='border-r border-border px-4 py-3 text-left outline-none transition-colors last:border-r-0 hover:bg-muted/60 focus-visible:ring-2 focus-visible:ring-inset focus-visible:ring-ring'
                key={key}
                onClick={() => onSelectDetail(card.kind, key)}
                type='button'
              >
                <span className='text-xs text-muted-foreground'>{monitoringStateLabel(key)}</span>
                <strong
                  className={`mt-1 block text-xl tabular-nums ${
                    key === 'success'
                      ? 'text-success-foreground'
                      : key === 'running'
                        ? 'text-running-foreground'
                        : 'text-destructive-foreground'
                  }`}
                >
                  {card.counts[key]}
                </strong>
              </button>
            ))}
          </div>
          <div className='grid gap-2.5 px-4 py-4 text-xs'>
            <div className='flex justify-between'>
              <span className='font-medium text-foreground'>成功执行的业务结果</span>
              <strong className='tabular-nums'>
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
            <p className='m-0 flex items-center gap-1.5 border-t border-dashed border-border pt-2.5 text-[0.68rem] text-muted-foreground'>
              <Database aria-hidden='true' className='size-3' />
              <span>数据来源：{card.source}</span>
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

function Identifier({ value }: { value: string }) {
  return (
    <span className='block max-w-56 truncate font-mono text-xs text-foreground' title={value}>
      {value}
    </span>
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
          <TableHead>采集执行 ID</TableHead>
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
            <TableCell>
              <Identifier value={item.execution_id} />
            </TableCell>
            <TableCell>
              <StateCell raw={item.raw_status} state={item.state} />
            </TableCell>
            <TableCell>{item.trigger_source}</TableCell>
            <TableCell>{formatTime(item.started_at)}</TableCell>
            <TableCell>{formatTime(item.completed_at)}</TableCell>
            <TableCell>{formatDuration(item.duration_ms)}</TableCell>
            <TableCell>{item.raw_results}</TableCell>
            <TableCell>{item.merged_results}</TableCell>
            <TableCell>{item.accepted_artifacts}</TableCell>
            <TableCell>
              <Identifier value={item.error_code || '—'} />
            </TableCell>
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
          <TableHead>提取记录 ID</TableHead>
          <TableHead>内容 ID</TableHead>
          <TableHead>采集执行 ID</TableHead>
          <TableHead>状态 / 原始枚举</TableHead>
          <TableHead>更新时间</TableHead>
          <TableHead>耗时</TableHead>
          <TableHead>Event 候选</TableHead>
          <TableHead>发布回执</TableHead>
          <TableHead>错误码</TableHead>
        </TableRow>
      </TableHeader>
      <TableBody>
        {items.map((item) => (
          <TableRow key={item.extraction_key}>
            <TableCell>
              <Identifier value={item.extraction_key} />
            </TableCell>
            <TableCell>
              <Identifier value={item.artifact_id} />
            </TableCell>
            <TableCell>
              <Identifier value={item.collector_execution_id} />
            </TableCell>
            <TableCell>
              <StateCell raw={item.raw_status} state={item.state} />
            </TableCell>
            <TableCell>{formatTime(item.updated_at)}</TableCell>
            <TableCell>{formatDuration(item.duration_ms)}</TableCell>
            <TableCell>{item.event_candidates}</TableCell>
            <TableCell>
              {item.acknowledged_journals} / {item.total_journals}
            </TableCell>
            <TableCell>
              <Identifier value={item.error_code || '—'} />
            </TableCell>
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
          <TableHead>语义处理 ID</TableHead>
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
            <TableCell>
              <Identifier value={item.work_item_id} />
            </TableCell>
            <TableCell>
              <Identifier value={item.event_id} />
            </TableCell>
            <TableCell>{item.trigger_source}</TableCell>
            <TableCell>
              <StateCell raw={item.raw_status} state={item.state} />
            </TableCell>
            <TableCell>{formatTime(item.updated_at)}</TableCell>
            <TableCell>{formatDuration(item.duration_ms)}</TableCell>
            <TableCell>
              {item.attempt_count} / {item.max_attempts}
            </TableCell>
            <TableCell>{item.accepted_candidates}</TableCell>
            <TableCell>{item.rejected_candidates}</TableCell>
            <TableCell>
              <Identifier value={item.error_code || '—'} />
            </TableCell>
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
function totalExecutions(counts: { success: number; running: number; failure: number }) {
  return counts.success + counts.running + counts.failure;
}

function monitoringCountsForKind(summary: MonitoringSummary, kind: MonitoringKind) {
  if (kind === 'artifact') return summary.artifact_extraction;
  if (kind === 'semantic') return summary.semantic;
  return summary.collector;
}

function monitoringStateCount(
  counts: { success: number; running: number; failure: number },
  state: MonitoringState
) {
  return state === 'all' ? totalExecutions(counts) : counts[state];
}

function monitoringWindowLabel(window: MonitoringWindow) {
  return windows.find((item) => item.value === window)?.label ?? window;
}

function monitoringKindLabel(kind: MonitoringKind) {
  return kinds.find((item) => item.id === kind)?.label ?? kind;
}

function monitoringStateLabel(state: MonitoringState) {
  return states.find((item) => item.id === state)?.label ?? state;
}

function formatDuration(durationMs?: number): string {
  if (durationMs === undefined) return '—';
  const seconds = Math.floor(durationMs / 1000);
  if (seconds < 60) return `${seconds}s`;
  const minutes = Math.floor(seconds / 60);
  if (minutes < 60) return `${minutes}m ${seconds % 60}s`;
  return `${Math.floor(minutes / 60)}h ${minutes % 60}m`;
}
