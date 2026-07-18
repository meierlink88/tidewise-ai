import Taro from '@tarojs/taro';
import { Button, Image, ScrollView, Text, View } from '@tarojs/components';
import clockIcon from '../../../assets/icons/clock.svg';

interface CategoryBarProps {
  categories: string[];
  activeCategory: string;
  trackingCount: number;
  onCategoryChange: (category: string) => void;
}

export function CategoryBar({ categories, activeCategory, trackingCount, onCategoryChange }: CategoryBarProps) {
  return (
    <View className='category-bar'>
      <ScrollView className='category-bar__scroll' scrollX showScrollbar={false}>
        <View className='category-bar__items'>
          {categories.map((category) => (
            <Button
              key={category}
              className={`category-chip${activeCategory === category ? ' category-chip--active' : ''}`}
              hoverClass='none'
              onClick={() => onCategoryChange(category)}
            >
              {category}
            </Button>
          ))}
        </View>
      </ScrollView>
      <Button
        className='tracking-button'
        hoverClass='none'
        onClick={() => void Taro.showToast({ title: '跟踪列表即将开放', icon: 'none', duration: 1600 })}
      >
        <Image className='tracking-button__icon' src={clockIcon} mode='aspectFit' />
        <Text>跟踪中 {trackingCount}</Text>
      </Button>
    </View>
  );
}
