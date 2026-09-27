import { useMemo, type ReactNode } from 'react';
import Taro from '@tarojs/taro';
import { Button, ScrollView, Text, View } from '@tarojs/components';
import { RiskNotice } from '../../components/risk-notice';
import { NavigationBar } from '../../platform/navigation-bar';
import { getHomeChromeMetrics } from '../../platform/system-ui';
import { openProfile } from '../../platform/identity';
import './preview.scss';
import { OverlayRoot } from '../../platform/overlay-root';

export function PreviewPage({
  title,
  subtitle,
  filters,
  headerSearch,
  children
}: {
  title: string;
  subtitle: string;
  filters?: ReactNode;
  headerSearch?: ReactNode;
  children: ReactNode;
}) {
  const chrome = useMemo(() => getHomeChromeMetrics(Taro), []);
  return (
    <View
      className={`preview-page${headerSearch ? ' preview-page--header-search' : ''}${filters ? ' preview-page--filtered' : ''}`}
    >
      <View className='preview-brand'>
        <NavigationBar
          title='观潮家'
          chrome={chrome}
          leading={
            <Button
              className='tidewise-button preview-profile'
              ariaLabel='我的'
              onClick={() => void openProfile()}
            >
              ◎
            </Button>
          }
        />
        <View className='preview-brand-copy'>
          {filters && <Text className='preview-simulation'>设计模拟</Text>}
          <View className='preview-between'>
            <Text className='preview-title'>{title}</Text>
            {!filters && <Text className='preview-simulation'>设计模拟</Text>}
          </View>
          <Text className='preview-subtitle'>{subtitle}</Text>
          {headerSearch}
        </View>
      </View>
      {filters && <View className='preview-filters'>{filters}</View>}
      <ScrollView scrollY className='preview-scroll'>
        <View className='preview-content'>{children}</View>
      </ScrollView>
      <RiskNotice />
    </View>
  );
}
export function PreviewTabs({
  names,
  variant = 'underline',
  selected,
  onChange
}: {
  names: readonly string[];
  variant?: 'underline' | 'pills';
  selected: number;
  onChange: (i: number) => void;
}) {
  return (
    <ScrollView scrollX className='preview-tabs-scroll'>
      <View className={`preview-tabs preview-tabs--${variant}`}>
        {names.map((name, i) => (
          <Button
            key={name}
            className={'tidewise-button ' + ('preview-tab ' + (selected === i ? 'selected' : ''))}
            ariaLabel={name + (selected === i ? '，已选中' : '')}
            onClick={() => onChange(i)}
          >
            {name}
          </Button>
        ))}
      </View>
    </ScrollView>
  );
}
export function PreviewSheet({
  title,
  subtitle,
  children,
  close
}: {
  title: string;
  subtitle?: string;
  children: ReactNode;
  close: () => void;
}) {
  return (
    <OverlayRoot>
      <View className='preview-overlay' onClick={close} catchMove>
        <View
          className={`preview-sheet${subtitle ? ' preview-sheet--report' : ''}`}
          onClick={(e) => e.stopPropagation()}
        >
          <View className='preview-sheet-header'>
            <View>
              <Text>{title}</Text>
              {subtitle && <Text className='preview-sheet-subtitle'>{subtitle}</Text>}
            </View>
            <Button className='tidewise-button preview-close' ariaLabel='关闭' onClick={close}>
              ×
            </Button>
          </View>
          <ScrollView scrollY className='preview-scroll'>
            <View className='preview-content'>{children}</View>
          </ScrollView>
        </View>
      </View>
    </OverlayRoot>
  );
}
