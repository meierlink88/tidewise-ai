import { useQuery } from '@tanstack/react-query';
import { ExternalLink, RefreshCw } from 'lucide-react';
import { loadCollectionDocument, type EvidenceItem } from '../../api/dataIngestion';
import {
  DetailItem,
  DetailList,
  DetailSection,
  nullableDetailValue
} from '../../components/admin/detail-description-list';
import { Badge } from '../../components/ui/badge';
import { Button } from '../../components/ui/Button';
import { StatusBadge } from '../../components/ui/StatusBadge';
import { Sheet, SheetContent, SheetDescription, SheetTitle } from '../../components/ui/sheet';
import { formatDateTime } from './shared';

export function EvidenceDetailSheet({
  evidence,
  open,
  token,
  onOpenChange
}: {
  evidence: EvidenceItem | null;
  open: boolean;
  token: string;
  onOpenChange: (open: boolean) => void;
}) {
  const rawEvidenceID = evidence?.raw_evidence_id ?? '';
  const collectionDocument = useQuery({
    queryKey: ['collection-center', 'collection-document', rawEvidenceID],
    queryFn: () => loadCollectionDocument(token, rawEvidenceID),
    enabled: open && rawEvidenceID !== ''
  });
  return (
    <Sheet open={open && evidence !== null} onOpenChange={onOpenChange}>
      {evidence ? (
        <SheetContent
          className='w-full max-w-[620px] gap-0 overflow-hidden p-0 sm:w-[min(620px,92vw)]'
          closeLabel='关闭证据详情'
          side='right'
        >
          <header className='shrink-0 border-b px-6 py-5 pr-16'>
            <span className='text-[10px] font-semibold tracking-[0.08em] text-muted-foreground uppercase'>
              证据详情
            </span>
            <SheetTitle className='mt-1.5 text-lg leading-7 font-semibold'>
              {evidence.title ?? '—'}
            </SheetTitle>
            <SheetDescription className='sr-only'>查看证据及其来源的完整业务信息</SheetDescription>
            <div className='mt-3 flex flex-wrap gap-2'>
              <StatusBadge tone={evidence.is_split ? 'running' : 'neutral'}>
                {evidence.is_split ? '已拆分' : '未拆分'}
              </StatusBadge>
              {evidence.categories.map((category) => (
                <Badge key={category.id} variant='outline'>
                  {category.name}
                </Badge>
              ))}
            </div>
          </header>
          <div className='min-h-0 flex-1 overflow-y-auto px-6 pb-8'>
            <DetailSection title='基本信息'>
              <p className='rounded-md bg-muted px-4 py-3 text-sm leading-6'>{evidence.summary}</p>
              <DetailList>
                <DetailItem label='拆分状态'>{evidence.is_split ? '已拆分' : '未拆分'}</DetailItem>
                <DetailItem full label='内容分类'>
                  {evidence.categories.length === 0 ? (
                    '—'
                  ) : (
                    <span className='grid gap-3'>
                      {evidence.categories.map((category) => (
                        <span className='grid gap-1' key={category.id}>
                          <span className='font-medium'>{category.name}</span>
                          <span className='text-muted-foreground'>{category.description}</span>
                        </span>
                      ))}
                    </span>
                  )}
                </DetailItem>
              </DetailList>
            </DetailSection>
            <DetailSection title='语义信息'>
              <DetailList>
                <DetailItem label='Who · 谁'>
                  {nullableDetailValue(evidence.semantic.who)}
                </DetailItem>
                <DetailItem label='When · 何时'>
                  {nullableDetailValue(evidence.semantic.when)}
                </DetailItem>
                <DetailItem full label='What · 什么'>
                  {evidence.semantic.what}
                </DetailItem>
                <DetailItem label='Where · 何地'>
                  {nullableDetailValue(evidence.semantic.where)}
                </DetailItem>
                <DetailItem label='Why · 为何'>
                  {nullableDetailValue(evidence.semantic.why)}
                </DetailItem>
                <DetailItem full label='How · 如何'>
                  {nullableDetailValue(evidence.semantic.how)}
                </DetailItem>
              </DetailList>
            </DetailSection>
            <DetailSection title='信源信息'>
              <DetailList>
                <DetailItem full label='信源 ID'>
                  <span className='font-mono'>{evidence.source_id}</span>
                </DetailItem>
                <DetailItem label='信源名称'>{evidence.source_name}</DetailItem>
                <DetailItem label='信源等级'>{evidence.source_level}</DetailItem>
                <DetailItem label='内容来源'>{evidence.is_original ? '原创' : '转载'}</DetailItem>
                {!evidence.is_original && evidence.quoted_source_name ? (
                  <DetailItem label='引用信源'>{evidence.quoted_source_name}</DetailItem>
                ) : null}
                <DetailItem full label='原始文章'>
                  <a
                    className='inline-flex items-center gap-1.5 text-primary underline-offset-4 hover:underline'
                    href={evidence.source_url}
                    rel='noreferrer noopener'
                    target='_blank'
                  >
                    访问原始文章
                    <ExternalLink aria-hidden='true' className='size-3.5' />
                  </a>
                </DetailItem>
                <DetailItem full label='采集文档'>
                  {collectionDocument.isPending ? (
                    <span className='text-muted-foreground'>正在加载采集文档…</span>
                  ) : collectionDocument.isError ? (
                    <span className='inline-flex flex-wrap items-center gap-2'>
                      <span className='text-destructive'>采集文档加载失败</span>
                      <Button
                        aria-label='重试加载采集文档'
                        onClick={() => void collectionDocument.refetch()}
                        size='sm'
                        type='button'
                        variant='outline'
                      >
                        <RefreshCw aria-hidden='true' className='size-3.5' />
                        重试
                      </Button>
                    </span>
                  ) : collectionDocument.data.available ? (
                    <a
                      className='inline-flex items-center gap-1.5 text-primary underline-offset-4 hover:underline'
                      href={collectionDocument.data.url}
                      rel='noreferrer noopener'
                      target='_blank'
                    >
                      打开采集文档
                      <ExternalLink aria-hidden='true' className='size-3.5' />
                    </a>
                  ) : (
                    <span className='text-muted-foreground'>暂无采集文档</span>
                  )}
                </DetailItem>
              </DetailList>
            </DetailSection>
            <DetailSection title='其他信息'>
              <DetailList>
                <DetailItem full label='关键词'>
                  {evidence.keywords.length === 0 ? (
                    '—'
                  ) : (
                    <span className='flex flex-wrap gap-2'>
                      {evidence.keywords.map((keyword, index) => (
                        <Badge key={`${index}-${keyword}`} variant='secondary'>
                          {keyword}
                        </Badge>
                      ))}
                    </span>
                  )}
                </DetailItem>
                <DetailItem label='发布时间'>
                  {evidence.published_at ? formatDateTime(evidence.published_at) : '—'}
                </DetailItem>
                <DetailItem label='采集时间'>{formatDateTime(evidence.collected_at)}</DetailItem>
              </DetailList>
            </DetailSection>
          </div>
        </SheetContent>
      ) : null}
    </Sheet>
  );
}
