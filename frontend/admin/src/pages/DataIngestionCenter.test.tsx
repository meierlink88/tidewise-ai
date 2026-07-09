import { render, screen, within } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { describe, expect, it, vi } from 'vitest';
import * as dataIngestionAPI from '../api/dataIngestion';
import * as schedulerAPI from '../api/scheduler';
import DataIngestionCenter from './DataIngestionCenter';

describe('DataIngestionCenter', () => {
  it('renders four tabs and loads raw documents by default', async () => {
    vi.spyOn(dataIngestionAPI, 'loadRawDocuments').mockResolvedValue({
      items: [
        {
          id: 'raw-1',
          source_name: '新华社',
          title: '央行公布金融数据',
          content_text: '摘要',
          collected_at: '2026-07-09T10:00:00Z',
          ingest_status: 'collected'
        }
      ],
      total: 1,
      page: 1,
      page_size: 50
    });
    vi.spyOn(dataIngestionAPI, 'loadEvents').mockResolvedValue({ items: [], total: 0, page: 1, page_size: 50 });
    vi.spyOn(dataIngestionAPI, 'loadSourceCatalogs').mockResolvedValue({ items: [] });
    vi.spyOn(schedulerAPI, 'loadSchedulerConfig').mockResolvedValue(defaultSchedulerConfig());
    vi.spyOn(schedulerAPI, 'loadSchedulerRuns').mockResolvedValue([]);
    vi.spyOn(schedulerAPI, 'saveSchedulerConfig').mockResolvedValue(defaultSchedulerConfig());

    render(<DataIngestionCenter token="secret-token" />);

    expect(await screen.findByRole('tab', { name: '原始数据' })).toBeInTheDocument();
    expect(screen.getByRole('tab', { name: '全球事件' })).toBeInTheDocument();
    expect(screen.getByRole('tab', { name: '搜索通道' })).toBeInTheDocument();
    expect(screen.getByRole('tab', { name: '调度器' })).toBeInTheDocument();
    expect(await screen.findByText('央行公布金融数据')).toBeInTheDocument();
    expect(dataIngestionAPI.loadRawDocuments).toHaveBeenCalledWith('secret-token', { page: 1, title: '' });
  });

  it('applies raw title search and event filters', async () => {
    const user = userEvent.setup();
    vi.spyOn(dataIngestionAPI, 'loadRawDocuments').mockResolvedValue({ items: [], total: 0, page: 1, page_size: 50 });
    vi.spyOn(dataIngestionAPI, 'loadEvents').mockResolvedValue({ items: [], total: 0, page: 1, page_size: 50 });
    vi.spyOn(dataIngestionAPI, 'loadSourceCatalogs').mockResolvedValue({ items: [] });
    vi.spyOn(schedulerAPI, 'loadSchedulerConfig').mockResolvedValue(defaultSchedulerConfig());
    vi.spyOn(schedulerAPI, 'loadSchedulerRuns').mockResolvedValue([]);
    vi.spyOn(schedulerAPI, 'saveSchedulerConfig').mockResolvedValue(defaultSchedulerConfig());

    render(<DataIngestionCenter token="secret-token" />);

    await screen.findByRole('tab', { name: '原始数据' });
    await user.type(screen.getByLabelText('原始数据标题搜索'), '央行');
    await user.click(screen.getByRole('button', { name: '搜索原始数据' }));

    expect(dataIngestionAPI.loadRawDocuments).toHaveBeenLastCalledWith('secret-token', { page: 1, title: '央行' });

    await user.click(screen.getByRole('tab', { name: '全球事件' }));
    await user.type(screen.getByLabelText('事件标题搜索'), '美联储');
    await user.selectOptions(screen.getByLabelText('事件状态'), 'confirmed');
    await user.selectOptions(screen.getByLabelText('事实状态'), 'verified');
    await user.type(screen.getByLabelText('事件时间开始'), '2026-07-09T00:00');
    await user.type(screen.getByLabelText('事件时间结束'), '2026-07-10T00:00');
    await user.click(screen.getByRole('button', { name: '搜索事件' }));

    expect(dataIngestionAPI.loadEvents).toHaveBeenLastCalledWith('secret-token', expect.objectContaining({
      page: 1,
      title: '美联储',
      event_status: 'confirmed',
      fact_status: 'verified'
    }));
  });

  it('lists source catalogs without exposing parser and shows scheduler runs', async () => {
    const user = userEvent.setup();
    vi.spyOn(dataIngestionAPI, 'loadRawDocuments').mockResolvedValue({ items: [], total: 0, page: 1, page_size: 50 });
    vi.spyOn(dataIngestionAPI, 'loadEvents').mockResolvedValue({ items: [], total: 0, page: 1, page_size: 50 });
    vi.spyOn(dataIngestionAPI, 'loadSourceCatalogs').mockResolvedValue({
      items: [
        {
          id: 'source-1',
          provider_key: 'llm_web_research',
          ingest_channel: 'ai_web_research',
          source_type: 'news',
          source_name: 'AI 全球政经搜索',
          source_url: 'https://example.com',
          status: 'active'
        }
      ]
    });
    vi.spyOn(schedulerAPI, 'loadSchedulerConfig').mockResolvedValue(defaultSchedulerConfig());
    vi.spyOn(schedulerAPI, 'loadSchedulerRuns').mockResolvedValue([
      {
        id: 'run-1',
        trigger_type: 'interval',
        status: 'succeeded',
        started_at: '2026-07-09T10:00:00Z',
        finished_at: '2026-07-09T10:00:30Z',
        total_sources: 2,
        succeeded_sources: 2,
        failed_sources: 0,
        skipped_sources: 0,
        error_summary: ''
      }
    ]);
    vi.spyOn(schedulerAPI, 'saveSchedulerConfig').mockResolvedValue(defaultSchedulerConfig());

    render(<DataIngestionCenter token="secret-token" />);

    await screen.findByRole('tab', { name: '原始数据' });
    await user.click(screen.getByRole('tab', { name: '搜索通道' }));
    expect(await screen.findByText('AI 全球政经搜索')).toBeInTheDocument();
    expect(screen.queryByText('parser')).not.toBeInTheDocument();

    await user.click(screen.getByRole('tab', { name: '调度器' }));
    const schedulerPanel = await screen.findByLabelText('调度器执行记录');
    expect(within(schedulerPanel).getByText('run-1')).toBeInTheDocument();
    expect(schedulerAPI.loadSchedulerRuns).toHaveBeenCalledWith('secret-token', 50);
  });
});

function defaultSchedulerConfig(): schedulerAPI.SchedulerConfig {
  return {
    enabled: false,
    mode: 'interval',
    interval_minutes: 60,
    fixed_times: [],
    concurrency: 1,
    batch_size: 10,
    timeout_seconds: 180,
    source_filter: {},
    timezone: 'Asia/Shanghai'
  };
}
