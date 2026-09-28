import { useRef, useState } from 'react';
import { Button, ScrollView, Text, View } from '@tarojs/components';
import Taro, { useDidHide } from '@tarojs/taro';
import { useTracking } from '../../features/tracking/use-tracking';
import type { Company } from '../../features/tracking/contract';
import { openLogin, readSession } from '../../platform/identity';
import { confirmUnfollow } from '../../platform/tracking';
import './index.scss';
import { CompanyCard } from '../../features/tracking/company-card';

export default function TrackingPage() {
  const tracking = useTracking();
  const confirmPending = useRef(false);
  const visibility = useRef(0);
  const [confirmation, setConfirmation] = useState('');
  const [modalError, setModalError] = useState('');
  useDidHide(() => {
    visibility.current++;
    setConfirmation('');
  });
  const items = tracking.items;
  const status = tracking.status;
  async function remove(item: Company) {
    if (confirmPending.current || tracking.pending) return;
    confirmPending.current = true;
    const visible = visibility.current;
    const owner = readSession()?.session_token;
    setConfirmation(item.id);
    setModalError('');
    try {
      if (await confirmUnfollow(item.title, item.symbol)) {
        if (visible === visibility.current && owner && readSession()?.session_token === owner)
          await tracking.change(item, false);
      }
    } catch {
      setModalError('未能打开确认窗口，请重试');
    } finally {
      confirmPending.current = false;
      setConfirmation('');
    }
  }
  async function discover() {
    try {
      await Taro.switchTab({ url: '/pages/company/index' });
    } catch {
      void Taro.showToast({ title: '打开失败，请重试', icon: 'none' });
    }
  }
  return (
    <View className='tracking-page'>
      <View className='tracking-header'>
        <View className='tracking-caption'>
          <Text>{tracking.guest ? '关注公司，持续跟踪' : `已跟踪 ${tracking.total} 家公司`}</Text>
        </View>
      </View>
      <ScrollView
        scrollY
        className='tracking-content'
        onScrollToLower={() => {
          if (tracking.hasMore && status === 'ready' && !tracking.pending && !confirmation)
            void tracking.loadMore();
        }}
      >
        {(tracking.error || modalError) && (
          <View className='tracking-error'>
            <Text>{modalError || tracking.error}</Text>
            <Button
              className='tidewise-button'
              onClick={() => {
                setModalError('');
                void tracking.retry();
              }}
            >
              重试
            </Button>
          </View>
        )}
        {items.map((item) => (
          <CompanyCard
            key={item.id}
            company={item}
            busy={!!tracking.pending || !!confirmation}
            pending={tracking.pending === item.id}
            onRemove={() => void remove(item)}
          />
        ))}
        {status === 'loading' && <View className='tracking-message'>加载中…</View>}
        {items.length === 0 && status !== 'loading' && status !== 'error' && (
          <View className='tracking-empty'>
            <Text className='tracking-empty-title'>
              {tracking.guest ? '登录后查看你的跟踪' : '还没有跟踪公司'}
            </Text>
            <Text>到公司洞察发现并跟踪感兴趣的公司</Text>
            <Button className='tidewise-button tracking-login' onClick={() => void discover()}>
              去公司洞察
            </Button>
            {tracking.guest && (
              <Button className='tidewise-button tracking-login' onClick={() => void openLogin()}>
                登录 / 注册
              </Button>
            )}
          </View>
        )}
        {items.length > 0 && !tracking.hasMore && status === 'ready' && (
          <View className='tracking-message'>以上是你跟踪的公司</View>
        )}
      </ScrollView>
    </View>
  );
}
