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
  researchNodeJudgment
} from '../../../features/research-reasoning-trees/presentation';

export function ReasoningThemeHero({ theme }: { theme: ResearchReasoningTreeTheme }) {
  return (
    <View className='reasoning-theme-hero'>
      <View className='reasoning-theme-hero__meta'>
        <Text className={`reasoning-impact reasoning-impact--${theme.impactStrength}`}>
          {researchImpactStrengthLabel(theme.impactStrength)}
        </Text>
        <Text className='reasoning-theme-hero__published'>
          {formatReasoningTimestamp(theme.publishedAt)} 发布
        </Text>
      </View>
      <Text className='reasoning-theme-hero__title'>{theme.oneLineConclusion}</Text>
      {theme.transmissionSummary ? (
        <View className='reasoning-theme-hero__path'>
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
  const judgmentCounts = tree.nodes.reduce(
    (counts, node) => {
      counts[researchNodeJudgment(node.impactDirection).kind] += 1;
      return counts;
    },
    { opportunity: 0, risk: 0, uncertain: 0 }
  );
  useEffect(
    () => setSelectedNodeId(tree.nodes.at(-1)?.id ?? ''),
    [tree.reasoningTreeId, tree.nodes]
  );
  return (
    <View className='reasoning-tree'>
      <View className='reasoning-tree__stack'>
        <EventFactCard detail={detail} />
        <View className='reasoning-tree-conclusion'>
          <Text className='reasoning-tree-conclusion__label'>本树结论</Text>
          <Text className='reasoning-tree-conclusion__title'>{tree.oneLineConclusion}</Text>
          <Text className='reasoning-tree-conclusion__direction'>
            {judgmentCounts.opportunity} 个机会 · {judgmentCounts.risk} 个风险 ·{' '}
            {judgmentCounts.uncertain} 个不确定
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
        <View className='reasoning-actions'>
          <View className='reasoning-action reasoning-action--boundary'>
            <Text className='reasoning-action__label'>判断边界</Text>
            <Text className='reasoning-action__text'>
              {tree.conclusionBoundarySummary || '暂无'}
            </Text>
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
  return (
    <Fragment>
      <View className='reasoning-chain'>
        <View className='reasoning-chain__head'>
          <Text className='reasoning-chain__title'>产业链节点传导</Text>
        </View>
        <ScrollView className='reasoning-chain__scroll' scrollX showScrollbar={false}>
          <View className='reasoning-chain__flow'>
            {nodes.map((node, index) => {
              const judgment = researchNodeJudgment(node.impactDirection);
              return (
                <Fragment key={node.id}>
                  {index > 0 ? <ChainConnector /> : null}
                  <View
                    className={`reasoning-chain-node ${selected?.id === node.id ? 'reasoning-chain-node--selected' : ''}`}
                    onClick={() => onSelect(node.id)}
                  >
                    <Text className='reasoning-chain-node__index'>
                      节点 {String(node.position).padStart(2, '0')}
                    </Text>
                    <View className='reasoning-chain-node__name-slot'>
                      <Text className='reasoning-chain-node__name'>{node.name}</Text>
                    </View>
                    <View className='reasoning-chain-node__signal-slot'>
                      <Text className='reasoning-chain-node__direction'>
                        {node.primarySignal.displaySummary}
                      </Text>
                    </View>
                    <View className='reasoning-chain-node__outlook'>
                      <Text
                        className={`reasoning-chain-node__judgment reasoning-judgment reasoning-judgment--${judgment.kind}`}
                      >
                        {judgment.label}
                      </Text>
                      <Text className='reasoning-chain-node__strength'>
                        {researchImpactStrengthLabel(node.impactStrength)}
                      </Text>
                    </View>
                  </View>
                </Fragment>
              );
            })}
          </View>
        </ScrollView>
      </View>
      {selected ? <ReasoningNodeDetail node={selected} /> : null}
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

function ReasoningNodeDetail({ node }: { node: ResearchReasoningTreeNode }) {
  const judgment = researchNodeJudgment(node.impactDirection);
  return (
    <View className='reasoning-node-detail'>
      <View className='reasoning-node-detail__head'>
        <View>
          <Text className='reasoning-node-detail__eyebrow'>
            节点 {String(node.position).padStart(2, '0')}
          </Text>
          <Text className='reasoning-node-detail__title'>{node.name}</Text>
        </View>
        <View className='reasoning-node-detail__judgment-wrap'>
          <Text
            className={`reasoning-node-detail__judgment reasoning-judgment reasoning-judgment--${judgment.kind}`}
          >
            {judgment.label}
          </Text>
          <Text className='reasoning-node-detail__strength'>
            {researchImpactStrengthLabel(node.impactStrength)}
          </Text>
        </View>
      </View>
      <View className='reasoning-node-detail__summary'>
        <View className='reasoning-node-detail__metric'>
          <Text className='reasoning-node-detail__metric-label'>变量状态</Text>
          <View className='reasoning-node-detail__signals'>
            {node.signals.map((signal) => (
              <Text
                key={`${signal.variableSignalKey}-${signal.displayOrder}`}
                className='reasoning-node-detail__signal-summary'
              >
                {signal.displaySummary}
              </Text>
            ))}
          </View>
        </View>
        <View className='reasoning-node-detail__metric'>
          <Text className='reasoning-node-detail__metric-label'>投资含义</Text>
          <Text className='reasoning-node-detail__impact-summary'>
            {node.impactSummary || '暂无'}
          </Text>
        </View>
      </View>
      {node.position > 1 && node.incomingTransmissionTitle && node.incomingTransmissionMechanism ? (
        <View className='reasoning-node-detail__mechanism'>
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
    </View>
  );
}

function EventFactCard({ detail }: { detail: ResearchReasoningTreeDetail }) {
  const tree = detail.reasoningTree;
  return (
    <View className='reasoning-facts'>
      <View className='reasoning-facts__head'>
        <Text className='reasoning-section-label'>事件事实汇总</Text>
        <Text className='reasoning-facts__count'>{tree.eventCount} 条政经事件</Text>
      </View>
      {tree.factSummary ? (
        <Text className='reasoning-facts__summary'>{tree.factSummary}</Text>
      ) : tree.events.length === 0 ? (
        <Text className='reasoning-facts__summary reasoning-facts__summary--empty'>
          暂无事件事实
        </Text>
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
          <Text className='reasoning-event__title'>{event.title}</Text>
          {event.eventTime ? (
            <Text className='reasoning-event__time'>
              {formatReasoningTimestamp(event.eventTime)}
            </Text>
          ) : null}
        </View>
        <Text className='reasoning-event__summary'>{event.summary}</Text>
      </View>
    </View>
  );
}
