import { QueryClientProvider } from '@tanstack/react-query';
import { act, render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { loadAgentStatuses } from '../api/agentManagement';
import { createQueryClient } from '../lib/query-client';
import AgentStatusMonitor from './AgentStatusMonitor';

vi.mock('../api/agentManagement', async () => {
  const actual =
    await vi.importActual<typeof import('../api/agentManagement')>('../api/agentManagement');
  return { ...actual, loadAgentStatuses: vi.fn() };
});

describe('AgentStatusMonitor', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    vi.mocked(loadAgentStatuses).mockResolvedValue([
      {
        agent_key: 'collector',
        display_name: 'Collector',
        current_version: 'collector.v1',
        is_working: false,
        current_execution_status: 'idle',
        updated_at: '2026-07-29T08:30:00Z'
      }
    ]);
  });

  it('shows only the frozen status projection and refreshes on demand', async () => {
    const user = userEvent.setup();
    renderStatusMonitor();

    expect(await screen.findByText('Collector')).toBeInTheDocument();
    expect(screen.getByText('collector.v1')).toBeInTheDocument();
    expect(screen.getAllByText('空闲').length).toBeGreaterThan(0);
    expect(screen.queryByText(/execution_id/i)).not.toBeInTheDocument();

    await user.click(screen.getByRole('button', { name: '刷新状态' }));
    expect(loadAgentStatuses).toHaveBeenCalledTimes(2);
  });

  it('shows a visible request error and recovers through retry', async () => {
    vi.mocked(loadAgentStatuses)
      .mockRejectedValueOnce(new Error('Agent 状态服务暂不可用'))
      .mockResolvedValueOnce([
        {
          agent_key: 'collector',
          display_name: 'Collector',
          current_version: 'collector.v1',
          is_working: false,
          current_execution_status: 'idle',
          updated_at: '2026-07-29T08:30:00Z'
        }
      ]);
    const user = userEvent.setup();

    renderStatusMonitor();

    expect(await screen.findByRole('alert')).toHaveTextContent('Agent 状态服务暂不可用');
    await user.click(screen.getByRole('button', { name: '重试' }));

    expect(await screen.findByText('Collector')).toBeInTheDocument();
    expect(screen.queryByRole('alert')).not.toBeInTheDocument();
  });

  it('polls the status projection every 15 seconds', async () => {
    vi.useFakeTimers();
    try {
      renderStatusMonitor();

      await act(async () => {
        await vi.advanceTimersByTimeAsync(0);
      });
      expect(loadAgentStatuses).toHaveBeenCalledTimes(1);

      await act(async () => {
        await vi.advanceTimersByTimeAsync(15_000);
      });
      expect(loadAgentStatuses).toHaveBeenCalledTimes(2);
    } finally {
      vi.useRealTimers();
    }
  });
});

function renderStatusMonitor() {
  const queryClient = createQueryClient();
  return render(
    <QueryClientProvider client={queryClient}>
      <AgentStatusMonitor token='token' />
    </QueryClientProvider>
  );
}
