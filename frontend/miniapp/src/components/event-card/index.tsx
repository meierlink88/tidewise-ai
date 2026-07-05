import { View, Text } from '@tarojs/components';
import type { EventHighlight } from '@/models/event';
import './index.scss';

interface EventCardProps {
  item: EventHighlight;
}

export function EventCard({ item }: EventCardProps) {
  return (
    <View className='event-card'>
      <Text className='event-card__title'>{item.title}</Text>
      <Text className='event-card__meta'>{item.region} · {item.impact}</Text>
    </View>
  );
}
