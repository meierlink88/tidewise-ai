import { View } from '@tarojs/components';
import type { ReactNode } from 'react';
import type { HomeChromeMetrics } from './system-ui';
import './navigation-bar.scss';

export function NavigationBar({
  title,
  chrome,
  leading
}: Readonly<{
  title: string;
  chrome: HomeChromeMetrics;
  leading: ReactNode;
}>) {
  return (
    <View className='navigation-bar'>
      <View style={{ height: `${chrome.statusBarHeight}px` }} />
      <View className='navigation-bar__row' style={{ height: `${chrome.navigationBarHeight}px` }}>
        <View className='navigation-bar__leading'>{leading}</View>
        <View
          className='navigation-bar__title'
          style={{
            left: `${chrome.rightReservedWidth}px`,
            right: `${chrome.rightReservedWidth}px`
          }}
        >
          {title}
        </View>
      </View>
    </View>
  );
}
