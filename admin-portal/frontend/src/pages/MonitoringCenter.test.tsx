import { QueryClientProvider } from '@tanstack/react-query';
import { render, screen, waitFor, within } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import {
  loadArtifactMonitoring,
  loadAgentStatuses,
  loadCollectorMonitoring,
  loadMonitoringSummary,
  loadRuntimeHealth,
  loadSemanticMonitoring
} from '../api/agentManagement';
import { createQueryClient } from '../lib/query-client';
import MonitoringCenter from './MonitoringCenter';

vi.mock('../api/agentManagement', async () => {
  const actual =
    await vi.importActual<typeof import('../api/agentManagement')>('../api/agentManagement');
  return {
    ...actual,
    loadMonitoringSummary: vi.fn(),
    loadCollectorMonitoring: vi.fn(),
    loadArtifactMonitoring: vi.fn(),
    loadSemanticMonitoring: vi.fn(),
    loadAgentStatuses: vi.fn(),
    loadRuntimeHealth: vi.fn()
  };
});

const emptyPage = {
  items: [],
  window: '1h' as const,
  generated_at: '2026-08-03T08:30:00Z',
  page: 1,
  page_size: 20,
  total_items: 0,
  total_pages: 0
};

describe('MonitoringCenter', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    vi.mocked(loadMonitoringSummary).mockResolvedValue({
      window: '1h',
      generated_at: '2026-08-03T08:30:00Z',
      collector: { success: 3, running: 1, failure: 1 },
      artifact_extraction: { success: 2, running: 1, failure: 0 },
      semantic: { success: 2, running: 0, failure: 1 },
      collector_raw_results: 18,
      collector_merged_results: 12,
      collector_accepted_artifacts: 10,
      artifact_published: 2,
      artifact_no_events: 1,
      artifact_formal_events: 6,
      semantic_submissions: 2,
      semantic_accepted_candidates: 5,
      semantic_rejected_candidates: 1
    });
    vi.mocked(loadCollectorMonitoring).mockResolvedValue({
      ...emptyPage,
      total_items: 1,
      total_pages: 1,
      items: [
        {
          execution_id: 'collector-execution-1',
          state: 'success',
          raw_status: 'succeeded',
          trigger_source: 'schedule',
          started_at: '2026-08-03T08:00:00Z',
          completed_at: '2026-08-03T08:01:00Z',
          duration_ms: 60_000,
          raw_results: 18,
          merged_results: 12,
          accepted_artifacts: 10
        }
      ]
    });
    vi.mocked(loadArtifactMonitoring).mockResolvedValue(emptyPage);
    vi.mocked(loadSemanticMonitoring).mockResolvedValue(emptyPage);
    vi.mocked(loadAgentStatuses).mockResolvedValue([
      {
        agent_key: 'collector',
        display_name: '综合采集 Agent',
        current_version: 'collector.v1',
        is_working: false,
        current_execution_status: 'idle',
        updated_at: '2026-08-03T08:30:00Z'
      }
    ]);
    vi.mocked(loadRuntimeHealth).mockResolvedValue(runtimeHealth());
  });

  it('shows a one-screen overview and opens independent execution detail', async () => {
    const user = userEvent.setup();
    renderCenter();

    expect(await screen.findByText('Data Service')).toBeInTheDocument();
    expect(await screen.findAllByText('Ready')).toHaveLength(2);
    expect(screen.getByText('Green')).toBeInTheDocument();
    expect(await screen.findByText('综合采集 Agent')).toBeInTheDocument();
    expect(screen.getByText(/idle ·/)).toBeInTheDocument();
    expect(screen.getByText('Raw Results')).toBeInTheDocument();
    expect(screen.getByText('Merged Results')).toBeInTheDocument();
    expect(screen.getByText(/Accepted Artifact 10/)).toBeInTheDocument();
    expect(loadCollectorMonitoring).not.toHaveBeenCalled();

    await user.click(screen.getByRole('button', { name: '查看事件采集执行明细' }));
    expect(await screen.findByText('collector-execution-1')).toBeInTheDocument();
    expect(screen.getByText('succeeded')).toBeInTheDocument();
    expect(screen.getByRole('heading', { name: '事件采集执行明细' })).toBeInTheDocument();
    expect(screen.getByRole('navigation', { name: '面包屑' })).toHaveTextContent(
      '监控中心 / 事件采集执行明细'
    );
    const detailTableRegion = screen.getByRole('region', {
      name: '事件采集执行明细表格滚动区域'
    });
    expect(within(detailTableRegion).getByRole('table')).toBeInTheDocument();
    expect(
      within(detailTableRegion).getByRole('columnheader', { name: '采集执行 ID' })
    ).toBeVisible();
    expect(
      within(detailTableRegion).queryByRole('button', { name: '下一页' })
    ).not.toBeInTheDocument();
    expect(screen.getByRole('button', { name: '下一页' })).toBeInTheDocument();

    await user.click(screen.getByRole('tab', { name: /^Event 提取/ }));
    expect(await screen.findByText('当前范围暂无执行记录')).toBeInTheDocument();
    expect(loadArtifactMonitoring).toHaveBeenCalledWith('token', '1h', 'all', 1, 20);

    await user.click(screen.getByRole('tab', { name: /^事件语义/ }));
    await waitFor(() =>
      expect(loadSemanticMonitoring).toHaveBeenCalledWith('token', '1h', 'all', 1, 20)
    );
  });

  it('forwards every supported state and time range', async () => {
    const user = userEvent.setup();
    renderCenter();
    await user.click(await screen.findByRole('button', { name: '查看事件采集执行明细' }));
    await screen.findByText('collector-execution-1');

    for (const [label, state] of [
      ['成功', 'success'],
      ['执行中', 'running'],
      ['失败', 'failure'],
      ['全部', 'all']
    ] as const) {
      await user.click(screen.getByRole('tab', { name: label }));
      await waitFor(() =>
        expect(loadCollectorMonitoring).toHaveBeenLastCalledWith('token', '1h', state, 1, 20)
      );
    }

    for (const [label, window] of [
      ['最近 6 小时', '6h'],
      ['最近 12 小时', '12h'],
      ['最近 1 天', '24h'],
      ['最近 1 小时', '1h']
    ] as const) {
      await user.click(screen.getByRole('combobox', { name: '监控时间范围' }));
      await user.click(screen.getByRole('option', { name: label }));
      await waitFor(() => expect(loadMonitoringSummary).toHaveBeenLastCalledWith('token', window));
      expect(loadCollectorMonitoring).toHaveBeenLastCalledWith('token', window, 'all', 1, 20);
    }
  });

  it('opens card status totals as the matching detail slice in the same time window', async () => {
    vi.mocked(loadArtifactMonitoring).mockResolvedValue({
      ...emptyPage,
      total_items: 2,
      total_pages: 1,
      items: [
        {
          extraction_key: 'artifact-extraction-1',
          artifact_id: 'artifact-1',
          collector_execution_id: 'collector-execution-1',
          state: 'success',
          raw_status: 'published',
          updated_at: '2026-08-03T08:10:00Z',
          started_at: '2026-08-03T08:09:00Z',
          completed_at: '2026-08-03T08:10:00Z',
          duration_ms: 60_000,
          event_candidates: 2,
          acknowledged_journals: 1,
          total_journals: 1
        }
      ]
    });
    const user = userEvent.setup();
    renderCenter();
    await screen.findByRole('button', { name: '查看Event 提取执行明细' });

    await user.click(screen.getByRole('button', { name: '查看 Event 提取成功明细，共 2 条' }));

    await waitFor(() =>
      expect(loadArtifactMonitoring).toHaveBeenLastCalledWith('token', '1h', 'success', 1, 20)
    );
    expect(await screen.findByText('artifact-extraction-1')).toBeInTheDocument();
    expect(screen.getByText('最近 1 小时 · Event 提取 · 成功 · 共 2 条')).toBeInTheDocument();

    await user.click(screen.getByRole('combobox', { name: '监控时间范围' }));
    await user.click(screen.getByRole('option', { name: '最近 6 小时' }));

    await waitFor(() =>
      expect(loadArtifactMonitoring).toHaveBeenLastCalledWith('token', '6h', 'success', 1, 20)
    );
    expect(screen.getByText('最近 6 小时 · Event 提取 · 成功 · 共 2 条')).toBeInTheDocument();

    await user.click(screen.getByRole('button', { name: '返回监控中心' }));
    await user.click(screen.getByRole('button', { name: '查看Event 提取执行明细' }));
    await waitFor(() =>
      expect(loadArtifactMonitoring).toHaveBeenLastCalledWith('token', '6h', 'all', 1, 20)
    );
  });

  it('shows loading feedback and manually refreshes both projections', async () => {
    let resolveSummary!: (value: Awaited<ReturnType<typeof loadMonitoringSummary>>) => void;
    vi.mocked(loadMonitoringSummary).mockReturnValueOnce(
      new Promise((resolve) => {
        resolveSummary = resolve;
      })
    );
    const user = userEvent.setup();
    renderCenter();

    expect(screen.getByText('正在加载监控摘要')).toBeInTheDocument();
    resolveSummary({
      window: '1h',
      generated_at: '2026-08-03T08:30:00Z',
      collector: { success: 0, running: 0, failure: 0 },
      artifact_extraction: { success: 0, running: 0, failure: 0 },
      semantic: { success: 0, running: 0, failure: 0 },
      collector_raw_results: 0,
      collector_merged_results: 0,
      collector_accepted_artifacts: 0,
      artifact_published: 0,
      artifact_no_events: 0,
      artifact_formal_events: 0,
      semantic_submissions: 0,
      semantic_accepted_candidates: 0,
      semantic_rejected_candidates: 0
    });
    expect(await screen.findAllByText('成功执行的业务结果')).toHaveLength(3);

    await user.click(screen.getByRole('button', { name: '刷新状态' }));
    await waitFor(() => expect(loadMonitoringSummary).toHaveBeenCalledTimes(2));
    expect(loadRuntimeHealth).toHaveBeenCalledTimes(2);
    expect(loadAgentStatuses).toHaveBeenCalledTimes(2);
  });

  it('shows detail errors and retries the selected projection', async () => {
    vi.mocked(loadCollectorMonitoring)
      .mockRejectedValueOnce(new Error('AgentRun 暂不可用'))
      .mockResolvedValueOnce(emptyPage);
    const user = userEvent.setup();
    renderCenter();
    await user.click(await screen.findByRole('button', { name: '查看事件采集执行明细' }));

    expect(await screen.findByRole('alert')).toHaveTextContent('AgentRun 暂不可用');
    await user.click(screen.getByRole('button', { name: '重试' }));

    await waitFor(() => expect(loadCollectorMonitoring).toHaveBeenCalledTimes(2));
    expect(screen.queryByRole('alert')).not.toBeInTheDocument();
  });

  it('keeps partial runtime failures visible without hiding monitoring summaries', async () => {
    vi.mocked(loadRuntimeHealth).mockResolvedValue({
      ...runtimeHealth(),
      status: 'degraded',
      services: runtimeHealth().services.map((service) =>
		service.key === 'qdrant'
          ? {
              ...service,
              status: 'down' as const,
              latency_ms: 17,
              reason_code: 'unreachable' as const
            }
          : service
      )
    });

    renderCenter();

    expect(await screen.findByText('无法连接')).toBeInTheDocument();
    expect(screen.getByText('Raw Results')).toBeInTheDocument();
    expect(screen.getByRole('button', { name: '查看事件采集执行明细' })).toBeInTheDocument();
  });
});

function runtimeHealth() {
  return {
    status: 'ready' as const,
    checked_at: '2026-08-03T08:30:00Z',
    services: [
      {
        key: 'data' as const,
        display_name: 'Data Service' as const,
        status: 'ready' as const,
        checked_at: '2026-08-03T08:30:00Z'
      },
      {
        key: 'agentrun' as const,
        display_name: 'AgentRun' as const,
        status: 'ready' as const,
        checked_at: '2026-08-03T08:30:00Z'
      },
      {
        key: 'qdrant' as const,
        display_name: 'Qdrant' as const,
        status: 'ready' as const,
        checked_at: '2026-08-03T08:30:00Z'
		}
    ]
  };
}

function renderCenter() {
  return render(
    <QueryClientProvider client={createQueryClient()}>
      <MonitoringCenter token='token' />
    </QueryClientProvider>
  );
}
