import { Image, ScrollView, Text, View } from '@tarojs/components';
import reportClockIcon from '../../../assets/icons/report-clock.svg';
import type {
  ReportEvidenceList,
  ReportEvidenceScope,
  ReportPort
} from '../../../features/reports/contract';
import type { ReportEvidenceRoute } from '../../../features/reports/navigation';
import {
  evidenceStableKey,
  formatShanghaiTimestamp,
  reportErrorCopy
} from '../../../features/reports/presentation';
import { ReportStatePanel } from '../../../features/reports/report-components';
import type { ReportResourceState } from '../../../features/reports/session';
import { useReportResource } from '../../../features/reports/use-report-resource';
import './report-evidence-sheet.scss';

const isEvidenceEmpty = (value: ReportEvidenceList) => value.items.length === 0;

export function HomeReportEvidenceSheet({
  route,
  port,
  onClose
}: {
  route: ReportEvidenceRoute;
  port: ReportPort;
  onClose: () => void;
}) {
  const resource = useReportResource(
    `home-report-evidences:${route.reportId}:${route.scopeType}:${route.scopeKey}`,
    () => loadHomeReportEvidences(port, route),
    isEvidenceEmpty
  );

  return (
    <HomeReportEvidenceSheetView
      title={route.title}
      state={resource.state}
      onRetry={() => void resource.retry()}
      onClose={onClose}
    />
  );
}

export function loadHomeReportEvidences(
  port: ReportPort,
  route: ReportEvidenceRoute
): Promise<ReportEvidenceList> {
  const scope: ReportEvidenceScope = { type: route.scopeType, key: route.scopeKey };
  return port.getEvidences(route.reportId, scope);
}

export function HomeReportEvidenceSheetView({
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
    <View className='home-report-evidence-sheet' catchMove>
      <View
        className='home-report-evidence-sheet__overlay'
        role='button'
        ariaLabel='关闭相关证据'
        catchMove
        onClick={onClose}
      />
      <View
        className='home-report-evidence-sheet__panel'
        role='dialog'
        ariaLabel={title}
        catchMove
        onClick={(event) => event.stopPropagation()}
      >
        <View className='home-report-evidence-sheet__handle-zone'>
          <View className='home-report-evidence-sheet__handle' />
        </View>
        <ScrollView className='home-report-evidence-sheet__scroll' scrollY>
          <View className='home-report-evidence-sheet__content'>
            <HomeReportEvidenceSheetContent state={state} onRetry={onRetry} />
          </View>
        </ScrollView>
      </View>
    </View>
  );
}

function HomeReportEvidenceSheetContent({
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
    return <ReportStatePanel title='暂无相关证据' description='该卡片当前没有发布证据条目' />;
  }

  return (
    <View className='home-report-evidence-sheet__list'>
      {state.data.items.map((item, index) => (
        <View
          className='home-report-evidence-sheet__item'
          key={`${evidenceStableKey(item)}-${index}`}
        >
          <View className='home-report-evidence-sheet__time-row'>
            <Image
              className='home-report-evidence-sheet__clock'
              src={reportClockIcon}
              mode='aspectFit'
            />
            <Text className='home-report-evidence-sheet__time'>
              {item.publishedAt ? formatHomeEvidenceTimestamp(item.publishedAt) : '时间待确认'}
            </Text>
          </View>
          <Text className='home-report-evidence-sheet__summary'>{item.summary}</Text>
          {item.keywords.length ? (
            <View className='home-report-evidence-sheet__keywords' ariaLabel='关键词'>
              {item.keywords.map((keyword) => (
                <Text className='home-report-evidence-sheet__keyword' key={keyword}>
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

function formatHomeEvidenceTimestamp(value: string): string {
  const formatted = formatShanghaiTimestamp(value);
  return `${formatted.slice(5, 7)}-${formatted.slice(8)}`;
}
