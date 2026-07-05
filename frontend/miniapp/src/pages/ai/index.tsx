import { View, Text } from '@tarojs/components';
import { useEffect, useState } from 'react';
import { getAssistantGreeting } from '@/services/ai-service';
import type { AiMessage } from '@/models/ai-message';
import './index.scss';

export default function AiPage() {
  const [message, setMessage] = useState<AiMessage>();

  useEffect(() => {
    getAssistantGreeting().then(setMessage);
  }, []);

  return (
    <View className='page-shell ai-page'>
      <Text className='page-title'>AI 助手</Text>
      <Text className='page-subtitle'>围绕事件、板块和资产传导提问。</Text>
      <View className='ai-page__notice'>
        <Text>内容仅用于市场理解和决策辅助，不构成直接投资建议。</Text>
      </View>
      {message ? (
        <View className='ai-page__message'>
          <Text>{message.content}</Text>
        </View>
      ) : null}
    </View>
  );
}
