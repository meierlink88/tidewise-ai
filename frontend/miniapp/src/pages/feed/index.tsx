import { View, Text } from '@tarojs/components';
import { useEffect, useState } from 'react';
import { getEventHighlights } from '@/services/event-service';
import type { EventHighlight } from '@/models/event';
import './index.scss';

export default function FeedPage() {
  const [events, setEvents] = useState<EventHighlight[]>([]);

  useEffect(() => {
    getEventHighlights().then(setEvents);
  }, []);

  return (
    <View className='page-shell feed-page'>
      <Text className='page-title'>行情</Text>
      <Text className='page-subtitle'>跟踪全球政经事件与市场传导线索。</Text>
      <View className='feed-page__list'>
        {events.map((event) => (
          <View className='feed-page__item' key={event.id}>
            <Text className='feed-page__item-title'>{event.title}</Text>
            <Text className='feed-page__item-meta'>{event.region} · {event.impact}</Text>
          </View>
        ))}
      </View>
    </View>
  );
}
