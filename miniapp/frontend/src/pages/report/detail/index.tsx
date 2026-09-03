import Taro, { usePullDownRefresh } from '@tarojs/taro';
import { Button, Image, ScrollView, Text, View } from '@tarojs/components';
import { useEffect, useMemo, useState } from 'react';
import reportActivityCoolingIcon from '../../../assets/icons/report-activity-cooling.svg';
import reportActivityDivergingIcon from '../../../assets/icons/report-activity-diverging.svg';
import reportActivityPendingIcon from '../../../assets/icons/report-activity-pending.svg';
import reportActivityWarmingIcon from '../../../assets/icons/report-activity-warming.svg';
import reportBarChartIcon from '../../../assets/icons/report-bar-chart.svg';
import reportArrowRightIcon from '../../../assets/icons/report-arrow-right.svg';
import reportConfidenceIcon from '../../../assets/icons/report-confidence.svg';
import reportGlobeIcon from '../../../assets/icons/report-globe.svg';
import reportInfoIcon from '../../../assets/icons/report-info.svg';
import reportLayersIcon from '../../../assets/icons/report-layers.svg';
import reportLinkIcon from '../../../assets/icons/report-link.svg';
import reportWarningIcon from '../../../assets/icons/report-warning.svg';
import reportWindowClockIcon from '../../../assets/icons/report-window-clock.svg';
import type {
  ReportIndustryChainDetail,
  ReportIndustryChainNode,
  ReportGraphNode,
  ReportLayerDetail,
  ReportLayerKey,
  ReportPort,
  ReportTransmissionPath
} from '../../../features/reports/contract';
import { ReportError } from '../../../features/reports/contract';
import {
  navigateToReportDetail,
  parseReportDetailRoute,
  type ReportDetailRoute,
  type ReportEvidenceRoute
} from '../../../features/reports/navigation';
import { getReportPort } from '../../../features/reports/port';
import { reportErrorCopy } from '../../../features/reports/presentation';
import { ReportEvidenceSheet } from '../../../features/reports/report-evidence-sheet';
import {
  ReportEvidenceButton,
  ReportImpactSignals,
  ReportStatePanel
} from '../../../features/reports/report-components';
import type { ReportResourceState } from '../../../features/reports/session';
import { useReportResource } from '../../../features/reports/use-report-resource';
import './index.scss';

export type LoadedReportDetail =
  | { targetType: 'layer'; detail: ReportLayerDetail; continuationDetail?: ReportLayerDetail }
  | { targetType: 'industry_chain'; detail: ReportIndustryChainDetail };

export default function ReportDetailPage() {
  const instance = useMemo(() => Taro.getCurrentInstance(), []);
  const route = useMemo(() => safeDetailRoute(instance.router?.params), [instance]);
  const port = useMemo(() => getReportPort(), []);
  const [evidenceRoute, setEvidenceRoute] = useState<ReportEvidenceRoute | null>(null);
  const resource = useReportResource(
    `report-detail:${route?.reportId ?? 'invalid'}:${route?.targetType ?? 'invalid'}:${route?.targetKey ?? 'invalid'}`,
    () => loadReportDetail(port, route)
  );

  useEffect(() => {
    void Taro.setNavigationBarTitle({ title: '推理详情' });
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

  return (
    <>
      <ReportDetailView
        state={resource.state}
        onRetry={() => void resource.retry()}
        onOpenDetail={(targetRoute) => navigateToReportDetail(Taro, targetRoute)}
        onOpenEvidence={setEvidenceRoute}
      />
      {evidenceRoute ? (
        <ReportEvidenceSheet
          route={evidenceRoute}
          port={port}
          onClose={() => setEvidenceRoute(null)}
        />
      ) : null}
    </>
  );
}

export async function loadReportDetail(
  port: ReportPort,
  route: ReportDetailRoute | null
): Promise<LoadedReportDetail> {
  if (!route) throw new ReportError('invalidRequest');
  if (route.targetType === 'layer') {
    if (route.targetKey === 'geopolitics') {
      const [detail, continuationDetail] = await Promise.all([
        port.getLayer(route.reportId, 'geopolitics'),
        optionalLayer(port, route.reportId, 'macroeconomics')
      ]);
      return continuationDetail
        ? { targetType: 'layer', detail, continuationDetail }
        : { targetType: 'layer', detail };
    }
    const detail = await port.getLayer(route.reportId, route.targetKey as ReportLayerKey);
    return { targetType: 'layer', detail };
  }
  const detail = await port.getIndustryChain(route.reportId, route.targetKey);
  return { targetType: 'industry_chain', detail };
}

async function optionalLayer(
  port: ReportPort,
  reportId: string,
  layerKey: ReportLayerKey
): Promise<ReportLayerDetail | undefined> {
  try {
    return await port.getLayer(reportId, layerKey);
  } catch (error) {
    if (error instanceof ReportError && error.kind === 'layerUnavailable') return undefined;
    throw error;
  }
}

export function ReportDetailView({
  state,
  onRetry,
  onOpenDetail,
  onOpenEvidence
}: {
  state: ReportResourceState<LoadedReportDetail>;
  onRetry: () => void;
  onOpenDetail: (route: ReportDetailRoute) => void;
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
    <View className='report-detail-page'>
      {state.refreshFailed ? (
        <View className='report-detail-refresh-warning'>刷新失败，当前展示上次成功读取的内容</View>
      ) : null}
      {state.data.targetType === 'layer' ? (
        <LayerDetailView
          detail={state.data.detail}
          continuationDetail={state.data.continuationDetail}
          onOpenDetail={onOpenDetail}
          onOpenEvidence={onOpenEvidence}
        />
      ) : (
        <IndustryChainDetailView detail={state.data.detail} onOpenEvidence={onOpenEvidence} />
      )}
    </View>
  );
}

function LayerDetailView({
  detail,
  continuationDetail,
  onOpenDetail,
  onOpenEvidence
}: {
  detail: ReportLayerDetail;
  continuationDetail?: ReportLayerDetail;
  onOpenDetail: (route: ReportDetailRoute) => void;
  onOpenEvidence: (route: ReportEvidenceRoute) => void;
}) {
  const { layer, report } = detail;
  const layerIcon = layer.key === 'geopolitics' ? reportGlobeIcon : reportBarChartIcon;
  return (
    <View className='report-detail-flow'>
      <View className='report-layer-panel'>
        <View className='report-layer-heading'>
          <View className='report-layer-heading__identity'>
            <View className='report-layer-heading__icon-box'>
              <Image className='report-layer-heading__icon' src={layerIcon} mode='aspectFit' />
            </View>
            <View>
              <Text className='report-detail-eyebrow'>{layer.title}</Text>
              <Text className='report-layer-heading__title'>{layer.title}层推导</Text>
            </View>
          </View>
          {layer.evidenceScopeToken ? (
            <ScopeEvidenceButton
              reportId={report.id}
              scopeToken={layer.evidenceScopeToken}
              title={`${layer.title}证据`}
              onOpen={onOpenEvidence}
            />
          ) : null}
        </View>

        <View className='report-conclusion-band'>
          <Text className='report-conclusion-band__label'>一句话结论</Text>
          <Text className='report-conclusion-band__copy'>{layer.conclusion}</Text>
        </View>

        <SectionHeading title='影响锚点' />
        <View className='report-anchor-list'>
          {layer.anchors.map((anchor) => (
            <View className='report-anchor-card' key={anchor.key}>
              <View className='report-anchor-card__top'>
                <Text className='report-anchor-card__name'>{anchor.name}</Text>
                {hasDirectEvidence(anchor.evidenceScopeToken, anchor.conclusionBasis?.code) ? (
                  <ScopeEvidenceTextButton
                    reportId={report.id}
                    scopeToken={anchor.evidenceScopeToken!}
                    title={`${anchor.name}证据`}
                    label='依据'
                    onOpen={onOpenEvidence}
                  />
                ) : null}
              </View>
              <ReportImpactSignals
                result={anchor.result}
                confidence={anchor.confidence}
                timeWindow={anchor.timeWindow}
                conclusionBasis={anchor.conclusionBasis}
                validationStatus={anchor.validationStatus}
              />
              <Text className='report-anchor-card__state'>{anchor.currentState}</Text>
              <View className='report-anchor-card__reason'>
                <Text>为什么</Text>
                <Text>{anchor.reasoning}</Text>
              </View>
            </View>
          ))}
        </View>

        {layer.uncertainty.reversalCondition ? (
          <View className='report-reversal-card'>
            <View className='report-reversal-card__icon-box'>
              <Image
                className='report-reversal-card__icon'
                src={reportLayersIcon}
                mode='aspectFit'
              />
            </View>
            <View>
              <Text className='report-reversal-card__title'>反转条件</Text>
              <Text className='report-reversal-card__copy'>
                {layer.uncertainty.reversalCondition}
              </Text>
            </View>
          </View>
        ) : null}

        <SectionHeading title='向下传导' />
        <View className='report-transmission-list'>
          {layer.transmissions.map((path) => (
            <TransmissionPathView
              key={path.key}
              reportId={report.id}
              path={path}
              onOpenDetail={onOpenDetail}
            />
          ))}
        </View>
      </View>

      {continuationDetail ? (
        <>
          <View className='report-detail-continuation' ariaLabel='继续看宏观经济'>
            <Image
              className='report-detail-continuation__arrow'
              src={reportArrowRightIcon}
              mode='aspectFit'
            />
            <Text>继续看宏观经济</Text>
          </View>
          <LayerDetailView
            detail={continuationDetail}
            onOpenDetail={onOpenDetail}
            onOpenEvidence={onOpenEvidence}
          />
        </>
      ) : (
        <View className='report-related-chain-panel'>
          <View className='report-related-chain-panel__heading'>
            <View>
              <Text className='report-related-chain-panel__eyebrow'>INDUSTRY CHAINS</Text>
              <Text className='report-related-chain-panel__title'>产业链</Text>
            </View>
            <Text className='report-related-chain-panel__count'>
              {detail.relatedIndustryChains.length} 条
            </Text>
          </View>
          <View className='report-related-chain-list'>
            {detail.relatedIndustryChains.map((chainItem) => (
              <View
                className='report-related-chain-item'
                key={chainItem.key}
                role='button'
                ariaLabel={`查看${chainItem.name}推理详情`}
                onClick={() =>
                  onOpenDetail({
                    reportId: report.id,
                    targetType: 'industry_chain',
                    targetKey: chainItem.key
                  })
                }
              >
                <Text>{chainItem.name}</Text>
                <Text
                  className={`report-result-chip report-result-chip--${resultStyle(chainItem.result.code)}`}
                >
                  {chainItem.result.label}
                </Text>
                <Image
                  className='report-related-chain-item__arrow'
                  src={reportArrowRightIcon}
                  mode='aspectFit'
                />
              </View>
            ))}
          </View>
        </View>
      )}
    </View>
  );
}

function TransmissionPathView({
  reportId,
  path,
  onOpenDetail
}: {
  reportId: string;
  path: ReportTransmissionPath;
  onOpenDetail: (route: ReportDetailRoute) => void;
}) {
  const target = path.targets[0];
  const targetRef = target?.ref;
  const isDetailTarget = targetRef?.type === 'layer' || targetRef?.type === 'industry_chain';
  return (
    <View className='report-transmission-card'>
      <View
        className={`report-transmission-card__heading ${isDetailTarget ? 'is-clickable' : ''}`}
        role={isDetailTarget ? 'button' : undefined}
        ariaLabel={isDetailTarget ? `查看${target.name}推理详情` : undefined}
        onClick={
          isDetailTarget
            ? () =>
                onOpenDetail({
                  reportId,
                  targetType: targetRef!.type as 'layer' | 'industry_chain',
                  targetKey: targetRef!.localKey
                })
            : undefined
        }
      >
        <Image
          className='report-transmission-card__link-icon'
          src={reportLinkIcon}
          mode='aspectFit'
        />
        <Text>传到{target?.name ?? '下游对象'}</Text>
        {target ? (
          <Text
            className={`report-result-chip report-result-chip--${resultStyle(target.result.code)}`}
          >
            {target.result.label}
          </Text>
        ) : null}
      </View>
      <Text className='report-transmission-card__source'>{path.sourceConclusion}</Text>
      <Text className='report-transmission-card__nature'>{path.kind.label}</Text>
      <Text className='report-transmission-card__logic'>{path.logic}</Text>
      <View className='report-transmission-card__status'>
        <Text>状态</Text>
        <Text>{path.status.label}</Text>
      </View>
    </View>
  );
}

function IndustryChainDetailView({
  detail,
  onOpenEvidence
}: {
  detail: ReportIndustryChainDetail;
  onOpenEvidence: (route: ReportEvidenceRoute) => void;
}) {
  const { report, industryChain } = detail;
  const [selectedNodeKey, setSelectedNodeKey] = useState(industryChain.nodes[0]?.key ?? '');
  useEffect(() => {
    setSelectedNodeKey(industryChain.nodes[0]?.key ?? '');
  }, [industryChain.key, industryChain.nodes]);
  const selectedNode =
    industryChain.nodes.find((nodeItem) => nodeItem.key === selectedNodeKey) ??
    industryChain.nodes[0];
  const chainResultIcon =
    (
      {
        warming: reportActivityWarmingIcon,
        cooling: reportActivityCoolingIcon,
        diverging: reportActivityDivergingIcon,
        stable: reportActivityPendingIcon,
        mixed: reportActivityDivergingIcon,
        pending: reportActivityPendingIcon
      } as Record<string, string>
    )[industryChain.result.code] ?? reportActivityPendingIcon;

  return (
    <View className='report-detail-flow'>
      <View className='report-chain-panel'>
        <View className='report-layer-heading'>
          <View className='report-layer-heading__identity'>
            <View className='report-layer-heading__icon-box'>
              <Image
                className='report-layer-heading__icon'
                src={reportLayersIcon}
                mode='aspectFit'
              />
            </View>
            <View>
              <Text className='report-detail-eyebrow'>产业链</Text>
              <Text className='report-layer-heading__title'>{industryChain.name}</Text>
            </View>
          </View>
          {industryChain.evidenceScopeToken ? (
            <ScopeEvidenceButton
              reportId={report.id}
              scopeToken={industryChain.evidenceScopeToken}
              title={`${industryChain.name}证据`}
              onOpen={onOpenEvidence}
            />
          ) : null}
        </View>

        <View className='report-conclusion-band report-conclusion-band--chain'>
          <Text className='report-conclusion-band__label'>一句话结论</Text>
          <Text className='report-conclusion-band__copy'>{industryChain.conclusion}</Text>
        </View>

        <View className='report-chain-metrics'>
          <View
            className={`report-chain-metric report-chain-metric--result report-chain-metric--${resultStyle(industryChain.result.code)}`}
          >
            <Image
              className='report-chain-metric__icon report-chain-metric__icon--result'
              src={chainResultIcon}
              mode='aspectFit'
            />
            <View className='report-chain-metric__copy'>
              <Text className='report-chain-metric__label'>链结果</Text>
              <Text className='report-chain-metric__value'>{industryChain.result.label}</Text>
            </View>
          </View>
          <View className='report-chain-metric report-chain-metric--window'>
            <Image
              className='report-chain-metric__icon'
              src={reportWindowClockIcon}
              mode='aspectFit'
            />
            <View className='report-chain-metric__copy'>
              <Text className='report-chain-metric__label'>时间窗口</Text>
              <Text className='report-chain-metric__value'>{industryChain.timeWindow.label}</Text>
            </View>
          </View>
          <View className='report-chain-metric report-chain-metric--confidence'>
            <Image
              className='report-chain-metric__icon'
              src={reportConfidenceIcon}
              mode='aspectFit'
            />
            <View className='report-chain-metric__copy'>
              <Text className='report-chain-metric__label'>置信度</Text>
              <Text className='report-chain-metric__value'>{industryChain.confidence.label}</Text>
            </View>
          </View>
        </View>

        <SectionHeading title='产业链图' aside='横向滑动 · 点击节点查看详情' />
        <ChainGraph
          topologyNodes={industryChain.topologyNodes}
          nodes={industryChain.nodes}
          edges={industryChain.edges}
          selectedNodeKey={selectedNodeKey}
          onSelect={setSelectedNodeKey}
        />

        {selectedNode ? (
          <View className='report-node-detail'>
            <View className='report-node-detail__heading'>
              <Text>{selectedNode.name}</Text>
              {hasDirectEvidence(
                selectedNode.evidenceScopeToken,
                selectedNode.conclusionBasis?.code
              ) ? (
                <ScopeEvidenceTextButton
                  reportId={report.id}
                  scopeToken={selectedNode.evidenceScopeToken!}
                  title={`${selectedNode.name}证据`}
                  label='依据'
                  onOpen={onOpenEvidence}
                />
              ) : null}
            </View>
            <ReportImpactSignals
              result={selectedNode.result}
              confidence={selectedNode.confidence}
              timeWindow={selectedNode.timeWindow}
              conclusionBasis={selectedNode.conclusionBasis}
              validationStatus={selectedNode.validationStatus}
            />
            <View className='report-detail-fact'>
              <Text>本次影响</Text>
              <Text>{selectedNode.impact}</Text>
            </View>
            <View className='report-detail-fact'>
              <Text>传导逻辑</Text>
              <Text>{selectedNode.reasoning}</Text>
            </View>
            {!hasDirectEvidence(
              selectedNode.evidenceScopeToken,
              selectedNode.conclusionBasis?.code
            ) ? (
              <Text className='report-node-detail__evidence-note'>暂无直接证据，待后续验证</Text>
            ) : null}
          </View>
        ) : null}

        {industryChain.counterevidenceAndGap ? (
          <View className='report-chain-boundary report-chain-boundary--gap'>
            <Image
              className='report-chain-boundary__icon report-chain-boundary__icon--gap'
              src={reportInfoIcon}
              mode='aspectFit'
            />
            <View className='report-chain-boundary__copy'>
              <Text>反证与缺口</Text>
              <Text>{industryChain.counterevidenceAndGap}</Text>
            </View>
          </View>
        ) : null}
        {industryChain.stopCondition ? (
          <View className='report-chain-boundary report-chain-boundary--stop'>
            <Image
              className='report-chain-boundary__icon report-chain-boundary__icon--stop'
              src={reportWarningIcon}
              mode='aspectFit'
            />
            <View className='report-chain-boundary__copy'>
              <Text>停止条件</Text>
              <Text>{industryChain.stopCondition}</Text>
            </View>
          </View>
        ) : null}
      </View>
    </View>
  );
}

function ChainGraph({
  topologyNodes,
  nodes,
  edges,
  selectedNodeKey,
  onSelect
}: {
  topologyNodes: ReportGraphNode[];
  nodes: ReportIndustryChainNode[];
  edges: ReportIndustryChainDetail['industryChain']['edges'];
  selectedNodeKey: string;
  onSelect: (key: string) => void;
}) {
  const nodeWidth = 276;
  const nodeGap = 56;
  const canvasPadding = 28;
  const step = nodeWidth + nodeGap;
  const nodeIndexes = new Map(topologyNodes.map((item, index) => [item.key, index]));
  const assessments = new Map(nodes.map((item) => [item.key, item]));
  const longEdges = edges.filter((edgeItem) => {
    const from = nodeIndexes.get(edgeItem.fromNodeLocalKey);
    const to = nodeIndexes.get(edgeItem.toNodeLocalKey);
    return from !== undefined && to !== undefined && Math.abs(from - to) > 1;
  });
  const laneHeight = 52 + longEdges.length * 34;
  const canvasWidth =
    canvasPadding * 2 +
    topologyNodes.length * nodeWidth +
    Math.max(0, topologyNodes.length - 1) * nodeGap;

  return (
    <ScrollView className='report-chain-scroll' scrollX>
      <View
        className='report-chain-canvas'
        style={graphStyle([
          ['width', canvasWidth],
          ['padding-top', laneHeight]
        ])}
      >
        {edges.map((edgeItem) => {
          const fromIndex = nodeIndexes.get(edgeItem.fromNodeLocalKey);
          const toIndex = nodeIndexes.get(edgeItem.toNodeLocalKey);
          if (fromIndex === undefined || toIndex === undefined) return null;
          const adjacent = Math.abs(fromIndex - toIndex) === 1;
          const movesRight = fromIndex < toIndex;
          const nodeTop = laneHeight;
          if (adjacent) {
            const sourceX = canvasPadding + fromIndex * step + (movesRight ? nodeWidth : 0);
            const targetX = canvasPadding + toIndex * step + (movesRight ? 0 : nodeWidth);
            return (
              <View key={`${edgeItem.fromNodeLocalKey}:${edgeItem.toNodeLocalKey}`}>
                <View
                  className='report-chain-edge report-chain-edge--adjacent'
                  style={graphStyle([
                    ['top', nodeTop + 78],
                    ['left', Math.min(sourceX, targetX)],
                    ['width', Math.abs(targetX - sourceX)]
                  ])}
                >
                  <Text>{edgeItem.relationLabel}</Text>
                </View>
                <View
                  className={`report-chain-edge-arrow ${movesRight ? 'is-right' : 'is-left'}`}
                  style={graphStyle([
                    ['top', nodeTop + 70],
                    ['left', targetX - 7]
                  ])}
                />
              </View>
            );
          }
          const sourceX = canvasPadding + fromIndex * step + nodeWidth / 2;
          const targetX = canvasPadding + toIndex * step + nodeWidth / 2;
          const lane = longEdges.findIndex(
            (candidate) =>
              candidate.fromNodeLocalKey === edgeItem.fromNodeLocalKey &&
              candidate.toNodeLocalKey === edgeItem.toNodeLocalKey
          );
          const laneTop = 18 + lane * 34;
          return (
            <View key={`${edgeItem.fromNodeLocalKey}:${edgeItem.toNodeLocalKey}`}>
              <View
                className='report-chain-edge report-chain-edge--long'
                style={graphStyle([
                  ['top', laneTop],
                  ['left', Math.min(sourceX, targetX)],
                  ['width', Math.abs(targetX - sourceX)]
                ])}
              >
                <Text>{edgeItem.relationLabel}</Text>
              </View>
              <View
                className='report-chain-edge-vertical'
                style={graphStyle([
                  ['top', laneTop],
                  ['left', sourceX],
                  ['height', nodeTop - laneTop]
                ])}
              />
              <View
                className='report-chain-edge-vertical'
                style={graphStyle([
                  ['top', laneTop],
                  ['left', targetX],
                  ['height', nodeTop - laneTop]
                ])}
              />
              <View
                className='report-chain-edge-arrow is-down'
                style={graphStyle([
                  ['top', nodeTop - 8],
                  ['left', targetX - 7]
                ])}
              />
            </View>
          );
        })}

        <View className='report-chain-node-row'>
          {topologyNodes.map((topologyNode) => {
            const assessment = assessments.get(topologyNode.key);
            return (
              <Button
                className={`tidewise-button report-chain-node ${assessment && selectedNodeKey === assessment.key ? 'is-selected' : ''}`}
                hoverClass='report-chain-node--pressed'
                key={topologyNode.key}
                disabled={!assessment}
                ariaLabel={
                  assessment
                    ? `查看${topologyNode.name}节点详情`
                    : `${topologyNode.name}暂无本期评估`
                }
                onClick={() => {
                  if (assessment) onSelect(assessment.key);
                }}
              >
                <Text className='report-chain-node__name'>{topologyNode.name}</Text>
                {assessment ? (
                  <ReportImpactSignals
                    result={assessment.result}
                    confidence={assessment.confidence}
                    timeWindow={assessment.timeWindow}
                    conclusionBasis={assessment.conclusionBasis}
                    validationStatus={assessment.validationStatus}
                  />
                ) : (
                  <Text className='report-chain-node__unassessed'>暂无本期评估</Text>
                )}
              </Button>
            );
          })}
        </View>
      </View>
    </ScrollView>
  );
}

function graphStyle(declarations: Array<[property: string, value: number]>): string {
  return declarations
    .map(([property, value]) => `${property}:${Taro.pxTransform(value)}`)
    .join(';');
}

function resetPageScroll(): void {
  void Taro.pageScrollTo({ scrollTop: 0, duration: 0 });
  if (process.env.TARO_ENV === 'h5' && typeof window !== 'undefined') {
    window.scrollTo(0, 0);
  }
}

function ScopeEvidenceButton({
  reportId,
  scopeToken,
  title,
  onOpen
}: {
  reportId: string;
  scopeToken: string;
  title: string;
  onOpen: (route: ReportEvidenceRoute) => void;
}) {
  return (
    <ReportEvidenceButton
      label={`查看${title}`}
      onClick={() =>
        onOpen({
          reportId,
          scopeToken,
          title
        })
      }
    />
  );
}

function ScopeEvidenceTextButton({
  reportId,
  scopeToken,
  title,
  label,
  onOpen
}: {
  reportId: string;
  scopeToken: string;
  title: string;
  label: string;
  onOpen: (route: ReportEvidenceRoute) => void;
}) {
  return (
    <Button
      className='tidewise-button report-evidence-text-button report-evidence-text-button--outline'
      hoverClass='report-evidence-text-button--pressed'
      ariaLabel={`查看${title}：${label}`}
      onClick={(event) => {
        event.stopPropagation();
        onOpen({
          reportId,
          scopeToken,
          title
        });
      }}
    >
      <Text>{label}</Text>
    </Button>
  );
}

function hasDirectEvidence(scopeToken: string | null, conclusionBasisCode?: string): boolean {
  return scopeToken !== null && conclusionBasisCode === 'direct_evidence';
}

function resultStyle(result: string): string {
  return ['warming', 'cooling', 'diverging', 'stable', 'mixed', 'pending'].includes(result)
    ? result
    : 'pending';
}

function SectionHeading({ title, aside }: { title: string; aside?: string }) {
  return (
    <View className='report-detail-section__heading'>
      <Text>{title}</Text>
      {aside ? <Text>{aside}</Text> : null}
    </View>
  );
}

function safeDetailRoute(value: unknown): ReportDetailRoute | null {
  try {
    return parseReportDetailRoute(value);
  } catch {
    return null;
  }
}
