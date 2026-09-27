import { Text, View } from '@tarojs/components';
import './index.scss';

export function RiskNotice() {
  return (
    <View className='risk-notice'>
      <Text className='risk-notice__copy'>
        本平台引用数据均来源于公开资料，相关分析与结论均由AI
        生成仅供参考，不构成任何投资建议或收益承诺，请独立判断。市场有风险，投资需谨慎。
      </Text>
    </View>
  );
}
