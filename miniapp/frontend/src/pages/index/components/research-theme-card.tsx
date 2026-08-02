import Taro from '@tarojs/taro';
import { Button, Image, Text, View } from '@tarojs/components';
import type { HomeResearchThemeItem } from '../../../features/research-themes/contract';
import {
  researchImpactStrengthLabel,
  researchInvestmentGuidanceActionLabel,
  researchNodeOutlook,
  researchNodeOutlookLabel
} from '../../../features/research-themes/presentation';
import { navigateToResearchReasoningTrees } from '../../../features/research-reasoning-trees/navigation';
import arrowRightIcon from '../../../assets/icons/arrow-right.svg';
import './research-theme-card.scss';

interface ResearchThemeCardProps {
  theme: HomeResearchThemeItem;
  onOpenEvents: (themeId: string) => void;
}

function showUnavailable(title: string) {
  void Taro.showToast({ title, icon: 'none', duration: 1600 });
}

export function ResearchThemeCard({ theme, onOpenEvents }: ResearchThemeCardProps) {
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
      {theme.transmissionSummary ? (
        <View className='theme-card__path'>
          <Text>{theme.transmissionSummary}</Text>
        </View>
      ) : null}

      <View className='theme-card__industries'>
        <View className='theme-card__industry-count'>
          <Text className='theme-card__industry-number'>{theme.impacts.length}</Text>
          <Text className='theme-card__industry-label'>个关注节点</Text>
        </View>
        <View className='theme-card__node-list'>
          {theme.impacts.map((node) => {
            const outlook = researchNodeOutlook(node.impactDirection);
            return (
              <View key={node.chainNodeEntityId} className='theme-card__node'>
                <Text className='theme-card__node-name'>{node.name}</Text>
                <Text className={`theme-card__outlook theme-card__outlook--${outlook}`}>
                  {researchNodeOutlookLabel(outlook)}
                </Text>
              </View>
            );
          })}
        </View>
      </View>

      <View className='theme-card__checkpoint'>
        <Text className='theme-card__checkpoint-label'>
          {researchInvestmentGuidanceActionLabel(theme.investmentGuidanceAction)}
        </Text>
        <Text className='theme-card__checkpoint-text'>{theme.investmentGuidanceSummary}</Text>
      </View>

      <View className='theme-card__footer'>
        {theme.evidenceEventCount > 0 ? (
          <View className='theme-card__event-action' catchMove>
            <Button
              className='tidewise-button theme-card__event-count theme-card__event-button'
              hoverClass='none'
              ariaLabel={`查看${theme.title}关联的${theme.evidenceEventCount}条政经事件`}
              onClick={(event) => {
                event.stopPropagation();
                onOpenEvents(theme.id);
              }}
            >
              {theme.evidenceEventCount} 条政经事件
            </Button>
          </View>
        ) : (
          <Text className='theme-card__event-count'>0 条政经事件</Text>
        )}
        <Text className='theme-card__path-count'>{theme.reasoningTreeCount} 条产业链路径</Text>
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
          <Text>推导详情</Text>
          <Image className='theme-card__detail-icon' src={arrowRightIcon} mode='scaleToFill' />
        </Button>
      </View>
    </View>
  );
}
