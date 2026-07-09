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

  it('disables saving until admin token is provided', async () => {
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

    render(<SchedulerSettings token="" loadConfig={loadConfig} saveConfig={vi.fn()} />);

    expect(await screen.findByRole('button', { name: '保存设置' })).toBeDisabled();
  });

  it('shows save failure reason', async () => {
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
    const saveConfig = vi.fn().mockRejectedValue(new Error('unauthorized'));

    render(<SchedulerSettings token="wrong-token" loadConfig={loadConfig} saveConfig={saveConfig} />);

    await screen.findByText('调度器设置');
    await user.click(screen.getByRole('button', { name: '保存设置' }));

    expect(await screen.findByText('保存失败：unauthorized')).toBeInTheDocument();
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

    await screen.findByDisplayValue('5');
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

  it('summarizes scheduler execution runs instead of source results', async () => {
    const loadConfig = vi.fn().mockResolvedValue({
      enabled: true,
      mode: 'interval',
      interval_minutes: 5,
      fixed_times: [],
      concurrency: 1,
      batch_size: 10,
      timeout_seconds: 180,
      source_filter: {},
      timezone: 'Asia/Shanghai',
      recent_run: {
        id: 'run-4',
        trigger_type: 'interval',
        status: 'succeeded',
        started_at: '2026-07-09T19:47:11+08:00',
        finished_at: '2026-07-09T19:47:48+08:00',
        total_sources: 2,
        succeeded_sources: 2,
        failed_sources: 0,
        skipped_sources: 0,
        error_summary: ''
      }
    });
    const loadRuns = vi.fn().mockResolvedValue([
      { id: 'run-4', status: 'succeeded', trigger_type: 'interval', started_at: '2026-07-09T19:47:11+08:00', finished_at: '2026-07-09T19:47:48+08:00', total_sources: 2, succeeded_sources: 2, failed_sources: 0, skipped_sources: 0, error_summary: '' },
      { id: 'run-3', status: 'succeeded', trigger_type: 'interval', started_at: '2026-07-09T19:42:11+08:00', finished_at: '2026-07-09T19:42:48+08:00', total_sources: 2, succeeded_sources: 2, failed_sources: 0, skipped_sources: 0, error_summary: '' },
      { id: 'run-2', status: 'failed', trigger_type: 'interval', started_at: '2026-07-09T19:37:11+08:00', finished_at: '2026-07-09T19:37:50+08:00', total_sources: 2, succeeded_sources: 1, failed_sources: 1, skipped_sources: 0, error_summary: 'error' },
      { id: 'run-1', status: 'partial', trigger_type: 'interval', started_at: '2026-07-09T19:31:41+08:00', finished_at: '2026-07-09T19:32:19+08:00', total_sources: 2, succeeded_sources: 1, failed_sources: 1, skipped_sources: 0, error_summary: 'partial' }
    ]);

    render(<SchedulerSettings token="secret" loadConfig={loadConfig} loadRuns={loadRuns} saveConfig={vi.fn()} />);

    expect(await screen.findByText('执行轮次 4')).toBeInTheDocument();
    expect(screen.getByText('成功 2 / 失败 2')).toBeInTheDocument();
    expect(screen.getByText('最近一轮：succeeded')).toBeInTheDocument();
    expect(screen.queryByText('成功 2 / 失败 0')).not.toBeInTheDocument();
  });
});
