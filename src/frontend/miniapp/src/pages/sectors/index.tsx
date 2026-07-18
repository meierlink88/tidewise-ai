import { View, Text } from '@tarojs/components';
import { useEffect, useState } from 'react';
import { getSectorSignals } from '@/services/sector-service';
import type { SectorSignal } from '@/models/sector';
import './index.scss';

export default function SectorsPage() {
  const [sectors, setSectors] = useState<SectorSignal[]>([]);

  useEffect(() => {
    getSectorSignals().then(setSectors);
  }, []);

  return (
    <View className='page-shell sectors-page'>
      <Text className='page-title'>板块</Text>
      <Text className='page-subtitle'>查看事件驱动下的板块热度和传导方向。</Text>
      <View className='sectors-page__list'>
        {sectors.map((sector) => (
          <View className='sectors-page__item' key={sector.id}>
            <Text className='sectors-page__name'>{sector.name}</Text>
            <Text className='sectors-page__score'>热度 {sector.heat}</Text>
          </View>
        ))}
      </View>
    </View>
  );
}
