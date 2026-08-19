import { QueryClientProvider } from '@tanstack/react-query';
import { act, render, screen, within } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { afterEach, expect, it, vi } from 'vitest';
import * as dataIngestionAPI from '../../api/dataIngestion';
import { createQueryClient } from '../../lib/query-client';
import { EvidenceDetailSheet } from './EvidenceDetailSheet';

afterEach(() => vi.restoreAllMocks());

it('retries a failed collection document query without removing the original article', async () => {
  const user = userEvent.setup();
  let rejectFirstRequest: ((reason: Error) => void) | undefined;
  const loadCollectionDocument = vi
    .spyOn(dataIngestionAPI, 'loadCollectionDocument')
    .mockImplementationOnce(
      () =>
        new Promise((_resolve, reject) => {
          rejectFirstRequest = reject;
        })
    )
    .mockResolvedValue({
      available: true,
      url: 'https://tideai.tripwise.cn/raw-evidence/documents/2026/08/17/11f0864fc4078b47a4cc758149a2b0b7923654d2c7c8a694ad5b2d5ced4fc998.md'
    });

  render(
    <QueryClientProvider client={createQueryClient()}>
      <EvidenceDetailSheet
        evidence={{
          id: 'EVD00000000-0000-5000-8000-000000000001',
          raw_evidence_id: 'RAW00000000-0000-5000-8000-000000000001',
          title: '材料标题',
          summary: '证据摘要',
          semantic: { who: null, what: '发布公告', when: null, where: null, why: null, how: null },
          categories: [],
          source_name: '官方信源',
          source_level: 'L1_OFFICIAL',
          source_url: 'https://example.com/original-report',
          is_original: true,
          quoted_source_name: null,
          keywords: [],
          is_split: false,
          published_at: null,
          collected_at: '2026-08-19T02:00:00Z'
        }}
        onOpenChange={() => undefined}
        open
        token='secret-token'
      />
    </QueryClientProvider>
  );

  const dialog = screen.getByRole('dialog', { name: '材料标题' });
  expect(within(dialog).getByRole('link', { name: '访问原始文章' })).toBeInTheDocument();
  expect(within(dialog).getByText('正在加载采集文档…')).toBeInTheDocument();
  await act(async () => rejectFirstRequest?.(new Error('temporarily unavailable')));
  await user.click(await within(dialog).findByRole('button', { name: '重试加载采集文档' }));
  expect(await within(dialog).findByRole('link', { name: '打开采集文档' })).toBeInTheDocument();
  expect(loadCollectionDocument).toHaveBeenCalledTimes(2);
});
