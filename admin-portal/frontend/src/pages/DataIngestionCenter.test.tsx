import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { afterEach, describe, expect, it, vi } from 'vitest';
import * as dataIngestionAPI from '../api/dataIngestion';
import DataIngestionCenter from './DataIngestionCenter';

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

    render(<DataIngestionCenter token='secret-token' />);

    expect(screen.getByRole('heading', { name: '事件中心' })).toBeInTheDocument();
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
    render(<DataIngestionCenter token='secret-token' />);

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
});
