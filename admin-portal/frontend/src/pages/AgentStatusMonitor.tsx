import { useQuery } from '@tanstack/react-query';
import { Activity } from 'lucide-react';
import { loadAgentStatuses, type AgentStatus } from '../api/agentManagement';
import MetricCard from '../components/admin/metric-card';
import QueryError from '../components/admin/query-error';
import { Badge } from '../components/ui/badge';
import { Button } from '../components/ui/Button';
import { Card, CardContent } from '../components/ui/Card';
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow
} from '../components/ui/table';

const refreshIntervalMs = 15_000;

export default function AgentStatusMonitor({ token }: { token: string }) {
  const statusQuery = useQuery({
    queryKey: ['admin', 'agent-status', token],
    queryFn: () => loadAgentStatuses(token),
    refetchInterval: refreshIntervalMs
  });
  const items = statusQuery.data ?? [];

  const workingCount = items.filter((item) => item.is_working).length;
  const errorMessage =
    statusQuery.error instanceof Error ? statusQuery.error.message : 'Agent 状态加载失败';

  return (
    <section className='grid h-full min-w-0 content-start gap-5 overflow-auto pb-6'>
      <div className='flex items-start justify-between gap-4 max-sm:flex-col'>
        <div>
          <span className='text-[0.6875rem] font-semibold uppercase tracking-[0.14em] text-muted-foreground'>
            Runtime monitor
          </span>
          <h2 className='my-1.5 text-2xl font-semibold tracking-tight'>Agent 运行状态</h2>
          <p className='m-0 text-sm text-muted-foreground'>
            只读展示当前 Agent、版本和执行状态；每 15 秒自动刷新。
          </p>
        </div>
        <Button
          className='max-sm:w-full'
          disabled={statusQuery.isFetching}
          variant='outline'
          onClick={() => void statusQuery.refetch()}
        >
          <Activity aria-hidden='true' className='size-4' />
          {statusQuery.isFetching && !statusQuery.isLoading ? '刷新中…' : '刷新状态'}
        </Button>
      </div>

      <div className='grid grid-cols-3 gap-3.5 max-sm:grid-cols-1' aria-label='Agent 状态概览'>
        <MetricCard label='已注册' value={items.length} />
        <MetricCard label='工作中' value={workingCount} />
        <MetricCard label='空闲' value={items.length - workingCount} />
      </div>

      {statusQuery.isError ? (
        <QueryError
          message={errorMessage}
          onRetry={() => void statusQuery.refetch()}
          retrying={statusQuery.isFetching}
        />
      ) : null}
      {!statusQuery.isError || items.length > 0 ? (
        <Card>
          <CardContent className='p-0'>
            {items.length === 0 ? (
              <div className='grid min-h-32 place-items-center px-4 text-sm text-muted-foreground'>
                {statusQuery.isLoading ? '正在加载 Agent 状态' : '暂无已注册 Agent'}
              </div>
            ) : (
              <AgentStatusTable items={items} />
            )}
          </CardContent>
        </Card>
      ) : null}
    </section>
  );
}

function AgentStatusTable({ items }: { items: AgentStatus[] }) {
  return (
    <Table className='min-w-[760px]'>
      <TableHeader>
        <TableRow>
          <TableHead>Agent</TableHead>
          <TableHead>当前版本</TableHead>
          <TableHead>工作状态</TableHead>
          <TableHead>执行状态</TableHead>
          <TableHead>更新时间</TableHead>
        </TableRow>
      </TableHeader>
      <TableBody>
        {items.map((item) => (
          <TableRow key={item.agent_key}>
            <TableCell>
              <div className='grid gap-1'>
                <strong>{item.display_name}</strong>
                <span className='text-xs text-muted-foreground'>{item.agent_key}</span>
              </div>
            </TableCell>
            <TableCell>{item.current_version}</TableCell>
            <TableCell>
              <Badge variant={item.is_working ? 'success' : 'secondary'}>
                {item.is_working ? '工作中' : '空闲'}
              </Badge>
            </TableCell>
            <TableCell>{executionStatusLabel(item.current_execution_status)}</TableCell>
            <TableCell className='whitespace-nowrap'>{formatDateTime(item.updated_at)}</TableCell>
          </TableRow>
        ))}
      </TableBody>
    </Table>
  );
}

function executionStatusLabel(status: string): string {
  const labels: Record<string, string> = {
    idle: '空闲',
    queued: '排队中',
    planning: '规划中',
    collecting: '采集中',
    materializing: '产物处理中',
    running: '运行中',
    succeeded: '已完成',
    failed: '失败',
    cancelled: '已取消'
  };
  return labels[status] ?? status;
}

function formatDateTime(value: string): string {
  const parsed = new Date(value);
  return Number.isNaN(parsed.getTime()) ? '-' : parsed.toLocaleString('zh-CN', { hour12: false });
}
