import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { describe, expect, it, vi } from 'vitest';
import SchedulerSettings from './SchedulerSettings';

describe('SchedulerSettings', () => {
  it('renders interval mode and saves config', async () => {
    const user = userEvent.setup();
    const loadConfig = vi.fn().mockResolvedValue({
      enabled: false,
      mode: 'interval',
      interval_minutes: 60,
      fixed_times: [],
      concurrency: 1,
      batch_size: 10,
      timeout_seconds: 180,
      source_filter: {},
      timezone: 'Asia/Shanghai'
    });
    const saveConfig = vi.fn().mockResolvedValue({});

    render(<SchedulerSettings token="secret" loadConfig={loadConfig} saveConfig={saveConfig} />);

    await screen.findByText('调度器设置');
    await user.click(screen.getByRole('switch'));
    await user.click(screen.getByRole('button', { name: '保存设置' }));

    expect(saveConfig).toHaveBeenCalledWith('secret', expect.objectContaining({
      enabled: true,
      mode: 'interval',
      interval_minutes: 60
    }));
  });

  it('renders fixed time mode with five time inputs', async () => {
    const loadConfig = vi.fn().mockResolvedValue({
      enabled: true,
      mode: 'fixed_times',
      interval_minutes: 0,
      fixed_times: ['09:00', '12:00', '15:00', '18:00', '21:00'],
      concurrency: 1,
      batch_size: 10,
      timeout_seconds: 180,
      source_filter: {},
      timezone: 'Asia/Shanghai'
    });

    render(<SchedulerSettings token="secret" loadConfig={loadConfig} saveConfig={vi.fn()} />);

    expect(await screen.findByDisplayValue('09:00')).toBeInTheDocument();
    expect(screen.getByDisplayValue('21:00')).toBeInTheDocument();
  });
});
