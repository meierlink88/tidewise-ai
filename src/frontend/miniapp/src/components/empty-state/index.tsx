import { View, Text } from '@tarojs/components';
import './index.scss';

interface EmptyStateProps {
  title: string;
  description?: string;
}

export function EmptyState({ title, description }: EmptyStateProps) {
  return (
    <View className='empty-state'>
      <Text className='empty-state__title'>{title}</Text>
      {description ? <Text className='empty-state__description'>{description}</Text> : null}
    </View>
  );
}
