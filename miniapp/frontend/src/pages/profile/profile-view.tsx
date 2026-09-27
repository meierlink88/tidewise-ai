import { Button, Image, ScrollView, Text, View } from '@tarojs/components';
import type { Profile } from '../../features/identity/session';
import type { IdentityAction } from '../../features/identity/use-identity';

export interface ProfileViewProps {
  profile: Profile | null;
  pendingAction: IdentityAction | null;
  error: string;
  onOpenLogin: () => void;
  onOpenTracking?: () => void;
  onOpenInformation: (section: 'privacy' | 'about') => void;
  onOpenEditor: () => void;
  onLogout: () => Promise<boolean>;
  onRetry: () => Promise<boolean>;
}

export function ProfileView(props: Readonly<ProfileViewProps>) {
  const { profile, pendingAction, error } = props;
  const busy = pendingAction !== null;
  const name = profile?.nickname || '观潮家用户';
  return (
    <View className='profile-page__body profile-page__body--landing'>
      <Button
        className='tidewise-button profile-page__identity profile-page__identity-button'
        disabled={busy}
        onClick={() => {
          if (profile) props.onOpenEditor();
          else props.onOpenLogin();
        }}
      >
        <View className='profile-page__portrait'>
          {profile?.avatarSource ? (
            <Image
              className='profile-page__saved-avatar'
              src={profile.avatarSource}
              mode='aspectFill'
            />
          ) : (
            <View className='profile-page__person' aria-hidden />
          )}
        </View>
        <View className='profile-page__identity-copy'>
          <Text className='profile-page__name'>{profile ? name : '登录/注册'}</Text>
          {profile && <Text className='profile-page__state'>欢迎回来，继续你的研究</Text>}
        </View>
        <View className='profile-page__row-chevron' aria-hidden />
      </Button>
      <ScrollView scrollY className='profile-page__scroll'>
        <>
          {props.onOpenTracking && (
            <View className='profile-page__group'>
              <Text className='profile-page__group-title'>我的研究</Text>
              <Button
                className='tidewise-button profile-page__about-row'
                onClick={props.onOpenTracking}
              >
                <View className='profile-page__about-mark'>
                  <View className='profile-page__bookmark' />
                </View>
                <View className='profile-page__row-copy'>
                  <Text>我的跟踪</Text>
                  <Text className='profile-page__row-note'>查看已跟踪公司</Text>
                </View>
                <View className='profile-page__row-chevron' aria-hidden />
              </Button>
            </View>
          )}
          <View className='profile-page__group'>
            <Text className='profile-page__group-title'>账户与支持</Text>
            {profile && (
              <Button
                className='tidewise-button profile-page__about-row'
                disabled={busy}
                onClick={props.onOpenEditor}
              >
                <View className='profile-page__about-mark'>
                  <View className='profile-page__person' aria-hidden />
                </View>
                <View className='profile-page__row-copy'>
                  <Text>个人资料</Text>
                  <Text className='profile-page__row-note'>头像与昵称</Text>
                </View>
                <View className='profile-page__row-chevron' aria-hidden />
              </Button>
            )}
            <Button
              className='tidewise-button profile-page__about-row'
              onClick={() => props.onOpenInformation('about')}
            >
              <View className='profile-page__about-mark'>
                <View className='profile-page__info-icon' aria-hidden />
              </View>
              <View className='profile-page__row-copy'>
                <Text>关于观潮家</Text>
                <Text className='profile-page__row-note'>读懂全球政经变化</Text>
              </View>
              <View className='profile-page__row-chevron' aria-hidden />
            </Button>
            <Button
              className='tidewise-button profile-page__about-row'
              onClick={() => props.onOpenInformation('privacy')}
            >
              <View className='profile-page__about-mark'>
                <View className='profile-page__lock-icon' />
              </View>
              <View className='profile-page__row-copy'>
                <Text>隐私政策</Text>
              </View>
              <View className='profile-page__row-chevron' aria-hidden />
            </Button>
          </View>
        </>
        {profile && (
          <View className='profile-page__session'>
            <Button
              className='tidewise-button profile-page__logout'
              disabled={busy}
              onClick={() => void props.onLogout()}
            >
              <View className='profile-page__logout-icon' aria-hidden />
              {pendingAction === 'logout' ? '正在退出…' : '退出登录'}
            </Button>
          </View>
        )}
        {error && (
          <View className='profile-page__error' role='alert'>
            <Text>{error}</Text>
            <Button
              className='tidewise-button profile-page__retry'
              disabled={busy}
              onClick={() => void props.onRetry()}
            >
              重新检查登录状态
            </Button>
          </View>
        )}
      </ScrollView>
    </View>
  );
}
