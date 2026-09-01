import Taro, { usePullDownRefresh } from '@tarojs/taro';
import { Image, Text, View } from '@tarojs/components';
import { useMemo, useState } from 'react';
import reportArrowRightIcon from '../../assets/icons/report-arrow-right.svg';
import reportBarChartIcon from '../../assets/icons/report-bar-chart.svg';
import reportCubeIcon from '../../assets/icons/report-cube.svg';
import reportGlobeIcon from '../../assets/icons/report-globe.svg';
import reportLayersIcon from '../../assets/icons/report-layers.svg';
import type { ReportCard, ReportHome, ReportHomeGroup } from '../../features/reports/contract';
import {
  navigateToReportDetail,
  navigateToReportEvidences,
  type ReportDetailRoute,
  type ReportEvidenceRoute
} from '../../features/reports/navigation';
import { getReportPort } from '../../features/reports/port';
import { formatShanghaiTimestamp, reportErrorCopy } from '../../features/reports/presentation';
import {
  ReportEvidenceButton,
  ReportImpactSignals,
  ReportStatePanel
} from '../../features/reports/report-components';
import type { ReportResourceState } from '../../features/reports/session';
import { useReportResource } from '../../features/reports/use-report-resource';
import { getHomeChromeMetrics, type HomeChromeMetrics } from '../../platform/system-ui';
import { HomeHeader } from './components/home-header';
import './index.scss';

interface HomeRefreshAPI {
  stopPullDownRefresh: () => unknown;
  showToast: (options: { title: string; icon: 'none'; duration: number }) => unknown;
}

const isHomeEmpty = (home: ReportHome) => home.reports.length === 0;

export default function IndexPage() {
  const [query, setQuery] = useState('');
  const chrome = useMemo(() => getHomeChromeMetrics(Taro), []);
  const port = useMemo(() => getReportPort(), []);
  const resource = useReportResource('report-home', () => port.getHome(), isHomeEmpty);

  usePullDownRefresh(async () => {
    await resource.refresh();
    const latest = resource.snapshot();
    if ((latest.status === 'ready' || latest.status === 'empty') && latest.refreshFailed) {
      void Taro.showToast({ title: '刷新失败，已保留当前内容', icon: 'none', duration: 1800 });
    }
    void Taro.stopPullDownRefresh();
  });

  return (
    <IndexView
      chrome={chrome}
      query={query}
      onQueryChange={setQuery}
      state={resource.state}
      onRetry={() => void resource.retry()}
      onOpenDetail={(route) => navigateToReportDetail(Taro, route)}
      onOpenEvidence={(route) => navigateToReportEvidences(Taro, route)}
    />
  );
}

export function IndexView({
  chrome,
  query,
  onQueryChange,
  state,
  onRetry,
  onOpenDetail,
  onOpenEvidence
}: {
  chrome: HomeChromeMetrics;
  query: string;
  onQueryChange: (query: string) => void;
  state: ReportResourceState<ReportHome>;
  onRetry: () => void;
  onOpenDetail: (route: ReportDetailRoute) => void;
  onOpenEvidence: (route: ReportEvidenceRoute) => void;
}) {
  const summary = homeSummary(state);
  return (
    <View className='home-page'>
      <HomeHeader chrome={chrome} query={query} onQueryChange={onQueryChange} />

      <View className='home-content'>
        <View className='home-section-heading'>
          <Text className='home-section-heading__title'>今日推理</Text>
          <Text className='home-section-heading__summary'>{summary}</Text>
        </View>
        <HomeReportState
          state={state}
          onRetry={onRetry}
          onOpenDetail={onOpenDetail}
          onOpenEvidence={onOpenEvidence}
        />
      </View>
    </View>
  );
}

function HomeReportState({
  state,
  onRetry,
  onOpenDetail,
  onOpenEvidence
}: {
  state: ReportResourceState<ReportHome>;
  onRetry: () => void;
  onOpenDetail: (route: ReportDetailRoute) => void;
  onOpenEvidence: (route: ReportEvidenceRoute) => void;
}) {
  if (state.status === 'idle' || state.status === 'loading') {
    return <ReportStatePanel title='正在读取报告' description='正在加载本次推理卡片' busy />;
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
    return <ReportStatePanel title='暂无推理报告' description='报告发布后会在这里展示' />;
  }
  return (
    <View className='home-report-list'>
      {state.refreshFailed ? (
        <View className='home-refresh-warning'>刷新失败，当前展示上次成功读取的内容</View>
      ) : null}
      {state.data.reports.map((group) => (
        <HomeReportGroupView
          key={group.report.id}
          group={group}
          fallback={state.data.selection.mode === 'latest_fallback'}
          onOpenDetail={onOpenDetail}
          onOpenEvidence={onOpenEvidence}
        />
      ))}
    </View>
  );
}

function HomeReportGroupView({
  group,
  fallback,
  onOpenDetail,
  onOpenEvidence
}: {
  group: ReportHomeGroup;
  fallback: boolean;
  onOpenDetail: (route: ReportDetailRoute) => void;
  onOpenEvidence: (route: ReportEvidenceRoute) => void;
}) {
  return (
    <View className='home-report-group' ariaLabel={group.report.title}>
      <View className='home-report-group__header'>
        <Text className='home-report-group__title'>{group.report.title}</Text>
        <View className='home-report-group__meta'>
          <Text>发布于 {formatShanghaiTimestamp(group.report.publishedAt)}</Text>
          {fallback ? <Text className='home-report-group__fallback'>最近发布</Text> : null}
        </View>
      </View>

      <View className='home-report-cards'>
        {group.cards.map((card) => (
          <HomeReportCard
            key={card.key}
            reportId={group.report.id}
            card={card}
            onOpenDetail={onOpenDetail}
            onOpenEvidence={onOpenEvidence}
          />
        ))}
      </View>

      <View className='home-company-boundary'>
        <View className='home-company-boundary__mark' ariaLabel='企业层'>
          <Image className='home-company-boundary__icon' src={reportCubeIcon} mode='aspectFit' />
        </View>
        <View className='home-company-boundary__copy'>
          <Text>{group.company.title}层未发布</Text>
          <Text>{group.company.boundary}</Text>
        </View>
      </View>
    </View>
  );
}

function HomeReportCard({
  reportId,
  card,
  onOpenDetail,
  onOpenEvidence
}: {
  reportId: string;
  card: ReportCard;
  onOpenDetail: (route: ReportDetailRoute) => void;
  onOpenEvidence: (route: ReportEvidenceRoute) => void;
}) {
  const detailRoute: ReportDetailRoute = {
    reportId,
    targetType: card.detailRef.type,
    targetKey: card.detailRef.key
  };
  return (
    <View
      className={`home-report-card home-report-card--${card.kind}`}
      role='button'
      ariaLabel={`查看${card.title}推理详情`}
      onClick={() => onOpenDetail(detailRoute)}
    >
      <View className='home-report-card__heading'>
        <View className='home-report-card__identity'>
          <View className='home-report-card__kind' ariaLabel={`${cardKindLabel(card.kind)}卡片`}>
            <Image
              className='home-report-card__kind-icon'
              src={cardKindIcon(card.kind)}
              mode='aspectFit'
            />
          </View>
          <View>
            <Text className='home-report-card__title'>{card.title}</Text>
            <Text className='home-report-card__subtitle'>{card.subtitle}</Text>
          </View>
        </View>
        {card.hasEvidence ? (
          <ReportEvidenceButton
            label={`查看${card.title}证据`}
            onClick={() =>
              onOpenEvidence({
                reportId,
                scopeType: 'report_card',
                scopeKey: card.key,
                title: `${card.title}证据`
              })
            }
          />
        ) : null}
      </View>

      <Text className='home-report-card__conclusion'>{card.conclusion}</Text>
      <View className='home-report-card__signals'>
        <ReportImpactSignals
          result={card.result}
          confidence={card.confidence}
          timeWindow={card.timeWindow}
        />
      </View>
      <View className='home-impact-list'>
        {card.impactItems.map((item) => (
          <View className='home-impact-item' key={`${item.ref.type}:${item.ref.key}`}>
            <View className='home-impact-item__identity'>
              <Text className='home-impact-item__name'>{item.name}</Text>
              {item.hasEvidence ? (
                <ReportEvidenceButton
                  label={`查看${item.name}证据`}
                  onClick={() =>
                    onOpenEvidence({
                      reportId,
                      scopeType: item.ref.type,
                      scopeKey: item.ref.key,
                      title: `${item.name}证据`
                    })
                  }
                />
              ) : null}
            </View>
            <ReportImpactSignals
              result={item.result}
              confidence={item.confidence}
              timeWindow={item.timeWindow}
            />
          </View>
        ))}
      </View>
      <View className='home-report-card__footer'>
        <Text>推理详情</Text>
        <Image className='home-report-card__arrow' src={reportArrowRightIcon} mode='aspectFit' />
      </View>
    </View>
  );
}

function cardKindIcon(kind: ReportCard['kind']): string {
  if (kind === 'geopolitics') return reportGlobeIcon;
  if (kind === 'macroeconomics') return reportBarChartIcon;
  return reportLayersIcon;
}

function cardKindLabel(kind: ReportCard['kind']): string {
  if (kind === 'geopolitics') return '地缘政治';
  if (kind === 'macroeconomics') return '宏观经济';
  return '产业链';
}

function homeSummary(state: ReportResourceState<ReportHome>): string {
  if (state.status === 'ready') {
    if (state.data.selection.mode === 'latest_fallback') return '今日暂无 · 展示最近发布';
    return `${state.data.reports.length} 份报告`;
  }
  if (state.status === 'error') return '读取失败';
  if (state.status === 'empty') return '暂无发布';
  return '加载中';
}

export async function stopHomeRefresh(api: HomeRefreshAPI): Promise<void> {
  void api.stopPullDownRefresh();
}
