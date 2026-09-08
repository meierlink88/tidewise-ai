import Taro, { usePullDownRefresh } from '@tarojs/taro';
import { Text, View } from '@tarojs/components';
import { useMemo, useState } from 'react';
import { NormalizedHome } from './normalized-home';

import type { ReportHome } from '../../features/reports/contract';
import {
  navigateToReportDetail,
  type ReportDetailRoute,
  type ReportEvidenceRoute
} from '../../features/reports/navigation';
import { getReportPort } from '../../features/reports/port';
import { reportErrorCopy } from '../../features/reports/presentation';
import { ReportStatePanel } from '../../features/reports/report-components';
import {
  ReportEvidenceSheetHost,
  ReportEvidenceSheetHostController
} from '../../features/reports/report-evidence-sheet';
import type { ReportResourceState } from '../../features/reports/session';
import { useReportResource } from '../../features/reports/use-report-resource';
import { getHomeChromeMetrics, type HomeChromeMetrics } from '../../platform/system-ui';
import { HomeHeader } from './components/home-header';
import { homeReportShare } from '../../features/reports/share';
import { usePageShare } from '../../platform/use-page-share';
import './index.scss';

interface HomeRefreshAPI {
  stopPullDownRefresh: () => unknown;
  showToast: (options: { title: string; icon: 'none'; duration: number }) => unknown;
}

const isHomeEmpty = (home: ReportHome) => home.reports.length === 0;

export default function IndexPage() {
  const isSinglePage = usePageShare(homeReportShare());
  const [query, setQuery] = useState('');
  const chrome = useMemo(() => getHomeChromeMetrics(Taro), []);
  const port = useMemo(() => getReportPort(), []);
  const evidenceSheet = useMemo(() => new ReportEvidenceSheetHostController(), []);
  const resource = useReportResource('report-home', () => port.getHome(), isHomeEmpty);

  const refreshHome = async () => {
    await resource.refresh();
    const latest = resource.snapshot();
    if ((latest.status === 'ready' || latest.status === 'empty') && latest.refreshFailed) {
      void Taro.showToast({ title: '刷新失败，已保留当前内容', icon: 'none', duration: 1800 });
    }
  };

  usePullDownRefresh(async () => {
    await refreshHome();
    await stopHomeRefresh(Taro);
  });

  return (
    <>
      <IndexView
        isSinglePage={isSinglePage}
        chrome={chrome}
        query={query}
        onQueryChange={setQuery}
        state={resource.state}
        onRetry={() => void resource.retry()}
        onRefresh={() => void refreshHome()}
        onOpenDetail={(route) => navigateToReportDetail(Taro, route)}
        onOpenEvidence={evidenceSheet.open}
      />
      <ReportEvidenceSheetHost controller={evidenceSheet} port={port} />
    </>
  );
}

export function IndexView({
  isSinglePage = false,
  chrome,
  query,
  onQueryChange,
  state,
  onRetry,
  onRefresh,
  onOpenDetail,
  onOpenEvidence
}: {
  isSinglePage?: boolean;
  chrome: HomeChromeMetrics;
  query: string;
  onQueryChange: (query: string) => void;
  state: ReportResourceState<ReportHome>;
  onRetry: () => void;
  onRefresh: () => void;
  onOpenDetail: (route: ReportDetailRoute) => void;
  onOpenEvidence: (route: ReportEvidenceRoute) => void;
}) {
  if (state.status === 'ready' && state.data.reports[0]?.analysisGroups) {
    return (
      <View className='home-page'>
        <HomeHeader
          chrome={chrome}
          query={query}
          onQueryChange={onQueryChange}
          isSinglePage={isSinglePage}
        />
        <View className='home-content'>
          {state.refreshFailed ? (
            <View className='home-refresh-warning' onClick={onRefresh}>
              刷新失败，点击重试；当前展示上次成功读取的内容
            </View>
          ) : null}
          <NormalizedHome
            key={state.data.reports[0].report.id}
            group={state.data.reports[0]}
            query={query}
            onDetail={isSinglePage ? undefined : onOpenDetail}
            onEvidence={onOpenEvidence}
          />
        </View>
      </View>
    );
  }
  return (
    <View className='home-page'>
      <HomeHeader
        chrome={chrome}
        query={query}
        onQueryChange={onQueryChange}
        isSinglePage={isSinglePage}
      />

      <View className='home-content'>
        <View className='home-section-heading'>
          <Text className='home-section-heading__title'>今日观潮</Text>
        </View>
        <HomeReportState state={state} onRetry={onRetry} />
      </View>
    </View>
  );
}

function HomeReportState({
  state,
  onRetry
}: {
  state: ReportResourceState<ReportHome>;
  onRetry: () => void;
}) {
  if (state.status === 'idle' || state.status === 'loading')
    return <ReportStatePanel title='正在读取报告' description='正在加载本次推理卡片' busy />;
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
  if (state.status === 'empty' || !state.data.reports.length)
    return <ReportStatePanel title='暂无推理报告' description='报告发布后会在这里展示' />;
  return <ReportStatePanel title='报告格式暂不支持' description='请读取新版报告' />;
}

export async function stopHomeRefresh(api: HomeRefreshAPI): Promise<void> {
  void api.stopPullDownRefresh();
}
