import Taro, { usePullDownRefresh } from '@tarojs/taro';
import { Button, Image, Text, View } from '@tarojs/components';
import { type ReactNode, useMemo, useState } from 'react';
import fileTextIcon from '../../assets/icons/file-text.svg';
import reportArrowRightIcon from '../../assets/icons/report-arrow-right-light.svg';
import reportActivityCoolingIcon from '../../assets/icons/report-activity-cooling.svg';
import reportActivityDivergingIcon from '../../assets/icons/report-activity-diverging.svg';
import reportActivityPendingIcon from '../../assets/icons/report-activity-pending.svg';
import reportActivityWarmingIcon from '../../assets/icons/report-activity-warming.svg';
import reportBarChartIcon from '../../assets/icons/report-bar-chart.svg';
import reportConfidenceIcon from '../../assets/icons/report-confidence.svg';
import reportCubeIcon from '../../assets/icons/report-cube.svg';
import reportGlobeIcon from '../../assets/icons/report-globe.svg';
import reportLayersIcon from '../../assets/icons/report-layers.svg';
import reportPublishedClockIcon from '../../assets/icons/report-clock.svg';
import reportWindowClockIcon from '../../assets/icons/report-window-clock.svg';
import type {
  ReportCard,
  ReportHome,
  ReportHomeGroup,
  ReportImpactItem,
  ReportResultCode
} from '../../features/reports/contract';
import {
  navigateToReportDetail,
  type ReportDetailRoute,
  type ReportEvidenceRoute
} from '../../features/reports/navigation';
import { getReportPort } from '../../features/reports/port';
import { formatShanghaiTimestamp, reportErrorCopy } from '../../features/reports/presentation';
import { ReportStatePanel } from '../../features/reports/report-components';
import type { ReportResourceState } from '../../features/reports/session';
import { useReportResource } from '../../features/reports/use-report-resource';
import { getHomeChromeMetrics, type HomeChromeMetrics } from '../../platform/system-ui';
import { HomeHeader } from './components/home-header';
import { HomeReportEvidenceSheet } from './components/report-evidence-sheet';
import './index.scss';

interface HomeRefreshAPI {
  stopPullDownRefresh: () => unknown;
  showToast: (options: { title: string; icon: 'none'; duration: number }) => unknown;
}

const isHomeEmpty = (home: ReportHome) => home.reports.length === 0;

export default function IndexPage() {
  const [query, setQuery] = useState('');
  const [evidenceRoute, setEvidenceRoute] = useState<ReportEvidenceRoute | null>(null);
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
    <>
      <IndexView
        chrome={chrome}
        query={query}
        onQueryChange={setQuery}
        state={resource.state}
        onRetry={() => void resource.retry()}
        onOpenDetail={(route) => navigateToReportDetail(Taro, route)}
        onOpenEvidence={setEvidenceRoute}
      />
      {evidenceRoute ? (
        <HomeReportEvidenceSheet
          route={evidenceRoute}
          port={port}
          onClose={() => setEvidenceRoute(null)}
        />
      ) : null}
    </>
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
  return (
    <View className='home-page'>
      <HomeHeader chrome={chrome} query={query} onQueryChange={onQueryChange} />

      <View className='home-content'>
        <View className='home-section-heading'>
          <Text className='home-section-heading__title'>今日观潮</Text>
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
  const geopolitics = group.cards.find((card) => card.kind === 'geopolitics');
  const macroeconomics = group.cards.find((card) => card.kind === 'macroeconomics');
  const industryCards = group.cards.filter((card) => card.kind === 'industry_chain');
  const industryChainCount = group.industryChainCount;

  return (
    <View className='home-report-group' ariaLabel='本期观潮报告'>
      <View className='home-report-group__header'>
        <View className='home-report-publish-row'>
          <Image
            className='home-report-publish-row__icon'
            src={reportPublishedClockIcon}
            mode='aspectFit'
          />
          <Text className='home-report-publish-row__label'>发布时间</Text>
          <Text className='home-report-publish-row__time'>
            {formatShanghaiTimestamp(group.report.publishedAt)}
          </Text>
        </View>
        {fallback ? <Text className='home-report-group__fallback'>最近发布</Text> : null}
      </View>

      <View className='home-report-flow'>
        {geopolitics ? (
          <HomeReportSection card={geopolitics}>
            <HomeReportCard
              reportId={group.report.id}
              card={geopolitics}
              onOpenDetail={onOpenDetail}
              onOpenEvidence={onOpenEvidence}
            />
          </HomeReportSection>
        ) : null}

        {macroeconomics ? (
          <HomeReportSection card={macroeconomics}>
            <HomeReportCard
              reportId={group.report.id}
              card={macroeconomics}
              onOpenDetail={onOpenDetail}
              onOpenEvidence={onOpenEvidence}
            />
          </HomeReportSection>
        ) : null}

        {industryCards.length > 0 ? (
          <View className='home-report-section'>
            <HomeReportSectionHeading
              kind='industry_chain'
              title='产业链'
              subtitle={`${industryChainCount} 条真实产业链 · 首页展示 ${industryCards.length} 条`}
            />
            <View className='home-industry-list'>
              {industryCards.map((card) => (
                <HomeReportCard
                  key={card.key}
                  reportId={group.report.id}
                  card={card}
                  onOpenDetail={onOpenDetail}
                  onOpenEvidence={onOpenEvidence}
                />
              ))}
            </View>
          </View>
        ) : null}

        <View className='home-report-section home-report-section--company'>
          <HomeReportSectionHeading
            kind='company'
            title={group.company.title}
            subtitle='本次报告未发布企业层结论'
          />
          <View className='home-company-boundary'>
            <View className='home-company-boundary__mark' ariaLabel='企业层'>
              <Image
                className='home-company-boundary__icon'
                src={reportCubeIcon}
                mode='aspectFit'
              />
            </View>
            <View className='home-company-boundary__copy'>
              <Text>本次推理尚未进入企业层</Text>
              <Text>{group.company.boundary}</Text>
            </View>
          </View>
        </View>
      </View>
    </View>
  );
}

function HomeReportSection({ card, children }: { card: ReportCard; children: ReactNode }) {
  return (
    <View className='home-report-section'>
      <HomeReportSectionHeading kind={card.kind} title={card.title} subtitle={card.subtitle} />
      {children}
    </View>
  );
}

function HomeReportSectionHeading({
  kind,
  title,
  subtitle
}: {
  kind: ReportCard['kind'] | 'company';
  title: string;
  subtitle: string;
}) {
  return (
    <View className='home-report-section__heading'>
      <View className='home-report-section__kind' ariaLabel={`${title}层`}>
        <Image
          className='home-report-section__kind-icon'
          src={reportSectionIcon(kind)}
          mode='aspectFit'
        />
      </View>
      <View className='home-report-section__heading-copy'>
        <Text className='home-report-section__title'>{title}</Text>
        <Text className='home-report-section__subtitle'>{subtitle}</Text>
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
    <View className={`home-report-card home-report-card--${card.kind}`} ariaLabel={card.title}>
      <Text className='home-report-card__conclusion'>{card.conclusion}</Text>
      {card.kind === 'industry_chain' ? (
        <View className='home-industry-identity'>
          <Text className='home-industry-identity__tag'>产业链</Text>
          <Text className='home-industry-identity__name'>{card.title}</Text>
        </View>
      ) : null}
      <View className='home-impact-list'>
        {card.impactItems.map((item) => (
          <View className='home-impact-item' key={`${item.ref.type}:${item.ref.key}`}>
            <Text className='home-impact-item__name'>{item.name}</Text>
            <HomeImpactSignals item={item} />
          </View>
        ))}
      </View>
      <View className='home-report-card__actions'>
        {card.hasEvidence ? (
          <Button
            className='tidewise-button home-card-evidence-action'
            hoverClass='home-card-evidence-action--pressed'
            ariaLabel={`查看${card.title}依据`}
            onClick={(event) => {
              event.stopPropagation();
              onOpenEvidence({
                reportId,
                scopeType: 'report_card',
                scopeKey: card.key,
                title: `${card.title}证据`
              });
            }}
          >
            <Image
              className='home-card-evidence-action__icon'
              src={fileTextIcon}
              mode='aspectFit'
            />
            <Text>依据</Text>
          </Button>
        ) : (
          <View />
        )}
        <Button
          className='tidewise-button home-card-detail-action'
          hoverClass='home-card-detail-action--pressed'
          ariaLabel={`查看${card.title}传导详情`}
          onClick={() => onOpenDetail(detailRoute)}
        >
          <Text>看传导</Text>
          <Image className='home-report-card__arrow' src={reportArrowRightIcon} mode='aspectFit' />
        </Button>
      </View>
    </View>
  );
}

function HomeImpactSignals({ item }: { item: ReportImpactItem }) {
  return (
    <View
      className='home-impact-signals'
      ariaLabel={`结果${item.result.label}，置信度${item.confidence.label}，时间窗口${item.timeWindow}`}
    >
      <View
        className={`home-impact-signal home-impact-signal--result home-impact-signal--${item.result.code}`}
      >
        <Image
          className='home-impact-signal__result-icon'
          src={reportActivityIcon(item.result.code)}
          mode='aspectFit'
        />
        <Text className='home-impact-signal__value'>{item.result.label}</Text>
      </View>
      <View className='home-impact-signal home-impact-signal--confidence'>
        <Image
          className='home-impact-signal__confidence-icon'
          src={reportConfidenceIcon}
          mode='aspectFit'
        />
        <Text>置信</Text>
        <Text className='home-impact-signal__value'>{item.confidence.label}</Text>
      </View>
      <View className='home-impact-signal home-impact-signal--window'>
        <Image
          className='home-impact-signal__window-icon'
          src={reportWindowClockIcon}
          mode='aspectFit'
        />
        <Text className='home-impact-signal__value'>{item.timeWindow}</Text>
      </View>
    </View>
  );
}

function reportSectionIcon(kind: ReportCard['kind'] | 'company'): string {
  if (kind === 'geopolitics') return reportGlobeIcon;
  if (kind === 'macroeconomics') return reportBarChartIcon;
  if (kind === 'industry_chain') return reportLayersIcon;
  return reportCubeIcon;
}

function reportActivityIcon(result: ReportResultCode): string {
  if (result === 'warming') return reportActivityWarmingIcon;
  if (result === 'cooling') return reportActivityCoolingIcon;
  if (result === 'diverging') return reportActivityDivergingIcon;
  return reportActivityPendingIcon;
}

export async function stopHomeRefresh(api: HomeRefreshAPI): Promise<void> {
  void api.stopPullDownRefresh();
}
