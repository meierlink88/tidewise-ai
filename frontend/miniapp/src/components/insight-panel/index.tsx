import { View, Text } from '@tarojs/components';
import './index.scss';

interface InsightPanelProps {
  title: string;
  content: string;
}

export function InsightPanel({ title, content }: InsightPanelProps) {
  return (
    <View className='insight-panel'>
      <Text className='insight-panel__title'>{title}</Text>
      <Text className='insight-panel__content'>{content}</Text>
    </View>
  );
}
