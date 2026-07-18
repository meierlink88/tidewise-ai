import { mockSubscriptions } from '../mocks/mock-subscriptions';
import type { SubscriptionTopic } from '../models/subscription';

export type SubscriptionTopicsLoader = () => Promise<SubscriptionTopic[]>;

export function createSubscriptionService(load: SubscriptionTopicsLoader) {
	return {
		getSubscriptionTopics: load
	};
}

const defaultService = createSubscriptionService(() => Promise.resolve(mockSubscriptions));

export function getSubscriptionTopics(): Promise<SubscriptionTopic[]> {
	return defaultService.getSubscriptionTopics();
}
