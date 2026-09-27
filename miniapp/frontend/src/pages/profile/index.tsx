import { useMemo, useRef, useState } from 'react';
import Taro from '@tarojs/taro';
import { Button, Text, View } from '@tarojs/components';
import { RiskNotice } from '../../components/risk-notice';
import { NavigationBar } from '../../platform/navigation-bar';
import { getHomeChromeMetrics } from '../../platform/system-ui';
import {
  confirmLogout,
  leaveProfile,
  openLogin,
  openProfileEditor,
  openProfileInformation,
  supportsWechatLogin
} from '../../platform/identity';
import { useIdentity } from '../../features/identity/use-identity';
import { ProfileView } from './profile-view';
import { LoginView } from '../login/login-view';
import { openTracking } from '../../platform/tracking';
import './index.scss';
import '../login/index.scss';

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
    <View
      className={`profile-page profile-page--personal account-page${identity.profile ? ' profile-page--signed-in' : ''}`}
    >
      {identity.profile ? (
        <View
          className='profile-page__brand-cap'
          style={{ height: `${chrome.statusBarHeight + chrome.navigationBarHeight}px` }}
        />
      ) : (
        <View className='profile-page__header account-header'>
          <NavigationBar
            title='欢迎登录观潮家'
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
      )}
      {!identity.profile && identity.pendingAction === 'refresh' ? (
        <View className='profile-page__body'>
          <Text>正在检查登录状态…</Text>
        </View>
      ) : !identity.profile ? (
        <LoginView
          pendingAction={identity.pendingAction}
          error={identity.error}
          canLogin={supportsWechatLogin}
          onLogin={identity.login}
          onOpenPrivacy={() => void openProfileInformation('privacy')}
        />
      ) : (
        <ProfileView
          key={identity.profile?.user_id || 'guest'}
          profile={identity.profile}
          pendingAction={identity.pendingAction}
          error={confirmationError || identity.error}
          onOpenLogin={() => void openLogin()}
          onOpenInformation={(section) => void openProfileInformation(section)}
          onOpenEditor={() => void openProfileEditor()}
          onLogout={logout}
          onRetry={identity.refresh}
          onOpenTracking={() => void openTracking()}
        />
      )}
      <RiskNotice />
    </View>
  );
}
