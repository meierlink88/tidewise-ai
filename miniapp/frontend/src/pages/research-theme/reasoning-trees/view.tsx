import { ScrollView, Text, View } from '@tarojs/components';
import { Fragment, useEffect, useMemo, useState } from 'react';
import type {
  ResearchReasoningTreeDetail,
  ResearchReasoningTreeEvent,
  ResearchReasoningTreeNode,
  ResearchReasoningTreeTheme
} from '../../../features/research-reasoning-trees/contract';
import { researchImpactStrengthLabel } from '../../../features/research-themes/presentation';
import {
  formatReasoningTimestamp,
  researchEvidenceRoleLabel,
  researchSignalDirectionLabel,
  researchStrengthLabel,
  researchTreeConclusionMeta
} from '../../../features/research-reasoning-trees/presentation';

export function ReasoningThemeHero({ theme }: { theme: ResearchReasoningTreeTheme }) {
  return (
    <View className='reasoning-theme-hero'>
      <View className='reasoning-theme-hero__meta'>
        <Text className={`reasoning-impact reasoning-impact--${theme.impactStrength}`}>
          {researchImpactStrengthLabel(theme.impactStrength)}
        </Text>
        <Text className='reasoning-theme-hero__name'>{theme.title}</Text>
        <Text className='reasoning-theme-hero__published'>
          {formatReasoningTimestamp(theme.publishedAt)} 发布
        </Text>
      </View>
      <Text className='reasoning-theme-hero__title'>{theme.oneLineConclusion}</Text>
      {theme.transmissionSummary ? (
        <View className='reasoning-theme-hero__path'>
          <Text className='reasoning-theme-hero__path-label'>Theme 传导摘要</Text>
          <Text className='reasoning-theme-hero__path-text'>{theme.transmissionSummary}</Text>
        </View>
      ) : null}
    </View>
  );
}

export function ReasoningTreeView({ detail }: { detail: ResearchReasoningTreeDetail }) {
  const { reasoningTree: tree, impactNodeIds } = detail;
  const initial = tree.nodes.at(-1);
  const [selectedNodeId, setSelectedNodeId] = useState(initial?.id ?? '');
  useEffect(
    () => setSelectedNodeId(tree.nodes.at(-1)?.id ?? ''),
    [tree.reasoningTreeId, tree.nodes]
  );
  const selected = useMemo(
    () => tree.nodes.find((node) => node.id === selectedNodeId) ?? tree.nodes.at(-1),
    [tree.nodes, selectedNodeId]
  );
  return (
    <View className='reasoning-tree'>
      <View className='reasoning-tree__stack'>
        <View className='reasoning-tree__heading'>
          <Text className='reasoning-tree__heading-title'>{tree.title}</Text>
          <Text className='reasoning-tree__heading-meta'>{tree.eventCount} 条 Event</Text>
        </View>
        <EventFactCard detail={detail} />
        <View className='reasoning-tree-conclusion'>
          <Text className='reasoning-tree-conclusion__label'>本树结论</Text>
          <Text className='reasoning-tree-conclusion__title'>{tree.oneLineConclusion}</Text>
          <Text className='reasoning-tree-conclusion__direction'>
            {researchTreeConclusionMeta(
              tree.impactDirection,
              tree.impactStrength,
              tree.impactSummary
            )}
          </Text>
          {tree.transmissionSummary ? (
            <Text className='reasoning-tree-conclusion__transmission'>
              {tree.transmissionSummary}
            </Text>
          ) : null}
        </View>
        <View className='reasoning-evidence'>
          <View className='reasoning-evidence__card reasoning-evidence__card--support'>
            <Text className='reasoning-evidence__label'>当前支持</Text>
            <Text className='reasoning-evidence__summary'>{tree.supportSummary || '—'}</Text>
          </View>
          <View className='reasoning-evidence__card reasoning-evidence__card--counter'>
            <Text className='reasoning-evidence__label'>当前反证</Text>
            <Text className='reasoning-evidence__summary'>
              {tree.counterSummary || '当前暂无明确反证'}
            </Text>
          </View>
        </View>
        <ChainPath
          nodes={tree.nodes}
          selectedNodeId={selected?.id ?? ''}
          onSelect={setSelectedNodeId}
        />
        {selected ? (
          <NodeDetail
            node={selected}
            isImpact={impactNodeIds.includes(selected.chainNodeEntityId)}
            isResult={selected.id === tree.nodes.at(-1)?.id}
          />
        ) : null}
        <View className='reasoning-action reasoning-action--trade'>
          <Text className='reasoning-action__label'>结论边界与失效条件</Text>
          {tree.conclusionBoundarySummary ? (
            <Text className='reasoning-action__text'>{tree.conclusionBoundarySummary}</Text>
          ) : null}
          {tree.invalidationConditions.length > 0 ? (
            tree.invalidationConditions.map((condition, index) => (
              <Text key={index} className='reasoning-action__text'>
                • {condition}
              </Text>
            ))
          ) : (
            <Text className='reasoning-action__text'>暂无</Text>
          )}
        </View>
        <View className='reasoning-action reasoning-action--checkpoint'>
          <Text className='reasoning-action__label'>下一检查点</Text>
          {tree.checkpoints.length > 0 ? (
            tree.checkpoints.map((checkpoint, index) => (
              <Text key={`${checkpoint.type}-${index}`} className='reasoning-action__text'>
                • {checkpoint.summary}
              </Text>
            ))
          ) : (
            <Text className='reasoning-action__text'>暂无</Text>
          )}
        </View>
      </View>
    </View>
  );
}

function ChainPath({
  nodes,
  selectedNodeId,
  onSelect
}: {
  nodes: ResearchReasoningTreeNode[];
  selectedNodeId: string;
  onSelect: (id: string) => void;
}) {
  return (
    <View className='reasoning-chain'>
      <View className='reasoning-chain__head'>
        <Text className='reasoning-chain__title'>产业链节点传导</Text>
        <Text className='reasoning-chain__hint'>选择节点查看详细依据</Text>
      </View>
      <ScrollView className='reasoning-chain__scroll' scrollX showScrollbar={false}>
        <View className='reasoning-chain__flow'>
          {nodes.map((node, index) => (
            <Fragment key={node.id}>
              {index > 0 ? <ChainConnector /> : null}
              <View
                className={`reasoning-chain-node ${selectedNodeId === node.id ? 'reasoning-chain-node--selected' : ''}`}
                onClick={() => onSelect(node.id)}
              >
                <Text className='reasoning-chain-node__index'>
                  节点 {String(node.position).padStart(2, '0')}
                </Text>
                <View className='reasoning-chain-node__name-slot'>
                  <Text className='reasoning-chain-node__name'>{node.name}</Text>
                </View>
                <View className='reasoning-chain-node__signal-slot'>
                  <Text
                    className={`reasoning-chain-node__direction reasoning-chain-node__direction--${node.primarySignal.signalDirection}`}
                  >
                    {node.primarySignal.displaySummary}
                  </Text>
                </View>
                <View className='reasoning-chain-node__gap'>
                  <Text className='reasoning-chain-node__gap-text'>
                    {node.evidenceGapSummary || '—'}
                  </Text>
                </View>
                <View className='reasoning-chain-node__action'>
                  <Text className='reasoning-chain-node__action-label'>
                    {selectedNodeId === node.id ? '当前节点' : '节点详情'}
                  </Text>
                </View>
              </View>
            </Fragment>
          ))}
        </View>
      </ScrollView>
    </View>
  );
}
function ChainConnector() {
  return (
    <View className='reasoning-chain-edge'>
      <View className='reasoning-chain-edge__line'>
        <View className='reasoning-chain-edge__arrow' />
      </View>
    </View>
  );
}

function NodeDetail({
  node,
  isImpact,
  isResult
}: {
  node: ResearchReasoningTreeNode;
  isImpact: boolean;
  isResult: boolean;
}) {
  return (
    <View className='reasoning-node-detail'>
      <View className='reasoning-node-detail__head'>
        <View>
          <Text className='reasoning-node-detail__eyebrow'>
            节点 {String(node.position).padStart(2, '0')} ·{' '}
            {node.position === 1 ? '信号入口' : isResult ? '结果节点' : '路径节点'}
            {node.position === 1 && isResult ? ' · 结果节点' : ''}
            {isImpact ? ' · Theme Impact' : ''}
          </Text>
          <Text className='reasoning-node-detail__title'>{node.name}</Text>
        </View>
        <Text className='reasoning-node-detail__signal'>{node.primarySignal.displaySummary}</Text>
      </View>
      <View className='reasoning-node-detail__status'>
        <View>
          <Text className='reasoning-node-detail__label'>影响状态</Text>
          <Text>
            {node.stateSummary ? `${node.stateSummary} · ` : ''}
            {researchStrengthLabel(node.impactStrength)}
          </Text>
        </View>
        <View>
          <Text className='reasoning-node-detail__label'>变量状态</Text>
          <Text>
            {researchSignalDirectionLabel(node.primarySignal.signalDirection)}
            {node.signalDisplaySummary ? ` · ${node.signalDisplaySummary}` : ''}
          </Text>
        </View>
      </View>
      {node.incomingTransmissionMechanism ? (
        <View className='reasoning-node-detail__section'>
          <View className='reasoning-node-detail__section-head'>
            <Text>传导机制</Text>
            <Text className='reasoning-node-detail__edge'>
              {node.incomingGraphEdge
                ? `${node.incomingGraphEdge.relationType} · ${node.incomingGraphEdge.reviewStatus}`
                : '分析推断'}
            </Text>
          </View>
          <Text className='reasoning-node-detail__section-title'>
            {node.incomingTransmissionTitle}
          </Text>
          <Text>{node.incomingTransmissionMechanism}</Text>
          {node.incomingConditionSummary ? (
            <Text className='reasoning-node-detail__condition'>
              成立条件：{node.incomingConditionSummary}
            </Text>
          ) : null}
        </View>
      ) : null}
      <View className='reasoning-node-detail__section'>
        <View className='reasoning-node-detail__section-head'>
          <Text>变量信号</Text>
          <Text className='reasoning-node-detail__edge'>主要 + 支持 / 反向</Text>
        </View>
        <View className='reasoning-node-detail__signals'>
          {node.signals.map((signal) => (
            <Text
              key={`${signal.variableSignalKey}-${signal.displayOrder}`}
              className={`reasoning-node-detail__signal-chip reasoning-node-detail__signal-chip--${signal.signalRole}`}
            >
              {signal.signalRole === 'primary'
                ? '主要'
                : signal.signalRole === 'supporting'
                  ? '支持'
                  : '反向'}{' '}
              {signal.displaySummary}
            </Text>
          ))}
        </View>
      </View>
      <View className='reasoning-node-detail__basis'>
        <View>
          <Text className='reasoning-node-detail__label'>推导依据</Text>
          <Text>{node.reasoningBasisSummary || '—'}</Text>
        </View>
        <View>
          <Text className='reasoning-node-detail__label'>数据缺口</Text>
          <Text>{node.evidenceGapSummary || '—'}</Text>
        </View>
      </View>
    </View>
  );
}

function EventFactCard({ detail }: { detail: ResearchReasoningTreeDetail }) {
  const tree = detail.reasoningTree;
  if (tree.events.length === 0 && !tree.factSummary) return null;
  return (
    <View className='reasoning-facts'>
      <View className='reasoning-facts__head'>
        <Text className='reasoning-section-label'>事件事实汇总</Text>
        <Text className='reasoning-facts__count'>{tree.eventCount} 条 Event</Text>
      </View>
      {tree.factSummary ? (
        <Text className='reasoning-facts__summary'>{tree.factSummary}</Text>
      ) : null}
      <View className='reasoning-event-list'>
        {tree.events.map((event, index) => (
          <EventFact key={event.eventId} event={event} index={index} />
        ))}
      </View>
    </View>
  );
}
function EventFact({ event, index }: { event: ResearchReasoningTreeEvent; index: number }) {
  return (
    <View className='reasoning-event'>
      <Text className='reasoning-event__number'>{String(index + 1).padStart(2, '0')}</Text>
      <View className='reasoning-event__body'>
        <View className='reasoning-event__meta'>
          <Text className={`reasoning-event__role reasoning-event__role--${event.evidenceRole}`}>
            {researchEvidenceRoleLabel(event.evidenceRole)}
          </Text>
          {event.eventTime ? (
            <Text className='reasoning-event__time'>
              {formatReasoningTimestamp(event.eventTime)}
            </Text>
          ) : null}
        </View>
        <Text className='reasoning-event__title'>{event.title}</Text>
        <Text className='reasoning-event__summary'>{event.summary}</Text>
      </View>
    </View>
  );
}
