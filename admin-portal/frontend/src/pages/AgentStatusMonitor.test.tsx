import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { loadAgentStatuses } from '../api/agentManagement';
import AgentStatusMonitor from './AgentStatusMonitor';

vi.mock('../api/agentManagement', async () => {
  const actual =
    await vi.importActual<typeof import('../api/agentManagement')>('../api/agentManagement');
  return { ...actual, loadAgentStatuses: vi.fn() };
});

describe('AgentStatusMonitor', () => {
  beforeEach(() => {
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
    render(<AgentStatusMonitor token='token' />);

    expect(await screen.findByText('Collector')).toBeInTheDocument();
    expect(screen.getByText('collector.v1')).toBeInTheDocument();
    expect(screen.getAllByText('空闲').length).toBeGreaterThan(0);
    expect(screen.queryByText(/execution_id/i)).not.toBeInTheDocument();

    await user.click(screen.getByRole('button', { name: '刷新状态' }));
    expect(loadAgentStatuses).toHaveBeenCalledTimes(2);
  });
});
