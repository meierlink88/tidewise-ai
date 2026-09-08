import Taro, { usePullDownRefresh } from '@tarojs/taro';
import { View } from '@tarojs/components';
import { useEffect, useMemo } from 'react';
import { NormalizedDetailView } from './normalized-detail';
import {
  analysisKinds,
  type AnalysisKind,
  type AnalysisDetail
} from '../../../features/reports/normalized-contract';

import type { ReportPort } from '../../../features/reports/contract';
import { ReportError } from '../../../features/reports/contract';
import {
  parseReportDetailRoute,
  type ReportDetailRoute,
  type ReportEvidenceRoute
} from '../../../features/reports/navigation';
import { getReportPort } from '../../../features/reports/port';
import { reportErrorCopy } from '../../../features/reports/presentation';
import {
  ReportEvidenceSheetHost,
  ReportEvidenceSheetHostController
} from '../../../features/reports/report-evidence-sheet';
import { ReportStatePanel } from '../../../features/reports/report-components';
import type { ReportResourceState } from '../../../features/reports/session';
import { useReportResource } from '../../../features/reports/use-report-resource';
import { reportDetailShare } from '../../../features/reports/share';
import { usePageShare } from '../../../platform/use-page-share';
import './index.scss';

export type LoadedReportDetail = {
  targetType: 'analysis';
  detail: AnalysisDetail;
  reportId: string;
  kind: AnalysisKind;
};

export default function ReportDetailPage() {
  const instance = useMemo(() => Taro.getCurrentInstance(), []);
  const route = useMemo(() => safeDetailRoute(instance.router?.params), [instance]);
  const port = useMemo(() => getReportPort(), []);
  const evidenceSheet = useMemo(() => new ReportEvidenceSheetHostController(), []);
  const resource = useReportResource(
    `report-detail:${route?.reportId ?? 'invalid'}:${route?.targetType ?? 'invalid'}:${route?.targetKey ?? 'invalid'}`,
    () => loadReportDetail(port, route)
  );

  usePageShare(
    reportDetailShare(
      route,
      resource.state.status === 'ready' ? resource.state.data.detail.summary.title : undefined
    )
  );

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

  return (
    <>
      <ReportDetailView
        state={resource.state}
        onRetry={() => void resource.retry()}
        onOpenEvidence={evidenceSheet.open}
      />
      <ReportEvidenceSheetHost controller={evidenceSheet} port={port} />
    </>
  );
}

export async function loadReportDetail(
  port: ReportPort,
  route: ReportDetailRoute | null
): Promise<LoadedReportDetail> {
  if (!route) throw new ReportError('invalidRequest');
  if (analysisKinds.includes(route.targetType as AnalysisKind)) {
    const kind = route.targetType as AnalysisKind;
    return {
      targetType: 'analysis',
      detail: await port.getAnalysis(route.reportId, kind, route.targetKey),
      reportId: route.reportId,
      kind
    };
  }
  throw new ReportError('invalidRequest');
}

export function ReportDetailView({
  state,
  onRetry,
  onOpenEvidence
}: {
  state: ReportResourceState<LoadedReportDetail>;
  onRetry: () => void;
  onOpenEvidence: (route: ReportEvidenceRoute) => void;
}) {
  if (state.status === 'idle' || state.status === 'loading') {
    return (
      <View className='report-detail-page'>
        <ReportStatePanel title='正在读取推理详情' description='正在加载报告快照' busy />
      </View>
    );
  }
  if (state.status === 'error') {
    const copy = reportErrorCopy(state.error.kind);
    return (
      <View className='report-detail-page'>
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
      <View className='report-detail-page'>
        <ReportStatePanel title='暂无推理详情' description='该对象没有发布可展示的详情' />
      </View>
    );
  }

  return (
    <NormalizedDetailView
      key={`${state.data.reportId}:${state.data.kind}:${state.data.detail.summary.local_key}`}
      detail={state.data.detail}
      reportId={state.data.reportId}
      kind={state.data.kind}
      onEvidence={onOpenEvidence}
    />
  );
}

function resetPageScroll(): void {
  void Taro.pageScrollTo({ scrollTop: 0, duration: 0 });
  if (process.env.TARO_ENV === 'h5' && typeof window !== 'undefined') window.scrollTo(0, 0);
}
function safeDetailRoute(value: unknown): ReportDetailRoute | null {
  try {
    return parseReportDetailRoute(value);
  } catch {
    return null;
  }
}
