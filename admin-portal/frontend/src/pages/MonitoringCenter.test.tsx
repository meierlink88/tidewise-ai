import { QueryClientProvider } from '@tanstack/react-query';
import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import {
  loadArtifactMonitoring,
  loadCollectorMonitoring,
  loadMonitoringSummary,
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
    loadSemanticMonitoring: vi.fn()
  };
});

const emptyPage = { items: [], page: 1, page_size: 20, total_items: 0, total_pages: 0 };

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
          raw_results: 18,
          merged_results: 12,
          accepted_artifacts: 10
        }
      ]
    });
    vi.mocked(loadArtifactMonitoring).mockResolvedValue(emptyPage);
    vi.mocked(loadSemanticMonitoring).mockResolvedValue(emptyPage);
  });

  it('shows the three execution kinds separately and preserves raw status evidence', async () => {
    const user = userEvent.setup();
    renderCenter();

    expect(await screen.findByText('collector-execution-1')).toBeInTheDocument();
    expect(screen.getByText('succeeded')).toBeInTheDocument();
    expect(screen.getByText('Raw Results')).toBeInTheDocument();
    expect(screen.getByText('Merged Results')).toBeInTheDocument();
    expect(screen.getByText(/Accepted Artifact 10/)).toBeInTheDocument();

    await user.click(screen.getByRole('tab', { name: 'Event 提取' }));
    expect(await screen.findByText('当前范围暂无执行记录')).toBeInTheDocument();
    expect(loadArtifactMonitoring).toHaveBeenCalledWith('token', '1h', 'all', 1, 20);

    await user.click(screen.getByRole('tab', { name: '事件语义' }));
    await waitFor(() =>
      expect(loadSemanticMonitoring).toHaveBeenCalledWith('token', '1h', 'all', 1, 20)
    );
  });

  it('forwards the selected state and each supported time range', async () => {
    const user = userEvent.setup();
    renderCenter();
    await screen.findByText('collector-execution-1');

    await user.click(screen.getByRole('tab', { name: '失败' }));
    await waitFor(() =>
      expect(loadCollectorMonitoring).toHaveBeenLastCalledWith('token', '1h', 'failure', 1, 20)
    );

    await user.click(screen.getByRole('combobox', { name: '监控时间范围' }));
    await user.click(screen.getByRole('option', { name: '最近 12 小时' }));
    await waitFor(() => expect(loadMonitoringSummary).toHaveBeenLastCalledWith('token', '12h'));
    expect(loadCollectorMonitoring).toHaveBeenLastCalledWith('token', '12h', 'failure', 1, 20);
  });

  it('shows downstream errors and retries both projections', async () => {
    vi.mocked(loadCollectorMonitoring)
      .mockRejectedValueOnce(new Error('AgentRun 暂不可用'))
      .mockResolvedValueOnce(emptyPage);
    const user = userEvent.setup();
    renderCenter();

    expect(await screen.findByRole('alert')).toHaveTextContent('AgentRun 暂不可用');
    await user.click(screen.getByRole('button', { name: '重试' }));

    await waitFor(() => expect(loadCollectorMonitoring).toHaveBeenCalledTimes(2));
    expect(screen.queryByRole('alert')).not.toBeInTheDocument();
  });
});

function renderCenter() {
  return render(
    <QueryClientProvider client={createQueryClient()}>
      <MonitoringCenter token='token' />
    </QueryClientProvider>
  );
}
