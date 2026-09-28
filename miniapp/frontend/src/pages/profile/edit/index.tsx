import { useMemo } from 'react';
import Taro from '@tarojs/taro';
import { Button, ScrollView, Text, View } from '@tarojs/components';
import { NavigationBar } from '../../../platform/navigation-bar';
import { getHomeChromeMetrics } from '../../../platform/system-ui';
import { leaveProfileEditor, openLogin } from '../../../platform/identity';
import { useIdentity } from '../../../features/identity/use-identity';
import { ProfileEditorView } from '../profile-editor-view';
import './index.scss';

export default function ProfileEditPage() {
  const chrome = useMemo(() => getHomeChromeMetrics(Taro), []);
  const identity = useIdentity();
  return (
    <View className='profile-editor'>
      <NavigationBar
        title='个人资料'
        chrome={chrome}
        leading={
          <Button
            className='tidewise-button profile-editor__back'
            aria-label='返回我的'
            disabled={identity.busy}
            onClick={() => void leaveProfileEditor()}
          >
            <View className='profile-editor__back-icon' aria-hidden />
          </Button>
        }
      />
      <ScrollView scrollY className='profile-editor__scroll'>
        <View className='profile-editor__content'>
          {identity.profile ? (
            <ProfileEditorView
              key={identity.profile.user_id}
              profile={identity.profile}
              pendingAction={identity.pendingAction}
              error={identity.error}
              onSaveNickname={identity.saveNickname}
              onSaved={() => void leaveProfileEditor()}
              onCancel={() => void leaveProfileEditor()}
            />
          ) : (
            <View className='profile-editor__status'>
              <Text>
                {identity.pendingAction === 'refresh'
                  ? '正在检查登录状态…'
                  : identity.error || '请先登录后修改个人资料'}
              </Text>
              {!identity.busy && (
                <Button
                  className='tidewise-button profile-editor__save'
                  onClick={() => void openLogin()}
                >
                  前往登录
                </Button>
              )}
            </View>
          )}
        </View>
      </ScrollView>
    </View>
  );
}
