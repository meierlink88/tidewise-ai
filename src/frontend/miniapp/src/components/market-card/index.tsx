import { View, Text } from '@tarojs/components';
import type { MarketAnchor } from '@/models/market';
import './index.scss';

interface MarketCardProps {
  item: MarketAnchor;
}

export function MarketCard({ item }: MarketCardProps) {
  return (
    <View className='market-card'>
      <Text className='market-card__name'>{item.name}</Text>
      <Text className='market-card__value'>{item.value}</Text>
      <Text className='market-card__trend'>{item.trend}</Text>
    </View>
  );
}
