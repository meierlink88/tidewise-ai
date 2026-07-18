import { View, Text } from '@tarojs/components';
import './index.scss';

interface ConfidenceBarProps {
  value: number;
}

export function ConfidenceBar({ value }: ConfidenceBarProps) {
  const normalized = Math.max(0, Math.min(100, value));

  return (
    <View className='confidence-bar'>
      <View className='confidence-bar__track'>
        <View className='confidence-bar__fill' style={{ width: `${normalized}%` }} />
      </View>
      <Text className='confidence-bar__label'>{normalized}%</Text>
    </View>
  );
}
