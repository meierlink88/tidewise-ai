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

  it('hides system source and timezone fields while preserving backend source filter', async () => {
    const user = userEvent.setup();
    const loadConfig = vi.fn().mockResolvedValue({
      enabled: false,
      mode: 'interval',
      interval_minutes: 5,
      fixed_times: [],
      concurrency: 1,
      batch_size: 10,
      timeout_seconds: 180,
      source_filter: {
        provider_key: 'llm_web_research',
        ingest_channel: 'ai_web_research',
        source_type: 'news'
      },
      timezone: 'UTC'
    });
    const saveConfig = vi.fn().mockResolvedValue({});

    render(<SchedulerSettings token="secret" loadConfig={loadConfig} saveConfig={saveConfig} />);

    await screen.findByText('调度器设置');

    expect(screen.queryByLabelText('Provider')).not.toBeInTheDocument();
    expect(screen.queryByLabelText('Channel')).not.toBeInTheDocument();
    expect(screen.queryByLabelText('Source Type')).not.toBeInTheDocument();
    expect(screen.queryByLabelText('时区')).not.toBeInTheDocument();

    await user.click(screen.getByRole('button', { name: '保存设置' }));

    expect(saveConfig).toHaveBeenCalledWith('secret', expect.objectContaining({
      timezone: 'Asia/Shanghai',
      source_filter: {
        provider_key: 'llm_web_research',
        ingest_channel: 'ai_web_research',
        source_type: 'news'
      }
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
