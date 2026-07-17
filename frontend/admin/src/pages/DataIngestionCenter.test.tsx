import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { describe, expect, it, vi } from 'vitest';
import * as dataIngestionAPI from '../api/dataIngestion';
import DataIngestionCenter from './DataIngestionCenter';

describe('DataIngestionCenter', () => {
  it('renders exactly the source, raw, and event tabs and loads raw documents by default', async () => {
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

    render(<DataIngestionCenter token="secret-token" />);

    expect(await screen.findByRole('tab', { name: '原始数据' })).toBeInTheDocument();
    expect(screen.getByRole('tab', { name: '全球事件' })).toBeInTheDocument();
    expect(screen.getByRole('tab', { name: '搜索通道' })).toBeInTheDocument();
    expect(screen.getAllByRole('tab')).toHaveLength(3);
    expect(screen.queryByRole('tab', { name: '调度器' })).not.toBeInTheDocument();
    expect(await screen.findByText('央行公布金融数据')).toBeInTheDocument();
    expect(dataIngestionAPI.loadRawDocuments).toHaveBeenCalledWith('secret-token', { page: 1, title: '' });
  });

  it('applies raw title search and event filters', async () => {
    const user = userEvent.setup();
    const eventTimeFrom = '2026-07-09T00:00';
    const eventTimeTo = '2026-07-10T00:00';
    const firstSeenFrom = '2026-07-08T00:00';
    const firstSeenTo = '2026-07-11T00:00';
    vi.spyOn(dataIngestionAPI, 'loadRawDocuments').mockResolvedValue({ items: [], total: 0, page: 1, page_size: 50 });
    vi.spyOn(dataIngestionAPI, 'loadEvents').mockResolvedValue({ items: [], total: 0, page: 1, page_size: 50 });
    vi.spyOn(dataIngestionAPI, 'loadSourceCatalogs').mockResolvedValue({ items: [] });

    render(<DataIngestionCenter token="secret-token" />);

    await screen.findByRole('tab', { name: '原始数据' });
    await user.type(screen.getByLabelText('原始数据标题搜索'), '央行');
    await user.click(screen.getByRole('button', { name: '搜索原始数据' }));

    expect(dataIngestionAPI.loadRawDocuments).toHaveBeenLastCalledWith('secret-token', { page: 1, title: '央行' });

    await user.click(screen.getByRole('tab', { name: '全球事件' }));
    await user.type(screen.getByLabelText('事件标题搜索'), '美联储');
    await user.selectOptions(screen.getByLabelText('事件状态'), 'confirmed');
    await user.selectOptions(screen.getByLabelText('事实状态'), 'verified');
    await user.type(screen.getByLabelText('事件时间开始'), eventTimeFrom);
    await user.type(screen.getByLabelText('事件时间结束'), eventTimeTo);
    await user.type(screen.getByLabelText('首次发现开始'), firstSeenFrom);
    await user.type(screen.getByLabelText('首次发现结束'), firstSeenTo);
    await user.click(screen.getByRole('button', { name: '搜索事件' }));

    expect(dataIngestionAPI.loadEvents).toHaveBeenLastCalledWith('secret-token', expect.objectContaining({
      page: 1,
      title: '美联储',
      event_status: 'confirmed',
      fact_status: 'verified',
      event_time_from: new Date(eventTimeFrom).toISOString(),
      event_time_to: new Date(eventTimeTo).toISOString(),
      first_seen_from: new Date(firstSeenFrom).toISOString(),
      first_seen_to: new Date(firstSeenTo).toISOString()
    }));
  });

  it('lists source catalogs without exposing parser or retired scheduler controls', async () => {
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
    render(<DataIngestionCenter token="secret-token" />);

    await screen.findByRole('tab', { name: '原始数据' });
    await user.click(screen.getByRole('tab', { name: '搜索通道' }));
    expect(await screen.findByText('AI 全球政经搜索')).toBeInTheDocument();
    expect(screen.getByText('https://example.com')).toBeInTheDocument();
    expect(screen.queryByText('parser')).not.toBeInTheDocument();
    expect(screen.queryByRole('tab', { name: '调度器' })).not.toBeInTheDocument();
    expect(screen.queryByLabelText('调度器执行记录')).not.toBeInTheDocument();
  });
});
