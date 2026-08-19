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

  it('opens the complete Event detail from a row and restores focus on Escape', async () => {
    const user = userEvent.setup();
    vi.spyOn(dataIngestionAPI, 'loadEvents').mockResolvedValue({
      items: [
        {
          id: 'EVT00000000-0000-5000-8000-000000000001',
          title: '稀土出口管理新规',
          summary: '商务部公布新的出口许可安排。',
          semantic: {
            who: '中华人民共和国商务部',
            what: '调整稀土出口许可要求',
            when: '2026-08-19',
            where: '中国',
            why: '加强战略性矿产出口管理',
            how: '通过公告明确申报流程'
          },
          modality: 'FACT',
          occurred_at: '2026-08-19T01:30:00Z',
          announced_at: null,
          status: 'ACTIVE'
        }
      ],
      total: 1,
      page: 1,
      page_size: 50
    });
    renderCenter();

    const row = await screen.findByRole('row', { name: '查看稀土出口管理新规详情' });
    row.focus();
    await user.keyboard('{Enter}');

    const dialog = screen.getByRole('dialog', { name: '稀土出口管理新规' });
    expect(within(dialog).getByText('商务部公布新的出口许可安排。')).toBeInTheDocument();
    expect(within(dialog).getByText('中华人民共和国商务部')).toBeInTheDocument();
    expect(within(dialog).getByText('调整稀土出口许可要求')).toBeInTheDocument();
    expect(within(dialog).getByText('—')).toBeInTheDocument();
    expect(within(dialog).queryByText(/EVT00000000/)).not.toBeInTheDocument();

    await user.keyboard('{Escape}');
    expect(screen.queryByRole('dialog')).not.toBeInTheDocument();
    await waitFor(() => expect(row).toHaveFocus());
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
          semantic: {
            who: '主体',
            what: '发生事项',
            when: null,
            where: null,
            why: null,
            how: null
          },
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
          source_url: 'https://example.com/report',
          is_original: true,
          quoted_source_name: null,
          keywords: ['政策'],
          is_split: true,
          published_at: '2026-08-19T01:00:00Z',
          collected_at: '2026-08-19T02:00:00Z'
        },
        {
          id: 'EVD00000000-0000-5000-8000-000000000002',
          raw_evidence_id: 'RAW00000000-0000-5000-8000-000000000002',
          title: null,
          summary: '无标题证据',
          semantic: { who: null, what: '另一事项', when: null, where: null, why: null, how: null },
          categories: [],
          source_name: '官方信源',
          source_level: 'L1_OFFICIAL',
          source_url: 'https://example.com/other',
          is_original: true,
          quoted_source_name: null,
          keywords: [],
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

  it('opens complete Evidence details and only shows quoted source for reposts', async () => {
    const user = userEvent.setup();
    vi.spyOn(dataIngestionAPI, 'loadEvents').mockResolvedValue({
      items: [],
      total: 0,
      page: 1,
      page_size: 50
    });
    vi.spyOn(dataIngestionAPI, 'loadEvidenceCategories').mockResolvedValue([]);
    vi.spyOn(dataIngestionAPI, 'loadCollectionDocument')
      .mockResolvedValueOnce({
        available: true,
        url: 'https://tideai.tripwise.cn/raw-evidence/documents/2026/08/17/11f0864fc4078b47a4cc758149a2b0b7923654d2c7c8a694ad5b2d5ced4fc998.md'
      })
      .mockResolvedValueOnce({ available: false, url: null });
    vi.spyOn(dataIngestionAPI, 'loadEvidences').mockResolvedValue({
      items: [
        {
          id: 'EVD00000000-0000-5000-8000-000000000001',
          raw_evidence_id: 'RAW00000000-0000-5000-8000-000000000001',
          title: '转载材料标题',
          summary: '原子证据摘要',
          semantic: {
            who: '赛意信息',
            what: '合作项目有序推进',
            when: '2026-08-17',
            where: '中国',
            why: '回应投资者关注',
            how: '通过公开互动渠道披露'
          },
          categories: [
            {
              id: 'EVC00000000-0000-5000-8000-000000000001',
              code: 'COMMENTARY',
              name: '评论 / 社论 / 观点',
              description: '机构或个人对事件的评论、判断或观点。'
            }
          ],
          source_name: '人民财讯',
          source_level: 'L3_MEDIA',
          source_url: 'https://example.com/reposted-report',
          is_original: false,
          quoted_source_name: '赛意信息投资者互动平台',
          keywords: ['赛意信息', '业务合作'],
          is_split: false,
          published_at: '2026-08-17T10:30:00Z',
          collected_at: '2026-08-17T10:31:00Z'
        },
        {
          id: 'EVD00000000-0000-5000-8000-000000000002',
          raw_evidence_id: 'RAW00000000-0000-5000-8000-000000000002',
          title: '原创材料标题',
          summary: '原创证据摘要',
          semantic: { who: null, what: '发布公告', when: null, where: null, why: null, how: null },
          categories: [],
          source_name: '商务部',
          source_level: 'L1_OFFICIAL',
          source_url: 'https://example.com/original-report',
          is_original: true,
          quoted_source_name: null,
          keywords: [],
          is_split: true,
          published_at: null,
          collected_at: '2026-08-19T01:00:00Z'
        }
      ],
      total: 2,
      page: 1,
      page_size: 50
    });
    renderCenter();
    await user.click(screen.getByRole('tab', { name: '证据中心' }));

    const repostRow = await screen.findByRole('row', { name: '查看转载材料标题详情' });
    await user.click(repostRow);
    const repostDialog = screen.getByRole('dialog', { name: '转载材料标题' });
    expect(within(repostDialog).getByText('原子证据摘要')).toBeInTheDocument();
    expect(
      within(repostDialog).getByText('机构或个人对事件的评论、判断或观点。')
    ).toBeInTheDocument();
    expect(within(repostDialog).getByText('合作项目有序推进')).toBeInTheDocument();
    expect(within(repostDialog).getByText('赛意信息投资者互动平台')).toBeInTheDocument();
    expect(within(repostDialog).getByText('业务合作')).toBeInTheDocument();
    expect(within(repostDialog).getByRole('link', { name: '访问原始文章' })).toHaveAttribute(
      'href',
      'https://example.com/reposted-report'
    );
    expect(await within(repostDialog).findByRole('link', { name: '打开采集文档' })).toHaveAttribute(
      'href',
      'https://tideai.tripwise.cn/raw-evidence/documents/2026/08/17/11f0864fc4078b47a4cc758149a2b0b7923654d2c7c8a694ad5b2d5ced4fc998.md'
    );
    expect(within(repostDialog).getByText('原始文章')).toBeInTheDocument();
    expect(within(repostDialog).getByText('采集文档')).toBeInTheDocument();
    expect(
      within(repostDialog).queryByText(/EVD00000000|RAW00000000|完整原文|哈希/)
    ).not.toBeInTheDocument();

    await user.click(within(repostDialog).getByRole('button', { name: '关闭证据详情' }));
    const originalRow = screen.getByRole('row', { name: '查看原创材料标题详情' });
    await user.click(originalRow);
    const originalDialog = screen.getByRole('dialog', { name: '原创材料标题' });
    expect(within(originalDialog).getByText('原创')).toBeInTheDocument();
    expect(within(originalDialog).queryByText('引用信源')).not.toBeInTheDocument();
    expect(within(originalDialog).getAllByText('—').length).toBeGreaterThan(0);
    expect(await within(originalDialog).findByText('暂无采集文档')).toBeInTheDocument();
  });
});
