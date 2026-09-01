import Taro, { usePullDownRefresh } from '@tarojs/taro';
import { Image, Text, View } from '@tarojs/components';
import { useEffect, useMemo } from 'react';
import reportClockIcon from '../../../assets/icons/report-clock.svg';
import type {
  ReportEvidenceList,
  ReportEvidenceScope,
  ReportPort
} from '../../../features/reports/contract';
import { ReportError } from '../../../features/reports/contract';
import {
  parseReportEvidenceRoute,
  type ReportEvidenceRoute
} from '../../../features/reports/navigation';
import { getReportPort } from '../../../features/reports/port';
import {
  evidenceStableKey,
  formatShanghaiTimestamp,
  reportErrorCopy
} from '../../../features/reports/presentation';
import { ReportStatePanel } from '../../../features/reports/report-components';
import type { ReportResourceState } from '../../../features/reports/session';
import { useReportResource } from '../../../features/reports/use-report-resource';
import './index.scss';

const isEvidenceEmpty = (value: ReportEvidenceList) => value.items.length === 0;

export default function ReportEvidencesPage() {
  const instance = useMemo(() => Taro.getCurrentInstance(), []);
  const route = useMemo(
    () => safeEvidenceRoute(instance.router?.params, process.env.TARO_ENV === 'h5'),
    [instance]
  );
  const port = useMemo(() => getReportPort(), []);
  const resource = useReportResource(
    `report-evidences:${route?.reportId ?? 'invalid'}:${route?.scopeType ?? 'invalid'}:${route?.scopeKey ?? 'invalid'}`,
    () => loadReportEvidences(port, route),
    isEvidenceEmpty
  );

  useEffect(() => {
    void Taro.setNavigationBarTitle({ title: route?.title ?? '相关证据' });
  }, [route]);

  useEffect(() => {
    resetPageScroll();
  }, [route, resource.state.status]);

  usePullDownRefresh(async () => {
    await resource.refresh();
    const latest = resource.snapshot();
    if ((latest.status === 'ready' || latest.status === 'empty') && latest.refreshFailed) {
      void Taro.showToast({ title: '刷新失败，已保留当前内容', icon: 'none', duration: 1800 });
    }
    void Taro.stopPullDownRefresh();
  });

  return <ReportEvidencesView state={resource.state} onRetry={() => void resource.retry()} />;
}

export async function loadReportEvidences(
  port: ReportPort,
  route: ReportEvidenceRoute | null
): Promise<ReportEvidenceList> {
  if (!route) throw new ReportError('invalidRequest');
  const scope: ReportEvidenceScope = { type: route.scopeType, key: route.scopeKey };
  return port.getEvidences(route.reportId, scope);
}

export function ReportEvidencesView({
  state,
  onRetry
}: {
  state: ReportResourceState<ReportEvidenceList>;
  onRetry: () => void;
}) {
  if (state.status === 'idle' || state.status === 'loading') {
    return (
      <View className='report-evidences-page'>
        <ReportStatePanel title='正在读取相关证据' description='正在加载已发布的证据投影' busy />
      </View>
    );
  }
  if (state.status === 'error') {
    const copy = reportErrorCopy(state.error.kind);
    return (
      <View className='report-evidences-page'>
        <ReportStatePanel
          title={copy.title}
          description={copy.description}
          actionLabel='重新加载'
          onAction={onRetry}
        />
      </View>
    );
  }
  if (state.status === 'empty') {
    return (
      <View className='report-evidences-page'>
        <ReportStatePanel title='暂无相关证据' description='该对象当前没有发布证据条目' />
      </View>
    );
  }

  return (
    <View className='report-evidences-page'>
      {state.refreshFailed ? (
        <View className='report-evidences-refresh-warning'>
          刷新失败，当前展示上次成功读取的内容
        </View>
      ) : null}
      <View className='report-evidence-list'>
        {state.data.items.map((item, index) => (
          <View className='report-evidence-item' key={`${evidenceStableKey(item)}-${index}`}>
            <View className='report-evidence-item__time-row'>
              <Image
                className='report-evidence-item__clock'
                src={reportClockIcon}
                mode='aspectFit'
              />
              <Text className='report-evidence-item__time'>
                {item.publishedAt ? formatShanghaiTimestamp(item.publishedAt) : '时间待确认'}
              </Text>
            </View>
            <Text className='report-evidence-item__summary'>{item.summary}</Text>
            {item.keywords.length ? (
              <View className='report-evidence-keywords' ariaLabel='关键词'>
                {item.keywords.map((keyword) => (
                  <Text className='report-evidence-keyword' key={keyword}>
                    {keyword}
                  </Text>
                ))}
              </View>
            ) : null}
          </View>
        ))}
      </View>
    </View>
  );
}

function resetPageScroll(): void {
  void Taro.pageScrollTo({ scrollTop: 0, duration: 0 });
  if (process.env.TARO_ENV === 'h5' && typeof window !== 'undefined') {
    window.scrollTo(0, 0);
  }
}

function safeEvidenceRoute(
  value: unknown,
  decodeInboundValues: boolean
): ReportEvidenceRoute | null {
  try {
    return parseReportEvidenceRoute(value, decodeInboundValues);
  } catch {
    return null;
  }
}
