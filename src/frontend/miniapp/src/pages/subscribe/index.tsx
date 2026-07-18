import { View, Text } from '@tarojs/components';
import { useEffect, useState } from 'react';
import { getSubscriptionTopics } from '@/services/subscription-service';
import type { SubscriptionTopic } from '@/models/subscription';
import './index.scss';

export default function SubscribePage() {
  const [topics, setTopics] = useState<SubscriptionTopic[]>([]);

  useEffect(() => {
    getSubscriptionTopics().then(setTopics);
  }, []);

  return (
    <View className='page-shell subscribe-page'>
      <Text className='page-title'>订阅</Text>
      <Text className='page-subtitle'>管理关注主题、企业和事件提醒。</Text>
      <View className='subscribe-page__list'>
        {topics.map((topic) => (
          <View className='subscribe-page__item' key={topic.id}>
            <Text className='subscribe-page__name'>{topic.name}</Text>
            <Text className='subscribe-page__desc'>{topic.description}</Text>
          </View>
        ))}
      </View>
    </View>
  );
}
