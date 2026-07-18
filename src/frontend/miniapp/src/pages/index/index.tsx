import Taro from '@tarojs/taro';
import { Text, View } from '@tarojs/components';
import { useEffect, useMemo, useState } from 'react';
import type { HomeResearchThemeFeed } from '../../features/research-themes/contract';
import { filterHomeResearchThemes, getHomeThemeCategories } from '../../features/research-themes/feed';
import { createMockResearchThemeFeedPort } from '../../mocks/research-themes/mock-port';
import { getHomeChromeMetrics } from '../../platform/system-ui';
import { CategoryBar } from './components/category-bar';
import { HomeHeader } from './components/home-header';
import { ResearchThemeCard } from './components/research-theme-card';
import './index.scss';

const feedPort = createMockResearchThemeFeedPort();

export default function IndexPage() {
  const [feed, setFeed] = useState<HomeResearchThemeFeed | null>(null);
  const [loadError, setLoadError] = useState(false);
  const [query, setQuery] = useState('');
  const [activeCategory, setActiveCategory] = useState('全部');
  const chrome = useMemo(() => getHomeChromeMetrics(Taro), []);

  useEffect(() => {
    let active = true;

    void feedPort
      .list()
      .then((nextFeed) => {
        if (active) {
          setFeed(nextFeed);
          setLoadError(false);
        }
      })
      .catch(() => {
        if (active) {
          setLoadError(true);
        }
      });

    return () => {
      active = false;
    };
  }, []);

  const categories = useMemo(() => getHomeThemeCategories(feed?.items ?? []), [feed]);
  const visibleThemes = useMemo(
    () => filterHomeResearchThemes(feed?.items ?? [], { category: activeCategory, query }),
    [activeCategory, feed, query]
  );

  return (
    <View className='home-page'>
      <HomeHeader chrome={chrome} query={query} onQueryChange={setQuery} />

      <View className='home-content'>
        <CategoryBar
          categories={categories}
          activeCategory={activeCategory}
          trackingCount={feed?.trackingCount ?? 17}
          onCategoryChange={setActiveCategory}
        />

        <View className='home-section-heading'>
          <Text className='home-section-heading__title'>今日推理主线</Text>
          <Text className='home-section-heading__summary'>
            {feed?.themeCount ?? 0} 条主线 · {feed?.eventCount ?? 0} 条政经事件
          </Text>
        </View>

        {loadError ? (
          <View className='home-state'>主线数据暂时不可用</View>
        ) : feed === null ? (
          <View className='home-state'>正在整理今日主线</View>
        ) : visibleThemes.length === 0 ? (
          <View className='home-state'>没有找到符合条件的推理主线</View>
        ) : (
          <View className='home-theme-list'>
            {visibleThemes.map((theme) => (
              <ResearchThemeCard key={theme.id} theme={theme} />
            ))}
          </View>
        )}
      </View>
    </View>
  );
}
