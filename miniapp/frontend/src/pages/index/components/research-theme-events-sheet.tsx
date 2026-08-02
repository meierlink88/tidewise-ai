import { Button, ScrollView, Text, View } from '@tarojs/components';
import type { HomeResearchThemeItem } from '../../../features/research-themes/contract';
import type { ResearchThemeDetailState } from '../../../features/research-themes/session';
import './research-theme-events-sheet.scss';

interface ResearchThemeEventsSheetProps {
  theme: HomeResearchThemeItem;
  detailState: ResearchThemeDetailState | undefined;
  onClose: () => void;
  onRetry: () => void;
}

export function ResearchThemeEventsSheet({
  theme,
  detailState,
  onClose,
  onRetry
}: ResearchThemeEventsSheetProps) {
  return (
    <View
      className='theme-events-overlay'
      catchMove
      ariaLabel={`${theme.title}关联政经事件`}
      onClick={onClose}
    >
      <View
        className='theme-events-sheet'
        onClick={(event) => {
          event.stopPropagation();
        }}
      >
        <View className='theme-events-sheet__handle' />
        <View className='theme-events-sheet__header'>
          <View className='theme-events-sheet__topline'>
            <Text className='theme-events-sheet__eyebrow'>关联政经事件</Text>
            <Button
              className='tidewise-button theme-events-sheet__close'
              hoverClass='none'
              ariaLabel='关闭关联政经事件'
              onClick={onClose}
            >
              ×
            </Button>
          </View>
          <Text className='theme-events-sheet__title'>{theme.title}</Text>
          <View className='theme-events-sheet__meta'>
            <Text>按事件发生时间倒序</Text>
            <Text>{theme.evidenceEventCount} 条事件</Text>
          </View>
        </View>

        <ThemeEventsContent state={detailState} onRetry={onRetry} />
      </View>
    </View>
  );
}

function ThemeEventsContent({
  state,
  onRetry
}: {
  state: ResearchThemeDetailState | undefined;
  onRetry: () => void;
}) {
  if (state === undefined || state.status === 'loading') {
    return (
      <View className='theme-events-state'>
        <View className='theme-events-state__pulse' />
        <Text>正在整理关联事件</Text>
      </View>
    );
  }
  if (state.status === 'error') {
    return (
      <View className='theme-events-state'>
        <Text className='theme-events-state__title'>
          {state.errorKind === 'themeUnavailable' ? '该主题事件暂不可用' : '事件清单加载失败'}
        </Text>
        <Text className='theme-events-state__description'>请稍后重试</Text>
        <Button
          className='tidewise-button theme-events-state__retry'
          hoverClass='none'
          onClick={onRetry}
        >
          重新加载
        </Button>
      </View>
    );
  }

  return (
    <ScrollView
      className='theme-events-scroll'
      scrollY
      showScrollbar={false}
      ariaLabel='关联政经事件时间线'
    >
      <View className='theme-events-timeline'>
        {state.value.events.map((event) => {
          const time = event.eventTime;
          return (
            <View key={event.eventId} className='theme-events-item'>
              <View
                className={`theme-events-item__time${time.status === 'pending' ? ' theme-events-item__time--pending' : ''}`}
              >
                {time.status === 'pending' ? (
                  <Text>时间待确认</Text>
                ) : (
                  <>
                    <Text>{time.date}</Text>
                    <Text>{time.time}</Text>
                  </>
                )}
              </View>
              <View className='theme-events-item__dot' />
              <View className='theme-events-item__content'>
                <Text className='theme-events-item__title'>{event.title}</Text>
                <Text className='theme-events-item__summary'>{event.summary}</Text>
              </View>
            </View>
          );
        })}
      </View>
    </ScrollView>
  );
}
