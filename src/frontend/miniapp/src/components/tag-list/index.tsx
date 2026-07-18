import { View, Text } from '@tarojs/components';
import './index.scss';

interface TagListProps {
  tags: string[];
}

export function TagList({ tags }: TagListProps) {
  return (
    <View className='tag-list'>
      {tags.map((tag) => (
        <Text className='tag-list__item' key={tag}>{tag}</Text>
      ))}
    </View>
  );
}
