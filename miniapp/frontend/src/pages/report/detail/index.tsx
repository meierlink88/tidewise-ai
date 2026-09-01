import Taro, { usePullDownRefresh } from '@tarojs/taro';
import { Button, Image, ScrollView, Text, View } from '@tarojs/components';
import { useEffect, useMemo, useState, type ReactNode } from 'react';
import reportArrowRightIcon from '../../../assets/icons/report-arrow-right.svg';
import type {
  ReportEvidenceScope,
  ReportIndustryChainDetail,
  ReportIndustryChainNode,
  ReportLayerDetail,
  ReportLayerKey,
  ReportPort,
  ReportTransmissionPath,
  ReportTransmissionTarget
} from '../../../features/reports/contract';
import { ReportError } from '../../../features/reports/contract';
import {
  navigateToReportDetail,
  navigateToReportEvidences,
  parseReportDetailRoute,
  type ReportDetailRoute,
  type ReportEvidenceRoute
} from '../../../features/reports/navigation';
import { getReportPort } from '../../../features/reports/port';
import { formatShanghaiTimestamp, reportErrorCopy } from '../../../features/reports/presentation';
import {
  ReportEvidenceButton,
  ReportImpactSignals,
  ReportStatePanel
} from '../../../features/reports/report-components';
import type { ReportResourceState } from '../../../features/reports/session';
import { useReportResource } from '../../../features/reports/use-report-resource';
import './index.scss';

export type LoadedReportDetail =
  | { targetType: 'layer'; detail: ReportLayerDetail }
  | { targetType: 'industry_chain'; detail: ReportIndustryChainDetail };

export default function ReportDetailPage() {
  const instance = useMemo(() => Taro.getCurrentInstance(), []);
  const route = useMemo(() => safeDetailRoute(instance.router?.params), [instance]);
  const port = useMemo(() => getReportPort(), []);
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
    if (
      (latest.status === 'ready' || latest.status === 'empty') &&
      latest.refreshFailed
    ) {
      void Taro.showToast({ title: '刷新失败，已保留当前内容', icon: 'none', duration: 1800 });
    }
    void Taro.stopPullDownRefresh();
  });

  return (
    <ReportDetailView
      state={resource.state}
      onRetry={() => void resource.retry()}
      onOpenDetail={(targetRoute) => navigateToReportDetail(Taro, targetRoute)}
      onOpenEvidence={(evidenceRoute) => navigateToReportEvidences(Taro, evidenceRoute)}
    />
  );
}

export async function loadReportDetail(
  port: ReportPort,
  route: ReportDetailRoute | null
): Promise<LoadedReportDetail> {
  if (!route) throw new ReportError('invalidRequest');
  if (route.targetType === 'layer') {
    const detail = await port.getLayer(route.reportId, route.targetKey as ReportLayerKey);
    return { targetType: 'layer', detail };
  }
  const detail = await port.getIndustryChain(route.reportId, route.targetKey);
  return { targetType: 'industry_chain', detail };
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

  const report = state.data.detail.report;
  return (
    <View className='report-detail-page'>
      {state.refreshFailed ? (
        <View className='report-detail-refresh-warning'>刷新失败，当前展示上次成功读取的内容</View>
      ) : null}
      <View className='report-detail-report-meta'>
        <Text>{report.title}</Text>
        <Text>发布于 {formatShanghaiTimestamp(report.publishedAt)}</Text>
      </View>
      {state.data.targetType === 'layer' ? (
        <LayerDetailView
          detail={state.data.detail}
          onOpenDetail={onOpenDetail}
          onOpenEvidence={onOpenEvidence}
        />
      ) : (
        <IndustryChainDetailView
          detail={state.data.detail}
          onOpenEvidence={onOpenEvidence}
        />
      )}
    </View>
  );
}

function LayerDetailView({
  detail,
  onOpenDetail,
  onOpenEvidence
}: {
  detail: ReportLayerDetail;
  onOpenDetail: (route: ReportDetailRoute) => void;
  onOpenEvidence: (route: ReportEvidenceRoute) => void;
}) {
  const { layer, report } = detail;
  return (
    <View className='report-detail-flow'>
      <View className='report-detail-hero'>
        <View className='report-detail-hero__heading'>
          <View>
            <Text className='report-detail-eyebrow'>{layer.title}</Text>
            <Text className='report-detail-hero__title'>一句话结论</Text>
          </View>
          {layer.hasEvidence ? (
            <ScopeEvidenceButton
              reportId={report.id}
              scope={layer.scope}
              title={`${layer.title}证据`}
              onOpen={onOpenEvidence}
            />
          ) : null}
        </View>
        <Text className='report-detail-hero__conclusion'>{layer.conclusion}</Text>
        <ReportImpactSignals
          result={layer.result}
          confidence={layer.confidence}
          timeWindow={layer.timeWindow}
        />
      </View>

      <DetailSection title='影响锚点'>
        <View className='report-anchor-list'>
          {layer.anchors.map((anchor) => (
            <View className='report-anchor-card' key={anchor.key}>
              <View className='report-anchor-card__top'>
                <Text className='report-anchor-card__name'>{anchor.name}</Text>
                {anchor.hasEvidence ? (
                  <ScopeEvidenceButton
                    reportId={report.id}
                    scope={anchor.scope}
                    title={`${anchor.name}证据`}
                    onOpen={onOpenEvidence}
                  />
                ) : null}
              </View>
              <ReportImpactSignals
                result={anchor.result}
                confidence={anchor.confidence}
                timeWindow={anchor.timeWindow}
                nature={anchor.nature}
              />
              <View className='report-detail-fact'>
                <Text>当前</Text>
                <Text>{anchor.currentState}</Text>
              </View>
              <View className='report-detail-fact'>
                <Text>推理</Text>
                <Text>{anchor.reasoning}</Text>
              </View>
            </View>
          ))}
        </View>
      </DetailSection>

      {layer.reasoningSteps.length ? (
        <DetailSection title='推理步骤'>
          <View className='report-step-list'>
            {layer.reasoningSteps.map((step) => (
              <View className='report-step-card' key={step.key}>
                <View className='report-step-card__heading'>
                  <Text>STEP {String(step.displayOrder).padStart(2, '0')}</Text>
                  <Text>{step.type}</Text>
                  {step.hasEvidence ? (
                    <ScopeEvidenceButton
                      reportId={report.id}
                      scope={step.scope}
                      title={`推理步骤 ${step.displayOrder} 证据`}
                      onOpen={onOpenEvidence}
                    />
                  ) : null}
                </View>
                <View className='report-step-card__path'>
                  <View>
                    <Text>输入</Text>
                    <Text>{step.input}</Text>
                  </View>
                  <Image
                    className='report-step-card__arrow'
                    src={reportArrowRightIcon}
                    mode='aspectFit'
                  />
                  <View>
                    <Text>输出</Text>
                    <Text>{step.output}</Text>
                  </View>
                </View>
                <View className='report-step-card__mechanism'>
                  <Text>传导机制</Text>
                  <Text>{step.mechanism}</Text>
                </View>
              </View>
            ))}
          </View>
        </DetailSection>
      ) : null}

      <DetailSection title='向下传导'>
        <Text className='report-detail-section-summary'>
          {layer.downwardTransmission.summary}
        </Text>
        <View className='report-transmission-list'>
          {layer.downwardTransmission.publishedPaths.map((path) => (
            <TransmissionPathView
              key={path.key}
              reportId={report.id}
              path={path}
              onOpenDetail={onOpenDetail}
              onOpenEvidence={onOpenEvidence}
            />
          ))}
        </View>
        {layer.downwardTransmission.candidateMechanisms.map((candidate) => (
          <View className='report-candidate-card' key={candidate.key}>
            <View className='report-candidate-card__heading'>
              <Text>待验证机制</Text>
              {candidate.hasEvidence ? (
                <ScopeEvidenceButton
                  reportId={report.id}
                  scope={candidate.scope}
                  title='待验证机制证据'
                  onOpen={onOpenEvidence}
                />
              ) : null}
            </View>
            <Text className='report-candidate-card__mechanism'>{candidate.mechanism}</Text>
            {candidate.evidenceGap ? (
              <Text className='report-candidate-card__gap'>{candidate.evidenceGap}</Text>
            ) : null}
            <Text className='report-candidate-card__confidence'>置信 {candidate.confidence.label}</Text>
          </View>
        ))}
        {layer.downwardTransmission.boundaryNotes.length ? (
          <View className='report-transmission-boundaries'>
            {layer.downwardTransmission.boundaryNotes.map((note) => (
              <Text key={note}>{note}</Text>
            ))}
          </View>
        ) : null}
      </DetailSection>

      <LayerUncertaintyView uncertainty={layer.uncertainty} />

      <DetailSection title='产业链'>
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
                  targetType: chainItem.detailRef.type,
                  targetKey: chainItem.detailRef.key
                })
              }
            >
              <Text>{chainItem.name}</Text>
              <Text
                className={`report-result-chip report-result-chip--${chainItem.result.code}`}
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
      </DetailSection>
    </View>
  );
}

function TransmissionPathView({
  reportId,
  path,
  onOpenDetail,
  onOpenEvidence
}: {
  reportId: string;
  path: ReportTransmissionPath;
  onOpenDetail: (route: ReportDetailRoute) => void;
  onOpenEvidence: (route: ReportEvidenceRoute) => void;
}) {
  return (
    <View className='report-transmission-card'>
      <View className='report-transmission-card__heading'>
        <Text>{path.relationNature}</Text>
        {path.hasEvidence ? (
          <ScopeEvidenceButton
            reportId={reportId}
            scope={path.scope}
            title='向下传导证据'
            onOpen={onOpenEvidence}
          />
        ) : null}
      </View>
      <Text className='report-transmission-card__source'>{path.sourceConclusion}</Text>
      <Text className='report-transmission-card__logic'>{path.logic}</Text>
      <View className='report-transmission-targets'>
        {path.targetRefs.map((target) => (
          <TransmissionTargetView
            key={`${target.ref.type}:${target.ref.key}`}
            reportId={reportId}
            target={target}
            confidence={path.confidence}
            onOpenDetail={onOpenDetail}
          />
        ))}
      </View>
      <View className='report-transmission-card__status'>
        <Text>边界</Text>
        <Text>{path.status}</Text>
      </View>
    </View>
  );
}

function TransmissionTargetView({
  reportId,
  target,
  confidence,
  onOpenDetail
}: {
  reportId: string;
  target: ReportTransmissionTarget;
  confidence: ReportTransmissionPath['confidence'];
  onOpenDetail: (route: ReportDetailRoute) => void;
}) {
  const isDetailTarget =
    target.ref.type === 'layer' || target.ref.type === 'industry_chain';
  const content = (
    <>
      <Text>{target.label}</Text>
      <ReportImpactSignals
        result={target.result}
        confidence={confidence}
        timeWindow={isDetailTarget ? '查看详情' : '报告对象'}
      />
      {isDetailTarget ? (
        <Image
          className='report-transmission-target__arrow'
          src={reportArrowRightIcon}
          mode='aspectFit'
        />
      ) : null}
    </>
  );
  if (!isDetailTarget) {
    return <View className='report-transmission-target'>{content}</View>;
  }
  return (
    <View
      className='report-transmission-target'
      role='button'
      ariaLabel={`查看${target.label}推理详情`}
      onClick={() =>
        onOpenDetail({
          reportId,
          targetType: target.ref.type as 'layer' | 'industry_chain',
          targetKey: target.ref.key
        })
      }
    >
      {content}
    </View>
  );
}

function LayerUncertaintyView({
  uncertainty
}: {
  uncertainty: ReportLayerDetail['layer']['uncertainty'];
}) {
  const items = [
    ['反证', uncertainty.counterevidence],
    ['证据缺口', uncertainty.evidenceGap],
    ['分析边界', uncertainty.boundary],
    ['反转条件', uncertainty.reversalCondition]
  ].filter((item): item is [string, string] => Boolean(item[1]));
  if (!items.length && !uncertainty.checkpoints.length) return null;
  return (
    <DetailSection title='不确定性与反转条件'>
      <View className='report-uncertainty-list'>
        {items.map(([label, value]) => (
          <View className='report-uncertainty-item' key={label}>
            <Text>{label}</Text>
            <Text>{value}</Text>
          </View>
        ))}
        {uncertainty.checkpoints.map((checkpoint) => (
          <View className='report-uncertainty-item' key={checkpoint.key}>
            <Text>观察点 {checkpoint.displayOrder}</Text>
            <Text>{checkpoint.summary}</Text>
          </View>
        ))}
      </View>
    </DetailSection>
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

  return (
    <View className='report-detail-flow'>
      <View className='report-chain-hero'>
        <View className='report-chain-hero__heading'>
          <Text>{industryChain.name}</Text>
          {industryChain.hasEvidence ? (
            <ScopeEvidenceButton
              reportId={report.id}
              scope={industryChain.scope}
              title={`${industryChain.name}证据`}
              onOpen={onOpenEvidence}
            />
          ) : null}
        </View>
        <Text className='report-chain-hero__conclusion'>{industryChain.conclusion}</Text>
        <ReportImpactSignals
          result={industryChain.result}
          confidence={industryChain.confidence}
          timeWindow={industryChain.timeWindow}
        />
        <View className='report-chain-status'>
          <Text>链状态</Text>
          <Text>{industryChain.status}</Text>
        </View>
      </View>

      {industryChain.pathSummary || industryChain.acceptedHypothesisSummary ? (
        <DetailSection title='链路判断'>
          <View className='report-chain-summary-list'>
            {industryChain.pathSummary ? (
              <View className='report-detail-fact'>
                <Text>链路摘要</Text>
                <Text>{industryChain.pathSummary}</Text>
              </View>
            ) : null}
            {industryChain.acceptedHypothesisSummary ? (
              <View className='report-detail-fact'>
                <Text>已采纳假设</Text>
                <Text>{industryChain.acceptedHypothesisSummary}</Text>
              </View>
            ) : null}
          </View>
        </DetailSection>
      ) : null}

      <DetailSection title='产业链图' aside='横向滑动 · 点击节点'>
        <ChainGraph
          nodes={industryChain.nodes}
          edges={industryChain.edges}
          selectedNodeKey={selectedNodeKey}
          onSelect={setSelectedNodeKey}
        />
      </DetailSection>

      {selectedNode ? (
        <DetailSection title='节点详情'>
          <View className='report-node-detail'>
            <View className='report-node-detail__heading'>
              <Text>{selectedNode.name}</Text>
              {selectedNode.hasEvidence ? (
                <ScopeEvidenceButton
                  reportId={report.id}
                  scope={selectedNode.scope}
                  title={`${selectedNode.name}证据`}
                  onOpen={onOpenEvidence}
                />
              ) : null}
            </View>
            <ReportImpactSignals
              result={selectedNode.result}
              confidence={selectedNode.confidence}
              timeWindow={selectedNode.timeWindow}
              nature={selectedNode.nature}
            />
            <View className='report-detail-fact'>
              <Text>本次影响</Text>
              <Text>{selectedNode.impact}</Text>
            </View>
            <View className='report-detail-fact'>
              <Text>传导逻辑</Text>
              <Text>{selectedNode.reasoning}</Text>
            </View>
          </View>
        </DetailSection>
      ) : null}

      {industryChain.uncertainty.counterevidenceAndGap ? (
        <View className='report-chain-boundary report-chain-boundary--gap'>
          <Text>反证与缺口</Text>
          <Text>{industryChain.uncertainty.counterevidenceAndGap}</Text>
        </View>
      ) : null}
      {industryChain.uncertainty.stopCondition ? (
        <View className='report-chain-boundary report-chain-boundary--stop'>
          <Text>停止条件</Text>
          <Text>{industryChain.uncertainty.stopCondition}</Text>
        </View>
      ) : null}
      {industryChain.uncertainty.checkpoints.length ? (
        <DetailSection title='后续观察点'>
          <View className='report-uncertainty-list'>
            {industryChain.uncertainty.checkpoints.map((checkpoint) => (
              <View className='report-uncertainty-item' key={checkpoint.key}>
                <Text>{checkpoint.displayOrder}</Text>
                <Text>{checkpoint.summary}</Text>
              </View>
            ))}
          </View>
        </DetailSection>
      ) : null}
    </View>
  );
}

function ChainGraph({
  nodes,
  edges,
  selectedNodeKey,
  onSelect
}: {
  nodes: ReportIndustryChainNode[];
  edges: ReportIndustryChainDetail['industryChain']['edges'];
  selectedNodeKey: string;
  onSelect: (key: string) => void;
}) {
  const nodeWidth = 248;
  const nodeGap = 56;
  const canvasPadding = 28;
  const step = nodeWidth + nodeGap;
  const nodeIndexes = new Map(nodes.map((item, index) => [item.key, index]));
  const longEdges = edges.filter((edgeItem) => {
    const from = nodeIndexes.get(edgeItem.fromNodeKey);
    const to = nodeIndexes.get(edgeItem.toNodeKey);
    return from !== undefined && to !== undefined && Math.abs(from - to) > 1;
  });
  const laneHeight = 52 + longEdges.length * 34;
  const canvasWidth =
    canvasPadding * 2 + nodes.length * nodeWidth + Math.max(0, nodes.length - 1) * nodeGap;

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
          const fromIndex = nodeIndexes.get(edgeItem.fromNodeKey);
          const toIndex = nodeIndexes.get(edgeItem.toNodeKey);
          if (fromIndex === undefined || toIndex === undefined) return null;
          const adjacent = Math.abs(fromIndex - toIndex) === 1;
          const movesRight = fromIndex < toIndex;
          const nodeTop = laneHeight;
          if (adjacent) {
            const sourceX =
              canvasPadding + fromIndex * step + (movesRight ? nodeWidth : 0);
            const targetX = canvasPadding + toIndex * step + (movesRight ? 0 : nodeWidth);
            return (
              <View key={edgeItem.key}>
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
          const lane = longEdges.findIndex((candidate) => candidate.key === edgeItem.key);
          const laneTop = 18 + lane * 34;
          return (
            <View key={edgeItem.key}>
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
          {nodes.map((nodeItem) => (
            <Button
              className={`tidewise-button report-chain-node ${selectedNodeKey === nodeItem.key ? 'is-selected' : ''}`}
              hoverClass='report-chain-node--pressed'
              key={nodeItem.key}
              ariaLabel={`查看${nodeItem.name}节点详情`}
              onClick={() => onSelect(nodeItem.key)}
            >
              <Text className='report-chain-node__name'>{nodeItem.name}</Text>
              <ReportImpactSignals
                result={nodeItem.result}
                confidence={nodeItem.confidence}
                timeWindow={nodeItem.timeWindow}
                nature={nodeItem.nature}
              />
            </Button>
          ))}
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
  scope,
  title,
  onOpen
}: {
  reportId: string;
  scope: ReportEvidenceScope;
  title: string;
  onOpen: (route: ReportEvidenceRoute) => void;
}) {
  return (
    <ReportEvidenceButton
      label={`查看${title}`}
      onClick={() =>
        onOpen({
          reportId,
          scopeType: scope.type,
          scopeKey: scope.key,
          title
        })
      }
    />
  );
}

function DetailSection({
  title,
  aside,
  children
}: {
  title: string;
  aside?: string;
  children: ReactNode;
}) {
  return (
    <View className='report-detail-section'>
      <View className='report-detail-section__heading'>
        <Text>{title}</Text>
        {aside ? <Text>{aside}</Text> : null}
      </View>
      {children}
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
