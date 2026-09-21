import { useMemo, useRef, useState } from 'react';
import Taro from '@tarojs/taro';
import { Button, View } from '@tarojs/components';
import { NavigationBar } from '../../platform/navigation-bar';
import { getHomeChromeMetrics } from '../../platform/system-ui';
import {
  confirmLogout,
  leaveProfile,
  openLogin,
  openProfileInformation
} from '../../platform/identity';
import { useIdentity } from '../../features/identity/use-identity';
import { ProfileView } from './profile-view';
import './index.scss';

export default function ProfilePage() {
  const chrome = useMemo(() => getHomeChromeMetrics(Taro), []);
  const identity = useIdentity();
  const confirming = useRef(false);
  const [confirmationError, setConfirmationError] = useState('');
  async function logout() {
    if (confirming.current || identity.busy) return false;
    confirming.current = true;
    setConfirmationError('');
    try {
      if (!(await confirmLogout())) return false;
      return await identity.logout();
    } catch {
      setConfirmationError('暂时无法打开确认窗口，请重试');
      return false;
    } finally {
      confirming.current = false;
    }
  }
  return (
    <View className='profile-page profile-page--personal account-page'>
      <View className='profile-page__header account-header'>
        <NavigationBar
          title='我的'
          chrome={chrome}
          leading={
            <Button
              className='tidewise-button profile-page__back'
              aria-label='返回推理'
              hoverClass='none'
              onClick={() => void leaveProfile()}
            >
              <View className='profile-page__chevron' />
            </Button>
          }
        />
      </View>
      <ProfileView
        key={identity.profile?.user_id || 'guest'}
        profile={identity.profile}
        pendingAction={identity.pendingAction}
        error={confirmationError || identity.error}
        onOpenLogin={() => void openLogin()}
        onOpenInformation={(section) => void openProfileInformation(section)}
        onSaveNickname={identity.saveNickname}
        onLogout={logout}
        onRetry={identity.refresh}
      />
    </View>
  );
}
