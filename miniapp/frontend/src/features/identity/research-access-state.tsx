import { Button, Text, View } from '@tarojs/components';
import './research-access-state.scss';

export function ResearchAccessState({
  error,
  retry
}: {
  error: string;
  retry: () => Promise<void>;
}) {
  return (
    <View className='research-access-state'>
      <Text>{error || '正在检查登录状态…'}</Text>
      {error && (
        <Button
          className='tidewise-button research-access-state__retry'
          onClick={() => void retry()}
        >
          重试
        </Button>
      )}
    </View>
  );
}
