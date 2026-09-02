import type { EventItem } from '../../api/dataIngestion';
import {
  DetailItem,
  DetailList,
  DetailSection
} from '../../components/admin/detail-description-list';
import { StatusBadge } from '../../components/ui/StatusBadge';
import { Sheet, SheetContent, SheetDescription, SheetTitle } from '../../components/ui/sheet';
import { formatDateTime } from './shared';

export function EventDetailSheet({
  event,
  open,
  onOpenChange
}: {
  event: EventItem | null;
  open: boolean;
  onOpenChange: (open: boolean) => void;
}) {
  return (
    <Sheet open={open && event !== null} onOpenChange={onOpenChange}>
      {event ? (
        <SheetContent
          className='w-full max-w-[620px] gap-0 overflow-hidden p-0 sm:w-[min(620px,92vw)]'
          closeLabel='关闭事件详情'
          side='right'
        >
          <header className='shrink-0 border-b px-6 py-5 pr-16'>
            <span className='text-[10px] font-semibold tracking-[0.08em] text-muted-foreground uppercase'>
              事件详情
            </span>
            <SheetTitle className='mt-1.5 text-lg leading-7 font-semibold'>
              {event.title}
            </SheetTitle>
            <SheetDescription className='sr-only'>查看事件的完整业务信息</SheetDescription>
            <div className='mt-3 flex flex-wrap gap-2'>
              <StatusBadge tone='running'>{event.semantic.modality}</StatusBadge>
              <StatusBadge
                tone={
                  event.status === 'ACTIVE'
                    ? 'success'
                    : event.status === 'DEPRECATED'
                      ? 'danger'
                      : 'neutral'
                }
              >
                {event.status}
              </StatusBadge>
            </div>
          </header>
          <div className='min-h-0 flex-1 overflow-y-auto px-6 pb-8'>
            <DetailSection title='基本信息'>
              <p className='rounded-md bg-muted px-4 py-3 text-sm leading-6'>{event.summary}</p>
              <DetailList>
                <DetailItem label='模态'>{event.semantic.modality}</DetailItem>
                <DetailItem label='状态'>{event.status}</DetailItem>
                <DetailItem label='发生时间'>
                  {formatNullableTime(event.semantic.time.occurred_at)}
                </DetailItem>
                <DetailItem label='公布时间'>
                  {formatNullableTime(event.semantic.time.announced_at)}
                </DetailItem>
                <DetailItem label='观察时间'>
                  {formatNullableTime(event.semantic.time.observed_at)}
                </DetailItem>
              </DetailList>
            </DetailSection>
            <DetailSection title='事件语义'>
              <DetailList>
                <DetailItem label='Actors · 参与方'>
                  {listDetailValue(event.semantic.actors)}
                </DetailItem>
                <DetailItem full label='Action · 动作'>
                  {event.semantic.action}
                </DetailItem>
                <DetailItem full label='Objects · 对象'>
                  {listDetailValue(event.semantic.objects)}
                </DetailItem>
                <DetailItem label='Stage · 阶段'>{event.semantic.stage}</DetailItem>
                <DetailItem label='Jurisdictions · 辖区'>
                  {listDetailValue(event.semantic.jurisdictions)}
                </DetailItem>
                <DetailItem label='Effective at · 生效时间'>
                  {formatNullableTime(event.semantic.time.effective_at)}
                </DetailItem>
                <DetailItem label='Time precision · 时间精度'>
                  {event.semantic.time.precision}
                </DetailItem>
                <DetailItem full label='Reason · 原因'>
                  {event.semantic.reason ?? '—'}
                </DetailItem>
                <DetailItem full label='Method · 执行方式'>
                  {event.semantic.method ?? '—'}
                </DetailItem>
              </DetailList>
            </DetailSection>
          </div>
        </SheetContent>
      ) : null}
    </Sheet>
  );
}

function formatNullableTime(value: string | null): string {
  return value ? formatDateTime(value) : '—';
}

function listDetailValue(values: string[]): string {
  return values.length > 0 ? values.join('、') : '—';
}
