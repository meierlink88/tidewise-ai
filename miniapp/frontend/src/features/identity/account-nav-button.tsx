import { useRef } from 'react';
import { Button, Image, Text, View } from '@tarojs/components';
import { openProfile } from '../../platform/identity';
import { useIdentity } from './use-identity';
import avatarImage from '../../assets/nav-avatar.png';
import './account-nav-button.scss';

export function AccountNavButton() {
  const identity = useIdentity({ refreshOnMount: true });
  const openingProfile = useRef(false);
  async function enterProfile() {
    if (openingProfile.current) return;
    openingProfile.current = true;
    try {
      await openProfile();
    } finally {
      openingProfile.current = false;
    }
  }
  return (
    <Button
      className='tidewise-button home-nav__identity-button'
      hoverClass='none'
      aria-label='个人中心'
      onClick={() => void enterProfile()}
    >
      <View className='home-nav__avatar-frame'>
        <Image
          className='home-nav__avatar'
          src={identity.profile?.avatarSource || avatarImage}
          mode='aspectFill'
        />
      </View>
      {identity.profile && (
        <Text className='home-nav__nickname'>{identity.profile.nickname || '观潮家用户'}</Text>
      )}
    </Button>
  );
}
