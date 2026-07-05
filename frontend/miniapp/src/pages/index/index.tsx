import { View, Text } from '@tarojs/components';
import { useEffect, useState } from 'react';
import { getMarketAnchors } from '@/services/market-service';
import type { MarketAnchor } from '@/models/market';
import './index.scss';

export default function IndexPage() {
  const [anchors, setAnchors] = useState<MarketAnchor[]>([]);

  useEffect(() => {
    getMarketAnchors().then(setAnchors);
  }, []);

  return (
    <View className='page-shell index-page'>
      <Text className='page-title'>指数</Text>
      <Text className='page-subtitle'>观察全球定价锚与风险偏好变化。</Text>
      <View className='index-page__grid'>
        {anchors.map((anchor) => (
          <View className='index-page__card' key={anchor.id}>
            <Text className='index-page__name'>{anchor.name}</Text>
            <Text className='index-page__value'>{anchor.value}</Text>
            <Text className='index-page__trend'>{anchor.trend}</Text>
          </View>
        ))}
      </View>
    </View>
  );
}
