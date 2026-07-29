import { useCallback, useEffect, useMemo, useState } from 'react';
import { loadAgentStatuses, type AgentStatus } from '../api/agentManagement';
import Button from '../components/ui/Button';
import Card from '../components/ui/Card';
import DataTable, { type DataTableColumn } from '../components/ui/DataTable';
import Icon from '../components/ui/Icon';
import StatusBadge from '../components/ui/StatusBadge';

const refreshIntervalMs = 15_000;

export default function AgentStatusMonitor({ token }: { token: string }) {
  const [items, setItems] = useState<AgentStatus[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');

  const refresh = useCallback(async () => {
    setLoading(true);
    setError('');
    try {
      setItems(await loadAgentStatuses(token));
    } catch (loadError) {
      setError(loadError instanceof Error ? loadError.message : 'Agent 状态加载失败');
    } finally {
      setLoading(false);
    }
  }, [token]);

  useEffect(() => {
    void refresh();
    const timer = window.setInterval(() => void refresh(), refreshIntervalMs);
    return () => window.clearInterval(timer);
  }, [refresh]);

  const columns = useMemo<DataTableColumn<AgentStatus>[]>(
    () => [
      {
        key: 'agent',
        header: 'Agent',
        render: (item) => (
          <div className='agent-status-identity'>
            <strong>{item.display_name}</strong>
            <span>{item.agent_key}</span>
          </div>
        )
      },
      { key: 'version', header: '当前版本', render: (item) => item.current_version },
      {
        key: 'working',
        header: '工作状态',
        render: (item) => (
          <StatusBadge tone={item.is_working ? 'success' : 'neutral'}>
            {item.is_working ? '工作中' : '空闲'}
          </StatusBadge>
        )
      },
      {
        key: 'execution',
        header: '执行状态',
        render: (item) => executionStatusLabel(item.current_execution_status)
      },
      { key: 'updated', header: '更新时间', render: (item) => formatDateTime(item.updated_at) }
    ],
    []
  );

  const workingCount = items.filter((item) => item.is_working).length;

  return (
    <section className='agent-status-monitor'>
      <div className='agent-status-toolbar'>
        <div>
          <span className='eyebrow'>Runtime monitor</span>
          <h2>Agent 运行状态</h2>
          <p>只读展示当前 Agent、版本和执行状态；每 15 秒自动刷新。</p>
        </div>
        <Button disabled={loading} variant='secondary' onClick={() => void refresh()}>
          <Icon name='activity' />
          刷新状态
        </Button>
      </div>

      <div className='agent-status-summary' aria-label='Agent 状态概览'>
        <Card>
          <span>已注册</span>
          <strong>{items.length}</strong>
        </Card>
        <Card>
          <span>工作中</span>
          <strong>{workingCount}</strong>
        </Card>
        <Card>
          <span>空闲</span>
          <strong>{items.length - workingCount}</strong>
        </Card>
      </div>

      {error ? <div className='ui-alert danger'>{error}</div> : null}
      <Card>
        <DataTable
          columns={columns}
          emptyText={loading ? '正在加载 Agent 状态' : '暂无已注册 Agent'}
          getRowKey={(item) => item.agent_key}
          items={items}
        />
      </Card>
    </section>
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
