export interface AiMessage {
  id: string;
  role: 'assistant' | 'user';
  content: string;
  safetyLevel: 'decision-support';
}
