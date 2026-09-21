import { useEffect, useMemo, useState } from 'react';
import Taro from '@tarojs/taro';
import { Button, Image, Input, Text, View } from '@tarojs/components';
import { NavigationBar } from '../../platform/navigation-bar';
import { getHomeChromeMetrics } from '../../platform/system-ui';
import { supportsWechatLogin } from '../../platform/identity';
import { useIdentity } from '../../features/identity/use-identity';
import avatar from '../../assets/tab/profile-active.png';
import './index.scss';

export default function ProfilePage() {
  const chrome = useMemo(() => getHomeChromeMetrics(Taro), []);
  const identity = useIdentity();
  const [nickname, setNickname] = useState('');
  useEffect(() => {
    setNickname(identity.profile?.nickname || '');
  }, [identity.profile?.nickname]);
  return (
    <View className='profile-page'>
      <NavigationBar title='我的' chrome={chrome} leading={null} />
      <View className='profile-page__body'>
        <View className='profile-page__identity'>
          <Image className='profile-page__avatar' src={avatar} mode='aspectFill' />
          <View>
            <Text className='profile-page__name'>
              {identity.profile ? identity.profile.nickname || '观潮家用户' : '未登录'}
            </Text>
            <Text className='profile-page__state'>
              {identity.profile ? '微信账号已登录' : '登录你的观潮家账号'}
            </Text>
          </View>
        </View>
        <View className='profile-page__card'>
          <Text className='profile-page__heading'>{identity.profile ? '已登录' : '微信登录'}</Text>
          <Text className='profile-page__description'>
            {identity.profile
              ? '下次打开小程序，可继续使用当前账号。'
              : '使用微信账号登录，无需填写手机号。'}
          </Text>
          {identity.profile ? (
            <>
              <Text className='profile-page__label'>用户昵称</Text>
              <Input
                className='profile-page__input'
                value={nickname}
                maxlength={32}
                placeholder='填写昵称'
                disabled={identity.busy}
                onInput={(event) => setNickname(event.detail.value)}
              />
              <Button
                className='profile-page__button'
                disabled={
                  identity.busy || !nickname.trim() || nickname.trim() === identity.profile.nickname
                }
                onClick={() => void identity.saveNickname(nickname)}
              >
                保存昵称
              </Button>
              <Button
                className='profile-page__button profile-page__button--secondary'
                disabled={identity.busy}
                onClick={() => void identity.logout()}
              >
                退出登录
              </Button>
            </>
          ) : (
            <Button
              className='profile-page__button'
              loading={identity.busy}
              disabled={identity.busy || !supportsWechatLogin}
              onClick={() => void identity.login()}
            >
              {supportsWechatLogin ? '微信登录' : '请在微信小程序中登录'}
            </Button>
          )}
          {identity.error ? (
            <View className='profile-page__error'>
              <Text>{identity.error}</Text>
              <Button
                className='profile-page__retry'
                disabled={identity.busy}
                onClick={() => void identity.refresh()}
              >
                重新检查登录状态
              </Button>
            </View>
          ) : null}
        </View>
        <Text className='profile-page__footer'>观潮家 · 读懂全球政经变化</Text>
      </View>
    </View>
  );
}
