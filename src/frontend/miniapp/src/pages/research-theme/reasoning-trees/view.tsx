import { ScrollView, Text, View } from '@tarojs/components';
import { Fragment } from 'react';
import type {
  ResearchReasoningTree,
  ResearchReasoningTreeEvent,
  ResearchReasoningTreePathNode,
  ResearchReasoningTreeTheme
} from '../../../features/research-reasoning-trees/contract';
import { researchImpactLabel } from '../../../features/research-themes/presentation';
import {
  formatReasoningTimestamp,
  researchChangeDirectionPresentation,
  researchEvidenceRoleLabel
} from '../../../features/research-reasoning-trees/presentation';

export function ReasoningThemeHero({ theme }: { theme: ResearchReasoningTreeTheme }) {
  return (
    <View className='reasoning-theme-hero'>
      <View className='reasoning-theme-hero__meta'>
        <Text className={`reasoning-impact reasoning-impact--${theme.impactLevel}`}>
          {researchImpactLabel(theme.impactLevel)}
        </Text>
        <Text className='reasoning-theme-hero__name'>{theme.name}</Text>
        <Text className='reasoning-theme-hero__published'>
          {formatReasoningTimestamp(theme.publishedAt)} 发布
        </Text>
      </View>
      <Text className='reasoning-theme-hero__title'>{theme.oneLineConclusion}</Text>
      <View className='reasoning-theme-hero__path'>
        <Text className='reasoning-theme-hero__path-label'>主题传导路径</Text>
        <Text className='reasoning-theme-hero__path-text'>{theme.transmissionPath}</Text>
      </View>
    </View>
  );
}

export function ReasoningTreeView({ tree }: { tree: ResearchReasoningTree }) {
  return (
    <View className='reasoning-tree'>
      <View className='reasoning-tree__heading'>
        <Text className='reasoning-tree__heading-title'>
          {tree.centerChainNode.name}影响推理树
        </Text>
        <Text className='reasoning-tree__heading-meta'>{tree.eventCount} 条 Event</Text>
      </View>

      <View className='reasoning-tree__stack'>
        <EventFactCard tree={tree} />

        <View className='reasoning-anchor-conclusion'>
          <Text className='reasoning-anchor-conclusion__label'>ANCHOR 结论</Text>
          <Text className='reasoning-anchor-conclusion__title'>{tree.oneLineConclusion}</Text>
          <Text className='reasoning-anchor-conclusion__direction'>
            {tree.netDirectionSummary}
          </Text>
        </View>

        <View className='reasoning-evidence'>
          <View className='reasoning-evidence__card reasoning-evidence__card--support'>
            <Text className='reasoning-evidence__label'>当前支持</Text>
            <Text className='reasoning-evidence__summary'>{tree.supportSummary}</Text>
          </View>
          <View className='reasoning-evidence__card reasoning-evidence__card--counter'>
            <Text className='reasoning-evidence__label'>当前反证</Text>
            <Text className='reasoning-evidence__summary'>
              {tree.counterSummary ?? '当前暂无明确反证'}
            </Text>
          </View>
        </View>

        <ChainPath tree={tree} />

        <View className='reasoning-action reasoning-action--trade'>
          <Text className='reasoning-action__label'>交易指向</Text>
          <Text className='reasoning-action__text'>{tree.tradingDirection}</Text>
        </View>

        <View className='reasoning-action reasoning-action--checkpoint'>
          <Text className='reasoning-action__label'>下一检查点</Text>
          <Text className='reasoning-action__text'>{tree.nextCheckpoint}</Text>
        </View>
      </View>
    </View>
  );
}

function EventFactCard({ tree }: { tree: ResearchReasoningTree }) {
  return (
    <View className='reasoning-facts'>
      <View className='reasoning-facts__head'>
        <Text className='reasoning-section-label'>事件事实汇总</Text>
        <Text className='reasoning-facts__count'>{tree.eventCount} 条 Event</Text>
      </View>
      <Text className='reasoning-facts__summary'>{tree.factSummary}</Text>
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

function ChainPath({ tree }: { tree: ResearchReasoningTree }) {
  return (
    <View className='reasoning-chain'>
      <View className='reasoning-chain__head'>
        <Text className='reasoning-chain__title'>产业链节点传导</Text>
        <Text className='reasoning-chain__hint'>左右滑动查看完整路径</Text>
      </View>
      <ScrollView className='reasoning-chain__scroll' scrollX showScrollbar={false}>
        <View className='reasoning-chain__flow'>
          {tree.pathNodes.map((node, index) => (
            <Fragment key={node.chainNodeId}>
              {index > 0 ? <ChainConnector /> : null}
              <ChainNode
                node={node}
                index={index}
                isAnchor={node.chainNodeId === tree.centerChainNode.id}
              />
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

function ChainNode({
  node,
  index,
  isAnchor
}: {
  node: ResearchReasoningTreePathNode;
  index: number;
  isAnchor: boolean;
}) {
  const direction = researchChangeDirectionPresentation(node.changeDirection);
  return (
    <View className={`reasoning-chain-node ${isAnchor ? 'reasoning-chain-node--anchor' : ''}`}>
      <Text className='reasoning-chain-node__index'>节点 {String(index + 1).padStart(2, '0')}</Text>
      <Text className='reasoning-chain-node__name'>{node.name}</Text>
      <Text
        className={`reasoning-chain-node__direction reasoning-chain-node__direction--${direction.tone}`}
      >
        {direction.label}
      </Text>
      <Text className='reasoning-chain-node__change'>{node.changeSummary}</Text>
      {node.incomingTransmissionMechanism ? (
        <View className='reasoning-chain-node__mechanism'>
          <Text className='reasoning-chain-node__mechanism-label'>传导机制</Text>
          <Text className='reasoning-chain-node__mechanism-text'>
            {node.incomingTransmissionMechanism}
          </Text>
        </View>
      ) : null}
      <View className='reasoning-chain-node__impact'>
        <Text className='reasoning-chain-node__impact-label'>影响</Text>
        <Text>{node.impactSummary}</Text>
      </View>
    </View>
  );
}
