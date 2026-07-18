import type { AiMessage } from '../models/ai-message';

export type AssistantGreetingLoader = () => Promise<AiMessage>;

export function createAIService(load: AssistantGreetingLoader) {
	return {
		getAssistantGreeting: load
	};
}

const defaultService = createAIService(() => Promise.resolve({
	id: 'ai-message-001',
	role: 'assistant',
	content: '可以从事件、板块或资产角度提问，我会按决策辅助定位给出结构化分析。',
	safetyLevel: 'decision-support'
}));

export function getAssistantGreeting(): Promise<AiMessage> {
	return defaultService.getAssistantGreeting();
}
