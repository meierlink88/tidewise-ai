import Taro, { usePullDownRefresh, useReachBottom } from '@tarojs/taro';
import { Button, Text, View } from '@tarojs/components';
import { useEffect, useMemo, useState } from 'react';
import { filterHomeResearchThemes } from '../../features/research-themes/feed';
import type { ResearchThemePeriod } from '../../features/research-themes/contract';
import { createResearchThemeHomepagePort } from '../../features/research-themes/port';
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

export function ThemeListPage({ period }: { period: ResearchThemePeriod }) {
  const session = useMemo(
    () => new ResearchThemeHomeSession(createResearchThemeHomepagePort(), { period }),
    [period]
  );
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

  useReachBottom(() => {
    if (period === 'history') void session.loadMore();
  });

  const handlePeriodAction = () => {
    if (period === 'today') {
      void Taro.navigateTo({ url: '/pages/research-theme/history/index' });
    } else {
      void Taro.navigateBack();
    }
  };

  return (
    <IndexView
      state={state}
      period={period}
      query={query}
      chrome={chrome}
      onQueryChange={setQuery}
      onRetryFeed={() => void session.retryFeed()}
      onOpenEvents={(themeId) => session.openThemeEvents(themeId)}
      onCloseEvents={() => session.closeThemeEvents()}
      onRetryEvents={() => session.retryThemeEvents()}
      onPeriodAction={handlePeriodAction}
      onLoadMore={() => void session.loadMore()}
    />
  );
}

export function IndexView({
  state,
  period = 'today',
  query,
  chrome,
  onQueryChange,
  onRetryFeed,
  onOpenEvents,
  onCloseEvents,
  onRetryEvents,
  onPeriodAction = () => undefined,
  onLoadMore = () => undefined
}: {
  state: ResearchThemeHomeSessionState;
  period?: ResearchThemePeriod;
  query: string;
  chrome: HomeChromeMetrics;
  onQueryChange: (query: string) => void;
  onRetryFeed: () => void;
  onOpenEvents: (themeId: string) => void;
  onCloseEvents: () => void;
  onRetryEvents: () => void;
  onPeriodAction?: () => void;
  onLoadMore?: () => void;
}) {
  const feed = state.feed.status === 'ready' ? state.feed.value : null;
  const visibleThemes = filterHomeResearchThemes(feed?.items ?? [], query);
  const selectedTheme = feed?.items.find((theme) => theme.id === state.selectedThemeId) ?? null;

  return (
    <View className='home-page'>
      <HomeHeader
        chrome={chrome}
        query={query}
        period={period}
        onQueryChange={onQueryChange}
        onPeriodAction={onPeriodAction}
      />

      <View className='home-content'>
        <View className='home-section-heading'>
          <Text className='home-section-heading__title'>
            {period === 'today' ? '今日主题' : '历史主题'}
          </Text>
          <Text className='home-section-heading__summary'>
            {feed?.themeCount ?? 0} 条主题 · {feed?.eventCount ?? 0} 条政经事件
          </Text>
        </View>

        {state.feed.status === 'error' ? (
          <View className='home-state'>
            <Text>主题数据暂时不可用</Text>
            <Button
              className='tidewise-button home-state__retry'
              hoverClass='none'
              onClick={onRetryFeed}
            >
              重新加载
            </Button>
          </View>
        ) : state.feed.status === 'idle' || state.feed.status === 'loading' ? (
          <View className='home-state'>
            {period === 'today' ? '正在整理今日主题' : '正在整理历史主题'}
          </View>
        ) : visibleThemes.length === 0 ? (
          <View className='home-state'>没有找到符合条件的主题</View>
        ) : (
          <View className='home-theme-list'>
            {visibleThemes.map((theme) => (
              <ResearchThemeCard key={theme.id} theme={theme} onOpenEvents={onOpenEvents} />
            ))}
            {period === 'history' ? (
              <View className='home-pagination-state'>
                {state.pagination === 'loading' ? (
                  <Text>正在加载更多主题</Text>
                ) : state.pagination === 'error' ? (
                  <Button
                    className='tidewise-button home-pagination-state__retry'
                    hoverClass='none'
                    onClick={onLoadMore}
                  >
                    加载失败，点击重试
                  </Button>
                ) : state.pagination === 'exhausted' ? (
                  <Text>已展示近 30 天全部主题</Text>
                ) : null}
              </View>
            ) : null}
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
      void api.showToast({ title: '主题已更新', icon: 'success', duration: 1200 });
    } else if (result === 'failed') {
      void api.showToast({ title: '刷新失败，请稍后重试', icon: 'none', duration: 1600 });
    }
  } finally {
    void api.stopPullDownRefresh();
  }
}
