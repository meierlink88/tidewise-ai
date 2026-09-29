import { useState } from 'react';
import { Button, Text, View, ScrollView, Checkbox, CheckboxGroup } from '@tarojs/components';
import type { IdentityAction } from '../../features/identity/use-identity';

export interface LoginViewProps {
  pendingAction: IdentityAction | null;
  canLogin: boolean;
  error: string;
  onLogin: (phoneCode: string) => Promise<boolean>;
  onOpenPrivacy: () => void;
}
export function LoginView(props: Readonly<LoginViewProps>) {
  const { pendingAction, canLogin, error } = props;
  const busy = pendingAction !== null;
  const [agreed, setAgreed] = useState(false);
  const [loginNotice, setLoginNotice] = useState('');
  return (
    <ScrollView scrollY className='login-page__body'>
      <View className='login-page__brand'>
        <View className='login-page__mark'>
          <Text>观</Text>
        </View>
        <Text className='login-page__name'>观潮家</Text>
      </View>
      <View className='profile-page__card profile-page__card--login'>
        <View className='profile-page__consent'>
          <CheckboxGroup
            onChange={(event) => {
              setAgreed(event.detail.value.includes('privacy'));
              setLoginNotice('');
            }}
          >
            <Checkbox
              value='privacy'
              checked={agreed}
              disabled={busy}
              color='#194d87'
              aria-label='同意隐私政策'
            >
              我已阅读并同意
            </Checkbox>
          </CheckboxGroup>
          <Button
            className='tidewise-button profile-page__policy-link'
            onClick={props.onOpenPrivacy}
          >
            《观潮家隐私政策》
          </Button>
        </View>
        <Button
          className='tidewise-button profile-page__button'
          hoverClass='profile-page__pressed'
          disabled={busy || !canLogin}
          openType={canLogin && agreed && !busy ? 'getPhoneNumber' : undefined}
          onClick={() => {
            if (!agreed) setLoginNotice('请先阅读并勾选隐私政策');
          }}
          onGetPhoneNumber={(event) => {
            if (!agreed || busy || !canLogin) return;
            if (!event.detail.code) {
              setLoginNotice('未完成手机号授权，请重试');
              return;
            }
            setLoginNotice('');
            void props.onLogin(event.detail.code);
          }}
        >
          {pendingAction === 'login'
            ? '正在登录…'
            : pendingAction === 'refresh'
              ? '正在检查登录状态…'
              : canLogin
                ? '手机号快捷登录'
                : '请在微信小程序中登录'}
        </Button>
        {loginNotice ? (
          <View className='profile-page__login-notice' role='alert'>
            {loginNotice}
          </View>
        ) : null}
        <Text className='profile-page__hint profile-page__hint--center'>
          未注册用户登录后自动创建观潮家账号
        </Text>
      </View>
      {error && (
        <View className='profile-page__error' role='alert'>
          {error}
        </View>
      )}
    </ScrollView>
  );
}
