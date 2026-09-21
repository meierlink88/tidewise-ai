import { useMemo, useRef } from 'react';
import Taro from '@tarojs/taro';
import { Button, View } from '@tarojs/components';
import { NavigationBar } from '../../platform/navigation-bar';
import { getHomeChromeMetrics } from '../../platform/system-ui';
import { leaveLogin, openProfileInformation, supportsWechatLogin } from '../../platform/identity';
import { useIdentity } from '../../features/identity/use-identity';
import { LoginView } from './login-view';
import '../profile/index.scss';
import './index.scss';

export default function LoginPage() {
  const chrome = useMemo(() => getHomeChromeMetrics(Taro), []);
  const identity = useIdentity();
  const leaving = useRef(false);
  async function login(phoneCode?: string) {
    if (leaving.current) return false;
    if (!(await identity.login(phoneCode))) return false;
    leaving.current = true;
    try {
      await leaveLogin();
    } finally {
      leaving.current = false;
    }
    return true;
  }
  return (
    <View className='login-page'>
      <NavigationBar
        title='欢迎登录观潮家'
        chrome={chrome}
        leading={
          <Button
            className='tidewise-button profile-page__back'
            aria-label='返回我的'
            onClick={() => void leaveLogin()}
          >
            <View className='profile-page__chevron' />
          </Button>
        }
      />
      <LoginView
        pendingAction={identity.pendingAction}
        error={identity.error}
        canLogin={supportsWechatLogin}
        onLogin={login}
        onOpenPrivacy={() => void openProfileInformation('privacy')}
      />
    </View>
  );
}
