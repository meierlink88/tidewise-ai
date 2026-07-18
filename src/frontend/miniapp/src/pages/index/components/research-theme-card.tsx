import Taro from '@tarojs/taro';
import { Button, Image, Text, View } from '@tarojs/components';
import type { BaseEventOrig } from '@tarojs/components/types/common';
import type { HomeResearchThemeItem, ResearchImpactLevel } from '../../../features/research-themes/contract';
import arrowRightIcon from '../../../assets/icons/arrow-right.svg';

const impactLabels: Record<ResearchImpactLevel, string> = {
  high: '高影响',
  focus: '重点关注',
  watch: '持续观察'
};

interface ResearchThemeCardProps {
  theme: HomeResearchThemeItem;
}

function showUnavailable(title: string) {
  void Taro.showToast({ title, icon: 'none', duration: 1600 });
}

function handleNestedTap(event: BaseEventOrig, title: string) {
  event.stopPropagation();
  showUnavailable(title);
}

export function ResearchThemeCard({ theme }: ResearchThemeCardProps) {
  return (
    <View
      className={`theme-card theme-card--${theme.impactLevel}`}
      onClick={() => showUnavailable('主线详情即将开放')}
    >
      <View className='theme-card__rail' />
      <View className='theme-card__topline'>
        <View className='theme-card__identity'>
          <Text className='theme-card__impact'>{impactLabels[theme.impactLevel]}</Text>
          <View className='theme-card__divider' />
          <Text className='theme-card__category'>{theme.name}</Text>
        </View>
        <View className='theme-card__updated'>
          <View className='theme-card__updated-dot' />
          <Text>{theme.updateLabel}</Text>
        </View>
      </View>

      <Text className='theme-card__title'>{theme.oneLineConclusion}</Text>
      <View className='theme-card__path'>
        <Text>{theme.transmissionPath}</Text>
      </View>

      <View className='theme-card__industries'>
        <View className='theme-card__industry-count'>
          <Text className='theme-card__industry-number'>{theme.affectedChainNodes.length}</Text>
          <Text className='theme-card__industry-label'>个产业受到影响</Text>
        </View>
        <View className='theme-card__node-list'>
          {theme.affectedChainNodes.map((node) => (
            <Button
              key={node.id}
              className='theme-card__node'
              hoverClass='none'
              onClick={(event) => handleNestedTap(event, `${node.name}详情即将开放`)}
            >
              {node.name}
            </Button>
          ))}
        </View>
      </View>

      <View className='theme-card__checkpoint'>
        <Text className='theme-card__checkpoint-label'>尚未显现</Text>
        <Text className='theme-card__checkpoint-text'>{theme.nextCheckpoint}</Text>
      </View>

      <View className='theme-card__footer'>
        <Button
          className='theme-card__event-button'
          hoverClass='none'
          onClick={(event) => handleNestedTap(event, '事件清单即将开放')}
        >
          <Text>{theme.supportingEventCount} 条政经事件</Text>
        </Button>
        <View className='theme-card__phase'>
          <Text>传导阶段</Text>
          <Text className='theme-card__phase-dot'>·</Text>
          <Text>{theme.transmissionPhaseLabel}</Text>
        </View>
        <Button
          className='theme-card__detail-button'
          hoverClass='none'
          onClick={(event) => handleNestedTap(event, '影响路径即将开放')}
        >
          <Text>查看影响路径</Text>
          <Image className='theme-card__detail-icon' src={arrowRightIcon} mode='aspectFit' />
        </Button>
      </View>
    </View>
  );
}
