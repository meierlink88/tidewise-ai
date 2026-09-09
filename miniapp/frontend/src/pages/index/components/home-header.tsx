import { Button, Image, Input, Text, View } from '@tarojs/components';
import type { HomeChromeMetrics } from '../../../platform/system-ui';
import avatarImage from '../../../assets/nav-avatar.png';
import searchIcon from '../../../assets/icons/search.svg';
import sendIcon from '../../../assets/icons/send.svg';

interface HomeHeaderProps {
  chrome: HomeChromeMetrics;
  isSinglePage?: boolean;
  publishedAt?: string;
  query: string;
  onQueryChange: (query: string) => void;
}

export function HomeHeader({
  chrome,
  publishedAt,
  query,
  onQueryChange,
  isSinglePage = false
}: HomeHeaderProps) {
  const timestamp = publishedAt ? Date.parse(publishedAt) : NaN;
  const date = Number.isFinite(timestamp) ? new Date(timestamp + 8 * 60 * 60 * 1000) : null;
  const two = (value: number) => String(value).padStart(2, '0');
  const dateLabel = date
    ? `${two(date.getUTCMonth() + 1)}.${two(date.getUTCDate())} 周${'日一二三四五六'[date.getUTCDay()]}`
    : '';
  const timeLabel = date ? `截至 ${two(date.getUTCHours())}:${two(date.getUTCMinutes())}` : '';

  return (
    <View className={isSinglePage ? 'home-hero home-hero--single-page' : 'home-hero'}>
      {!isSinglePage && (
        <>
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
              aria-label='个人中心'
              disabled
            >
              <Image className='home-nav__avatar' src={avatarImage} mode='aspectFill' />
            </Button>
            <View className='home-nav__title'>观潮家</View>
          </View>
        </>
      )}
      <View className='home-brief'>
        <View className='home-brief__copy'>
          <Text className='home-brief__title'>全球政经事件</Text>
          <Text className='home-brief__subtitle'>
            {dateLabel ? `${dateLabel} · ` : ''}读懂全球政经变化
          </Text>
        </View>
        {timeLabel ? <Text className='home-brief__window'>{timeLabel}</Text> : null}
      </View>
      <View className='home-search-row'>
        <View className='home-search'>
          <Image className='home-search__icon' src={searchIcon} mode='scaleToFill' />
          <Input
            className='home-search__input'
            type='text'
            value={query}
            confirmType='search'
            placeholder='搜索报告结论'
            placeholderClass='home-search__placeholder'
            onInput={(event) => onQueryChange(event.detail.value)}
          />
          <Button
            className='tidewise-button home-search__send'
            hoverClass='none'
            aria-label='问潮'
            disabled
          >
            <Image className='home-search__send-icon' src={sendIcon} mode='scaleToFill' />
          </Button>
        </View>
      </View>
    </View>
  );
}
