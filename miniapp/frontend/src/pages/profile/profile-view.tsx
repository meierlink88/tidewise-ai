import { useState } from 'react';
import { Button, Image, Input, Text, View } from '@tarojs/components';
import type { Profile } from '../../features/identity/session';
import type { IdentityAction } from '../../features/identity/use-identity';
import avatar from '../../assets/tab/profile-active.png';

export interface ProfileViewProps {
  profile: Profile | null;
  pendingAction: IdentityAction | null;
  error: string;
  canLogin: boolean;
  onLogin: () => Promise<boolean>;
  onSaveNickname: (nickname: string) => Promise<boolean>;
  onLogout: () => Promise<boolean>;
  onRetry: () => Promise<boolean>;
}

export function ProfileView(props: Readonly<ProfileViewProps>) {
  const { profile, pendingAction, error, canLogin } = props;
  const [editing, setEditing] = useState(false);
  const [draft, setDraft] = useState('');
  const [notice, setNotice] = useState('');
  const busy = pendingAction !== null;
  const name = profile?.nickname || '观潮家用户';
  const trimmed = draft.trim();
  const invalid =
    !trimmed || [...trimmed].length > 32 || /[\u0000-\u001f\u007f-\u009f]/.test(trimmed);
  function edit() {
    setDraft(profile?.nickname || '');
    setNotice('');
    setEditing(true);
  }
  async function save() {
    if (busy || invalid) return;
    const saved = await props.onSaveNickname(trimmed);
    if (saved) {
      setEditing(false);
      setNotice('个人资料已更新');
    }
  }
  async function logout() {
    setNotice('');
    if (await props.onLogout()) {
      setEditing(false);
      setDraft('');
    }
  }
  return (
    <View className='profile-page__body'>
      <View className='profile-page__identity'>
        <View className='profile-page__portrait'>
          <Image className='profile-page__avatar' src={avatar} mode='aspectFit' />
        </View>
        <View className='profile-page__identity-copy'>
          <Text className='profile-page__name'>{profile ? name : '欢迎来到观潮家'}</Text>
          <Text className='profile-page__state'>
            {profile ? '微信账号已登录' : '读懂全球政经变化'}
          </Text>
        </View>
      </View>
      {profile ? (
        <>
          <View className='profile-page__card'>
            <View className='profile-page__card-heading'>
              <Text className='profile-page__heading'>{editing ? '编辑个人资料' : '个人资料'}</Text>
              {!editing && (
                <Button
                  className='profile-page__edit'
                  hoverClass='profile-page__pressed'
                  disabled={busy}
                  onClick={edit}
                >
                  编辑资料
                </Button>
              )}
            </View>
            {editing ? (
              <>
                <Text className='profile-page__label'>昵称</Text>
                <Input
                  className='profile-page__input'
                  aria-label='昵称'
                  type='text'
                  value={draft}
                  maxlength={32}
                  placeholder='输入你的昵称'
                  disabled={busy}
                  focus
                  onInput={(event) => setDraft(event.detail.value)}
                  confirmType='done'
                  onConfirm={() => void save()}
                />
                <Text className='profile-page__hint'>昵称最多 32 个字符</Text>
                <View className='profile-page__actions'>
                  <Button
                    className='profile-page__button profile-page__button--quiet'
                    disabled={busy}
                    onClick={() => {
                      setEditing(false);
                      setNotice('');
                    }}
                  >
                    取消
                  </Button>
                  <Button
                    className='profile-page__button'
                    hoverClass='profile-page__pressed'
                    disabled={busy || invalid || trimmed === profile.nickname}
                    onClick={() => void save()}
                  >
                    {pendingAction === 'nickname' ? '保存中…' : '保存修改'}
                  </Button>
                </View>
              </>
            ) : (
              <View className='profile-page__detail-row'>
                <Text className='profile-page__detail-label'>昵称</Text>
                <Text className='profile-page__detail-value'>{profile.nickname || '尚未设置'}</Text>
              </View>
            )}
            {notice && !error ? (
              <View className='profile-page__notice' role='status'>
                {notice}
              </View>
            ) : null}
          </View>
          {!editing && (
            <View className='profile-page__session'>
              <Button
                className='profile-page__logout'
                hoverClass='profile-page__pressed'
                disabled={busy}
                onClick={() => void logout()}
              >
                {pendingAction === 'logout' ? '正在退出…' : '退出登录'}
              </Button>
              <Text className='profile-page__hint profile-page__hint--center'>
                退出后仍可浏览推理内容
              </Text>
            </View>
          )}
        </>
      ) : (
        <View className='profile-page__card profile-page__card--login'>
          <Text className='profile-page__heading'>登录观潮家</Text>
          <Text className='profile-page__description'>
            使用微信账号快捷登录，保存你的个人资料。
          </Text>
          <Button
            className='profile-page__button'
            hoverClass='profile-page__pressed'
            disabled={busy || !canLogin}
            onClick={() => void props.onLogin()}
          >
            {pendingAction === 'login'
              ? '正在登录…'
              : pendingAction === 'refresh'
                ? '正在检查登录状态…'
                : canLogin
                  ? '微信登录'
                  : '请在微信小程序中登录'}
          </Button>
          <Text className='profile-page__hint profile-page__hint--center'>
            无需填写手机号 · 未登录也可浏览推理
          </Text>
        </View>
      )}
      {error ? (
        <View className='profile-page__error' role='alert'>
          <Text>{error}</Text>
          {!editing && (
            <Button
              className='profile-page__retry'
              disabled={busy}
              onClick={() => void props.onRetry()}
            >
              重新检查登录状态
            </Button>
          )}
        </View>
      ) : null}
      <View className='profile-page__footer'>
        <Text>观潮家</Text>
        <Text className='profile-page__footer-caption'>读懂全球政经变化</Text>
      </View>
    </View>
  );
}
