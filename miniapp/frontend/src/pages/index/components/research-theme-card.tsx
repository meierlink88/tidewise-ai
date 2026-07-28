import Taro from '@tarojs/taro';
import { Button, Image, Text, View } from '@tarojs/components';
import type { BaseEventOrig } from '@tarojs/components/types/common';
import type { HomeResearchThemeItem } from '../../../features/research-themes/contract';
import {
  researchImpactStrengthLabel,
  researchInvestmentGuidanceActionLabel,
  researchTransmissionStageLabel
} from '../../../features/research-themes/presentation';
import { navigateToResearchReasoningTrees } from '../../../features/research-reasoning-trees/navigation';
import arrowRightIcon from '../../../assets/icons/arrow-right.svg';

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
    <View className={`theme-card theme-card--${theme.impactStrength}`}>
      <View className='theme-card__rail' />
      <View className='theme-card__topline'>
        <View className='theme-card__identity'>
          <Text className='theme-card__impact'>
            {researchImpactStrengthLabel(theme.impactStrength)}
          </Text>
          <View className='theme-card__divider' />
          <Text className='theme-card__category'>{theme.title}</Text>
        </View>
        <View className='theme-card__updated'>
          <View className='theme-card__updated-dot' />
          <Text>{theme.updateLabel}</Text>
        </View>
      </View>

      <Text className='theme-card__title'>{theme.oneLineConclusion}</Text>
      <View className='theme-card__path'>
        <Text>{theme.transmissionSummary || '—'}</Text>
      </View>

      <View className='theme-card__industries'>
        <View className='theme-card__industry-count'>
          <Text className='theme-card__industry-number'>{theme.impacts.length}</Text>
          <Text className='theme-card__industry-label'>个受影响节点</Text>
        </View>
        <View className='theme-card__node-list'>
          {theme.impacts.map((node) => (
            <Button
              key={node.chainNodeEntityId}
              className='tidewise-button theme-card__node'
              hoverClass='none'
              onClick={(event) => handleNestedTap(event, `${node.name}详情即将开放`)}
            >
              {node.name}
            </Button>
          ))}
        </View>
      </View>

      <View className='theme-card__checkpoint'>
        <Text className='theme-card__checkpoint-label'>
          {researchInvestmentGuidanceActionLabel(theme.investmentGuidanceAction)}
        </Text>
        <Text className='theme-card__checkpoint-text'>{theme.investmentGuidanceSummary}</Text>
      </View>

      <View className='theme-card__footer'>
        <Button
          className='tidewise-button theme-card__event-button'
          hoverClass='none'
          onClick={(event) => handleNestedTap(event, '事件清单即将开放')}
        >
          <Text>{theme.evidenceEventCount} 条政经事件</Text>
        </Button>
        <View className='theme-card__phase'>
          <Text>传导阶段</Text>
          <Text className='theme-card__phase-dot'>·</Text>
          <Text>{researchTransmissionStageLabel(theme.transmissionStage)}</Text>
        </View>
        <Button
          className='tidewise-button theme-card__detail-button'
          hoverClass='none'
          onClick={(event) => {
            event.stopPropagation();
            void navigateToResearchReasoningTrees(theme.id, (options) =>
              Taro.navigateTo(options)
            ).catch(() => showUnavailable('影响路径暂时无法打开'));
          }}
        >
          <Text>查看影响路径</Text>
          <Image className='theme-card__detail-icon' src={arrowRightIcon} mode='scaleToFill' />
        </Button>
      </View>
    </View>
  );
}
