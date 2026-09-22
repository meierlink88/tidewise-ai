import Taro from '@tarojs/taro';

export async function confirmUnfollow(title: string, symbol: string): Promise<boolean> {
  const result = await Taro.showModal({
    title: '取消跟踪？',
    content: title + '（' + symbol + '）将从你的跟踪列表移除。',
    confirmText: '取消跟踪',
    cancelText: '继续跟踪',
    confirmColor: '#0b1f33'
  });
  return result.confirm === true;
}
