import Taro, { usePullDownRefresh } from '@tarojs/taro';
import { Text, View } from '@tarojs/components';
import { useEffect, useMemo, useState } from 'react';
import { filterHomeResearchThemes } from '../../features/research-themes/feed';
import { createResearchThemeFeedPort } from '../../features/research-themes/port';
import {
  ResearchThemeHomeSession,
  type ResearchThemeHomeSessionState
} from '../../features/research-themes/session';
import { getHomeChromeMetrics, type HomeChromeMetrics } from '../../platform/system-ui';
import { HomeHeader } from './components/home-header';
import { ResearchThemeCard } from './components/research-theme-card';
import { ResearchThemeEventsSheet } from './components/research-theme-events-sheet';
import './index.scss';

interface HomeRefreshAPI {
  stopPullDownRefresh: () => unknown;
  showToast: (options: { title: string; icon: 'none' | 'success'; duration: number }) => unknown;
}

export default function IndexPage() {
  const session = useMemo(() => new ResearchThemeHomeSession(createResearchThemeFeedPort()), []);
  const [state, setState] = useState<ResearchThemeHomeSessionState>(() => session.getState());
  const [query, setQuery] = useState('');
  const chrome = useMemo(() => getHomeChromeMetrics(Taro), []);

  useEffect(() => {
    const unsubscribe = session.subscribe(setState);
    void session.start();
    return () => {
      unsubscribe();
      session.dispose();
    };
  }, [session]);

  usePullDownRefresh(() => {
    void refreshHomeFeed(session, Taro);
  });

  return (
    <IndexView
      state={state}
      query={query}
      chrome={chrome}
      onQueryChange={setQuery}
      onOpenEvents={(themeId) => session.openThemeEvents(themeId)}
      onCloseEvents={() => session.closeThemeEvents()}
      onRetryEvents={() => session.retryThemeEvents()}
    />
  );
}

export function IndexView({
  state,
  query,
  chrome,
  onQueryChange,
  onOpenEvents,
  onCloseEvents,
  onRetryEvents
}: {
  state: ResearchThemeHomeSessionState;
  query: string;
  chrome: HomeChromeMetrics;
  onQueryChange: (query: string) => void;
  onOpenEvents: (themeId: string) => void;
  onCloseEvents: () => void;
  onRetryEvents: () => void;
}) {
  const feed = state.feed.status === 'ready' ? state.feed.value : null;
  const visibleThemes = filterHomeResearchThemes(feed?.items ?? [], query);
  const selectedTheme = feed?.items.find((theme) => theme.id === state.selectedThemeId) ?? null;

  return (
    <View className='home-page'>
      <HomeHeader chrome={chrome} query={query} onQueryChange={onQueryChange} />

      <View className='home-content'>
        <View className='home-section-heading'>
          <Text className='home-section-heading__title'>今日推理主线</Text>
          <Text className='home-section-heading__summary'>
            {feed?.themeCount ?? 0} 条主线 · {feed?.eventCount ?? 0} 条政经事件
          </Text>
        </View>

        {state.feed.status === 'error' ? (
          <View className='home-state'>主线数据暂时不可用</View>
        ) : state.feed.status === 'idle' || state.feed.status === 'loading' ? (
          <View className='home-state'>正在整理今日主线</View>
        ) : visibleThemes.length === 0 ? (
          <View className='home-state'>没有找到符合条件的推理主线</View>
        ) : (
          <View className='home-theme-list'>
            {visibleThemes.map((theme) => (
              <ResearchThemeCard key={theme.id} theme={theme} onOpenEvents={onOpenEvents} />
            ))}
          </View>
        )}
      </View>

      {selectedTheme ? (
        <ResearchThemeEventsSheet
          theme={selectedTheme}
          detailState={state.detailsByThemeId[selectedTheme.id]}
          onClose={onCloseEvents}
          onRetry={onRetryEvents}
        />
      ) : null}
    </View>
  );
}

export async function refreshHomeFeed(
  session: ResearchThemeHomeSession,
  api: HomeRefreshAPI
): Promise<void> {
  try {
    const result = await session.refreshFeed();
    if (result === 'updated') {
      void api.showToast({ title: '主线已更新', icon: 'success', duration: 1200 });
    } else if (result === 'failed') {
      void api.showToast({ title: '刷新失败，请稍后重试', icon: 'none', duration: 1600 });
    }
  } finally {
    void api.stopPullDownRefresh();
  }
}
