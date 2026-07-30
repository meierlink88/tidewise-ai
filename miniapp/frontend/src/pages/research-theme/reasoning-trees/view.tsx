import { ScrollView, Text, View } from '@tarojs/components';
import { Fragment, useEffect, useState } from 'react';
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
          <Text className='reasoning-theme-hero__path-label'>主题传导摘要</Text>
          <Text className='reasoning-theme-hero__path-text'>{theme.transmissionSummary}</Text>
        </View>
      ) : null}
    </View>
  );
}

export function ReasoningTreeView({ detail }: { detail: ResearchReasoningTreeDetail }) {
  const { reasoningTree: tree } = detail;
  const initial = tree.nodes.at(-1);
  const [selectedNodeId, setSelectedNodeId] = useState(initial?.id ?? '');
  useEffect(
    () => setSelectedNodeId(tree.nodes.at(-1)?.id ?? ''),
    [tree.reasoningTreeId, tree.nodes]
  );
  return (
    <View className='reasoning-tree'>
      <View className='reasoning-tree__stack'>
        <View className='reasoning-tree__heading'>
          <Text className='reasoning-tree__heading-title'>{tree.title}</Text>
          <Text className='reasoning-tree__heading-meta'>{tree.eventCount} 条政经事件</Text>
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
        <ReasoningTreeTransmission
          nodes={tree.nodes}
          selectedNodeId={selectedNodeId}
          onSelect={setSelectedNodeId}
        />
        <View className='reasoning-action reasoning-action--boundary'>
          <Text className='reasoning-action__label'>判断边界</Text>
          <Text className='reasoning-action__text'>{tree.conclusionBoundarySummary || '暂无'}</Text>
        </View>
        <View className='reasoning-action reasoning-action--verification'>
          <Text className='reasoning-action__label'>后续验证</Text>
          {tree.checkpoints.length > 0 ? (
            <View className='reasoning-action__list'>
              {tree.checkpoints.map((checkpoint, index) => (
                <View
                  key={`${checkpoint.type}-${index}`}
                  className='reasoning-action__verification-item'
                >
                  <Text className='reasoning-action__text'>{checkpoint.summary}</Text>
                </View>
              ))}
            </View>
          ) : (
            <Text className='reasoning-action__text'>暂无</Text>
          )}
        </View>
      </View>
    </View>
  );
}

export function ReasoningTreeTransmission({
  nodes,
  selectedNodeId,
  onSelect
}: {
  nodes: ResearchReasoningTreeNode[];
  selectedNodeId: string;
  onSelect: (id: string) => void;
}) {
  const selectedIndex = nodes.findIndex((node) => node.id === selectedNodeId);
  const resolvedSelectedIndex = selectedIndex >= 0 ? selectedIndex : nodes.length - 1;
  const selected = nodes[resolvedSelectedIndex];
  const previousNode = resolvedSelectedIndex > 0 ? nodes[resolvedSelectedIndex - 1] : undefined;
  return (
    <Fragment>
      <View className='reasoning-chain'>
        <View className='reasoning-chain__head'>
          <Text className='reasoning-chain__title'>产业链节点传导</Text>
          <Text className='reasoning-chain__hint'>选择节点查看详情</Text>
        </View>
        <ScrollView className='reasoning-chain__scroll' scrollX showScrollbar={false}>
          <View className='reasoning-chain__flow'>
            {nodes.map((node, index) => (
              <Fragment key={node.id}>
                {index > 0 ? <ChainConnector /> : null}
                <View
                  className={`reasoning-chain-node ${selected?.id === node.id ? 'reasoning-chain-node--selected' : ''}`}
                  onClick={() => onSelect(node.id)}
                >
                  <Text className='reasoning-chain-node__index'>
                    节点 {String(node.position).padStart(2, '0')}
                    {index === nodes.length - 1 ? ' · 结果' : ''}
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
                  <Text className='reasoning-chain-node__strength'>
                    {researchStrengthLabel(node.impactStrength)}
                  </Text>
                  <View className='reasoning-chain-node__action'>
                    <Text className='reasoning-chain-node__action-label'>
                      {selected?.id === node.id ? '当前节点' : '节点详情'}
                    </Text>
                  </View>
                </View>
              </Fragment>
            ))}
          </View>
        </ScrollView>
      </View>
      {selected ? (
        <ReasoningNodeDetail
          node={selected}
          previousNode={previousNode}
          isResult={resolvedSelectedIndex === nodes.length - 1}
        />
      ) : null}
    </Fragment>
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

function ReasoningNodeDetail({
  node,
  previousNode,
  isResult
}: {
  node: ResearchReasoningTreeNode;
  previousNode?: ResearchReasoningTreeNode;
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
          </Text>
          <Text className='reasoning-node-detail__title'>{node.name}</Text>
        </View>
        <Text className='reasoning-node-detail__signal'>{node.primarySignal.displaySummary}</Text>
      </View>
      {node.position > 1 &&
      previousNode &&
      node.incomingTransmissionTitle &&
      node.incomingTransmissionMechanism ? (
        <View className='reasoning-node-detail__mechanism'>
          <View className='reasoning-node-detail__section-head'>
            <Text>传导机制</Text>
            <Text className='reasoning-node-detail__edge'>
              节点 {String(previousNode.position).padStart(2, '0')} → 节点{' '}
              {String(node.position).padStart(2, '0')}
            </Text>
          </View>
          <Text className='reasoning-node-detail__section-title'>
            {node.incomingTransmissionTitle}
          </Text>
          <Text>{node.incomingTransmissionMechanism}</Text>
          {node.incomingConditionSummary ? (
            <Text className='reasoning-node-detail__condition'>
              成立前提：{node.incomingConditionSummary}
            </Text>
          ) : null}
        </View>
      ) : null}
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
        <Text className='reasoning-facts__count'>{tree.eventCount} 条政经事件</Text>
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
