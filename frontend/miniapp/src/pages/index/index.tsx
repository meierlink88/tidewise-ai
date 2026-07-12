import Taro, { useDidShow } from '@tarojs/taro';
import { Text, View } from '@tarojs/components';
import { useCallback, useReducer, useState } from 'react';
import { BriefHero, ConclusionCard, ResourceStatePanel, SafetyNote } from '../../components/daily-brief';
import type { DailyBriefHomeView } from '../../models/daily-brief-view';
import { createInitialResourceState, resourceStateReducer } from '../../models/resource-state';
import { getDailyBriefService, setDailyBriefMockScenario } from '../../services/daily-brief';
import { mapDailyBriefToHome } from '../../templates/daily-brief';
import { getVisibleHomeSections } from '../../templates/home-sections';
import { getGraphComingSoonMessage } from '../../utils/coming-soon';
import './index.scss';

export default function IndexPage() {
  const [resource, dispatch] = useReducer(resourceStateReducer<DailyBriefHomeView>, createInitialResourceState<DailyBriefHomeView>());
  const [collapsed, setCollapsed] = useState(true);
  const [activeConclusion, setActiveConclusion] = useState(0);

  const loadBrief = useCallback(async () => {
    dispatch({ type: 'load' });
    const storedScenario = Taro.getStorageSync('dailyBriefMockScenario');
    if (storedScenario === 'ready' || storedScenario === 'empty' || storedScenario === 'error' || storedScenario === 'loading') {
      setDailyBriefMockScenario(storedScenario);
    }
    try {
      const brief = await getDailyBriefService().getDailyBrief();
      dispatch({ type: 'resolve', data: brief ? mapDailyBriefToHome(brief) : null });
    } catch (error) {
      dispatch({ type: 'reject', message: error instanceof Error ? error.message : '今日观潮加载失败' });
    }
  }, []);

  useDidShow(() => {
    void loadBrief();
  });

  if (resource.status !== 'ready') {
    return <View className='daily-brief-page daily-brief-page--state'><ResourceStatePanel state={resource} onRetry={() => void loadBrief()} /></View>;
  }

  const brief = resource.data;
  const sections = getVisibleHomeSections(brief);
  const visibleSections = new Set(sections.map((section) => section.key));
  const conclusion = brief.conclusions[activeConclusion];

  return (
    <View className='daily-brief-page'>
      <View className='daily-brief-page__sea'>
        <BriefHero brief={brief} collapsed={collapsed} onToggle={() => setCollapsed((value) => !value)} showSummary={visibleSections.has('brief-summary')} showThemes={visibleSections.has('themes')} />
      </View>
      <View className='daily-brief-page__content'>
        <View className='daily-brief-page__divider'><View /><Text>观潮分析 · {brief.conclusions.length} 条主线</Text><View /></View>
        {sections.some((section) => section.key === 'conclusions') && conclusion ? (
          <ConclusionCard
            conclusion={conclusion}
            position={activeConclusion}
            total={brief.conclusions.length}
            onPrevious={() => setActiveConclusion((value) => Math.max(0, value - 1))}
            onNext={() => setActiveConclusion((value) => Math.min(brief.conclusions.length - 1, value + 1))}
            onGraph={() => Taro.showToast({ title: getGraphComingSoonMessage(), icon: 'none' })}
            showImpacts={visibleSections.has('impacts')}
            showEvidence={visibleSections.has('evidence')}
          />
        ) : null}
        {visibleSections.has('safety-note') ? <SafetyNote text={brief.disclaimer} /> : null}
      </View>
    </View>
  );
}
