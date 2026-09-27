import { Text, View } from '@tarojs/components';
import './index.scss';

export function RiskNotice() {
  return (
    <View className='risk-notice'>
      <Text className='risk-notice__copy'>AI分析仅供参考，不构成投资建议。</Text>
    </View>
  );
}
