import { View, Text } from '@tarojs/components';
import type { SectorSignal } from '@/models/sector';
import './index.scss';

interface SectorCardProps {
  item: SectorSignal;
}

export function SectorCard({ item }: SectorCardProps) {
  return (
    <View className='sector-card'>
      <Text className='sector-card__name'>{item.name}</Text>
      <Text className='sector-card__score'>热度 {item.heat}</Text>
    </View>
  );
}
