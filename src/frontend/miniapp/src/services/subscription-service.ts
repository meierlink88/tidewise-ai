import { mockSubscriptions } from '@/data/mock-subscriptions';
import type { SubscriptionTopic } from '@/models/subscription';
import { request } from './request';

export function getSubscriptionTopics(): Promise<SubscriptionTopic[]> {
  return request({
    mock: () => mockSubscriptions
  });
}
