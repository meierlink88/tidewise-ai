import Taro from '@tarojs/taro';
import { Button, Image, Input, Text, View } from '@tarojs/components';
import type { HomeChromeMetrics } from '../../../platform/system-ui';
import type { ResearchThemePeriod } from '../../../features/research-themes/contract';
import avatarImage from '../../../assets/nav-avatar.png';
import themeHistoryIcon from '../../../assets/icons/theme-history.svg';
import todayThemeIcon from '../../../assets/icons/today-theme.svg';
import searchIcon from '../../../assets/icons/search.svg';
import sendIcon from '../../../assets/icons/send.svg';

interface HomeHeaderProps {
  chrome: HomeChromeMetrics;
  query: string;
  period: ResearchThemePeriod;
  onQueryChange: (query: string) => void;
  onPeriodAction: () => void;
}

function showUnavailable(title: string) {
  void Taro.showToast({ title, icon: 'none', duration: 1600 });
}

export function HomeHeader({
  chrome,
  query,
  period,
  onQueryChange,
  onPeriodAction
}: HomeHeaderProps) {
  return (
    <View className='home-hero'>
      <View style={{ height: `${chrome.statusBarHeight}px` }} />
      <View
        className='home-nav'
        style={{
          height: `${chrome.navigationBarHeight}px`,
          paddingRight: `${chrome.rightReservedWidth}px`
        }}
      >
        <Button
          className='tidewise-button home-nav__avatar-button'
          hoverClass='none'
          onClick={() => showUnavailable('个人中心即将开放')}
        >
          <Image className='home-nav__avatar' src={avatarImage} mode='aspectFill' />
        </Button>
        <View className='home-nav__title'>观潮</View>
      </View>

      <View className='home-search-row'>
        <View className='home-search'>
          <Image className='home-search__icon' src={searchIcon} mode='scaleToFill' />
          <Input
            className='home-search__input'
            type='text'
            value={query}
            confirmType='search'
            placeholder='搜索事件、产业，或直接向问潮提问'
            placeholderClass='home-search__placeholder'
            onInput={(event) => onQueryChange(event.detail.value)}
          />
          <Button
            className='tidewise-button home-search__send'
            hoverClass='none'
            onClick={() => showUnavailable('问潮对话即将开放')}
          >
            <Image className='home-search__send-icon' src={sendIcon} mode='scaleToFill' />
          </Button>
        </View>
        <Button
          className={`tidewise-button home-history-button home-history-button--${
            period === 'today' ? 'history' : 'today'
          }`}
          hoverClass='none'
          ariaLabel={period === 'today' ? '查看历史主题' : '返回今日主题'}
          onClick={onPeriodAction}
        >
          <Image
            className='home-history-button__icon'
            src={period === 'today' ? themeHistoryIcon : todayThemeIcon}
            mode='scaleToFill'
          />
          <Text className='home-history-button__label'>{period === 'today' ? '历史' : '今日'}</Text>
        </Button>
      </View>
    </View>
  );
}
