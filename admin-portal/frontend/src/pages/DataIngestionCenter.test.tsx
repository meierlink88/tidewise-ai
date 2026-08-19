import { render, screen, waitFor, within } from '@testing-library/react';
import { QueryClientProvider } from '@tanstack/react-query';
import userEvent from '@testing-library/user-event';
import { afterEach, describe, expect, it, vi } from 'vitest';
import * as dataIngestionAPI from '../api/dataIngestion';
import DataIngestionCenter from './DataIngestionCenter';
import { createQueryClient } from '../lib/query-client';

function renderCenter() {
  return render(
    <QueryClientProvider client={createQueryClient()}>
      <DataIngestionCenter token='secret-token' />
    </QueryClientProvider>
  );
}

describe('DataIngestionCenter', () => {
  afterEach(() => vi.restoreAllMocks());

  it('loads and renders the current Event contract', async () => {
    const what = '维持利率不变';
    vi.spyOn(dataIngestionAPI, 'loadEvents').mockResolvedValue({
      items: [
        {
          id: 'EVT00000000-0000-5000-8000-000000000001',
          title: '全球市场事件',
          summary: '摘要',
          semantic: { who: null, what, when: null, where: null, why: null, how: null },
          modality: 'FACT',
          occurred_at: '2026-07-09T08:00:00Z',
          announced_at: null,
          status: 'ACTIVE'
        }
      ],
      total: 1,
      page: 1,
      page_size: 50
    });

    renderCenter();

    expect(screen.getByRole('heading', { name: '采集中心' })).toBeInTheDocument();
    expect(screen.getByRole('tab', { name: '事件中心' })).toHaveAttribute('aria-selected', 'true');
    expect(screen.getByRole('tab', { name: '证据中心' })).toBeInTheDocument();
    expect(screen.getByRole('tab', { name: '信源管理' })).toBeInTheDocument();
    const eventFilters = screen.getByRole('form', { name: '事件筛选条件' });
    const eventSearch = screen.getByRole('button', { name: '搜索事件' });
    expect(eventSearch).toHaveAttribute('form', 'event-filters');
    expect(
      within(eventFilters).queryByRole('button', { name: '搜索事件' })
    ).not.toBeInTheDocument();
    expect(within(screen.getByRole('toolbar', { name: '事件中心操作' })).getByRole('button')).toBe(
      eventSearch
    );
    expect(screen.queryByRole('button', { name: /新增|删除|刷新|重置/ })).not.toBeInTheDocument();
    expect(await screen.findByText('全球市场事件')).toBeInTheDocument();
    expect(screen.getByText('FACT')).toBeInTheDocument();
    expect(screen.getByText('ACTIVE')).toBeInTheDocument();
    expect(dataIngestionAPI.loadEvents).toHaveBeenCalledWith('secret-token', {
      page: 1,
      title: ''
    });
  });

  it('applies Event filters and retries a safe error', async () => {
    const user = userEvent.setup();
    const loadEvents = vi
      .spyOn(dataIngestionAPI, 'loadEvents')
      .mockRejectedValueOnce(new Error('internal server error'))
      .mockResolvedValue({ items: [], total: 0, page: 1, page_size: 50 });
    renderCenter();

    expect(await screen.findByRole('alert')).toHaveTextContent('数据加载失败，请稍后重试。');
    await user.click(screen.getByRole('button', { name: '重试' }));
    await waitFor(() => expect(loadEvents).toHaveBeenCalledTimes(2));

    await user.type(screen.getByLabelText('事件标题搜索'), '美联储');
    await user.click(screen.getByLabelText('模态'));
    await user.click(screen.getByRole('option', { name: '事实' }));
    await user.click(screen.getByLabelText('状态'));
    await user.click(screen.getByRole('option', { name: '活跃' }));
    await user.click(screen.getByRole('button', { name: '搜索事件' }));

    await waitFor(() =>
      expect(loadEvents).toHaveBeenLastCalledWith(
        'secret-token',
        expect.objectContaining({
          page: 1,
          title: '美联储',
          modality: 'FACT',
          status: 'ACTIVE'
        })
      )
    );
  });

  it('loads the confirmed Evidence and Source list fields when tabs are selected', async () => {
    const user = userEvent.setup();
    vi.spyOn(dataIngestionAPI, 'loadEvents').mockResolvedValue({
      items: [],
      total: 0,
      page: 1,
      page_size: 50
    });
    vi.spyOn(dataIngestionAPI, 'loadEvidenceCategories').mockResolvedValue([
      {
        id: 'EVC00000000-0000-5000-8000-000000000001',
        code: 'EVENT_BRIEF',
        name: '事件简报',
        description: '说明'
      }
    ]);
    vi.spyOn(dataIngestionAPI, 'loadEvidences').mockResolvedValue({
      items: [
        {
          id: 'EVD00000000-0000-5000-8000-000000000001',
          raw_evidence_id: 'RAW00000000-0000-5000-8000-000000000001',
          title: '原始标题',
          summary: '证据摘要',
          categories: [
            {
              id: 'EVC00000000-0000-5000-8000-000000000001',
              code: 'EVENT_BRIEF',
              name: '事件简报',
              description: '说明'
            }
          ],
          source_name: '官方信源',
          source_level: 'L1_OFFICIAL',
          is_split: true,
          published_at: '2026-08-19T01:00:00Z',
          collected_at: '2026-08-19T02:00:00Z'
        },
        {
          id: 'EVD00000000-0000-5000-8000-000000000002',
          raw_evidence_id: 'RAW00000000-0000-5000-8000-000000000002',
          title: null,
          summary: '无标题证据',
          categories: [],
          source_name: '官方信源',
          source_level: 'L1_OFFICIAL',
          is_split: false,
          published_at: null,
          collected_at: '2026-08-19T03:00:00Z'
        }
      ],
      total: 2,
      page: 1,
      page_size: 50
    });
    vi.spyOn(dataIngestionAPI, 'loadSources').mockResolvedValue({
      items: [
        {
          id: 'SRC00000000-0000-5000-8000-000000000001',
          code: 'official',
          name: '官方信源',
          ownership_type: 'fixed',
          channel_type: 'api',
          enabled: true,
          priority: 1,
          default_source_level: 'L1_OFFICIAL',
          updated_at: '2026-08-19T02:00:00Z'
        }
      ],
      total: 1,
      page: 1,
      page_size: 50
    });
    renderCenter();

    await user.click(screen.getByRole('tab', { name: '证据中心' }));
    expect(await screen.findByText('原始标题')).toBeInTheDocument();
    expect(screen.getByText('证据摘要')).toBeInTheDocument();
    expect(screen.getAllByText('事件简报').length).toBeGreaterThan(0);
    expect(screen.getAllByText('—')).toHaveLength(2);

    await user.click(screen.getByRole('tab', { name: '信源管理' }));
    expect(await screen.findByText('official')).toBeInTheDocument();
    expect(screen.getByText('L1_OFFICIAL')).toBeInTheDocument();
    expect(screen.queryByText(/endpoint/i)).not.toBeInTheDocument();
    expect(screen.queryByText(/adapter/i)).not.toBeInTheDocument();
  });
});
