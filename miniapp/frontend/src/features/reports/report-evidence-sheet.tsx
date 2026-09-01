import { Image, ScrollView, Text, View } from '@tarojs/components';
import reportClockIcon from '../../assets/icons/report-clock.svg';
import type { ReportEvidenceList, ReportEvidenceScope, ReportPort } from './contract';
import type { ReportEvidenceRoute } from './navigation';
import { evidenceStableKey, formatShanghaiTimestamp, reportErrorCopy } from './presentation';
import { ReportStatePanel } from './report-components';
import type { ReportResourceState } from './session';
import { useReportResource } from './use-report-resource';
import './report-evidence-sheet.scss';

const isEvidenceEmpty = (value: ReportEvidenceList) => value.items.length === 0;

export function ReportEvidenceSheet({
  route,
  port,
  onClose
}: {
  route: ReportEvidenceRoute;
  port: ReportPort;
  onClose: () => void;
}) {
  const resource = useReportResource(
    `report-evidences:${route.reportId}:${route.scopeType}:${route.scopeKey}`,
    () => loadReportEvidences(port, route),
    isEvidenceEmpty
  );

  return (
    <ReportEvidenceSheetView
      title={route.title}
      state={resource.state}
      onRetry={() => void resource.retry()}
      onClose={onClose}
    />
  );
}

export function loadReportEvidences(
  port: ReportPort,
  route: ReportEvidenceRoute
): Promise<ReportEvidenceList> {
  const scope: ReportEvidenceScope = { type: route.scopeType, key: route.scopeKey };
  return port.getEvidences(route.reportId, scope);
}

export function ReportEvidenceSheetView({
  title,
  state,
  onRetry,
  onClose
}: {
  title: string;
  state: ReportResourceState<ReportEvidenceList>;
  onRetry: () => void;
  onClose: () => void;
}) {
  return (
    <View className='report-evidence-sheet' catchMove>
      <View
        className='report-evidence-sheet__overlay'
        role='button'
        ariaLabel='关闭相关证据'
        catchMove
        onClick={onClose}
      />
      <View
        className='report-evidence-sheet__panel'
        role='dialog'
        ariaLabel={title}
        catchMove
        onClick={(event) => event.stopPropagation()}
      >
        <View className='report-evidence-sheet__handle-zone'>
          <View className='report-evidence-sheet__handle' />
        </View>
        <ScrollView className='report-evidence-sheet__scroll' scrollY>
          <View className='report-evidence-sheet__content'>
            <ReportEvidenceSheetContent state={state} onRetry={onRetry} />
          </View>
        </ScrollView>
      </View>
    </View>
  );
}

function ReportEvidenceSheetContent({
  state,
  onRetry
}: {
  state: ReportResourceState<ReportEvidenceList>;
  onRetry: () => void;
}) {
  if (state.status === 'idle' || state.status === 'loading') {
    return (
      <ReportStatePanel title='正在读取相关证据' description='正在加载已发布的证据投影' busy />
    );
  }
  if (state.status === 'error') {
    const copy = reportErrorCopy(state.error.kind);
    return (
      <ReportStatePanel
        title={copy.title}
        description={copy.description}
        actionLabel='重新加载'
        onAction={onRetry}
      />
    );
  }
  if (state.status === 'empty') {
    return <ReportStatePanel title='暂无相关证据' description='该对象当前没有发布证据条目' />;
  }

  return (
    <View className='report-evidence-sheet__list'>
      {state.data.items.map((item, index) => (
        <View className='report-evidence-sheet__item' key={`${evidenceStableKey(item)}-${index}`}>
          <View className='report-evidence-sheet__time-row'>
            <Image
              className='report-evidence-sheet__clock'
              src={reportClockIcon}
              mode='aspectFit'
            />
            <Text className='report-evidence-sheet__time'>
              {item.publishedAt ? formatEvidenceTimestamp(item.publishedAt) : '时间待确认'}
            </Text>
          </View>
          <Text className='report-evidence-sheet__summary'>{item.summary}</Text>
          {item.keywords.length ? (
            <View className='report-evidence-sheet__keywords' ariaLabel='关键词'>
              {item.keywords.map((keyword) => (
                <Text className='report-evidence-sheet__keyword' key={keyword}>
                  {keyword}
                </Text>
              ))}
            </View>
          ) : null}
        </View>
      ))}
    </View>
  );
}

function formatEvidenceTimestamp(value: string): string {
  const formatted = formatShanghaiTimestamp(value);
  return `${formatted.slice(5, 7)}-${formatted.slice(8)}`;
}
