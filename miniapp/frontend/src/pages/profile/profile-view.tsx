import { useState } from 'react';
import { Button, Form, Image, Input, Text, View } from '@tarojs/components';
import { supportsWechatLogin } from '../../platform/identity';
import type { Profile } from '../../features/identity/session';
import type { IdentityAction } from '../../features/identity/use-identity';

export interface ProfileViewProps {
  profile: Profile | null;
  pendingAction: IdentityAction | null;
  error: string;
  onOpenLogin: () => void;
  onOpenInformation: (section: 'privacy' | 'about') => void;
  onSaveNickname: (nickname: string, avatarPath?: string) => Promise<boolean>;
  onLogout: () => Promise<boolean>;
  onRetry: () => Promise<boolean>;
}

export function ProfileView(props: Readonly<ProfileViewProps>) {
  const { profile, pendingAction, error } = props;
  const [editing, setEditing] = useState(false);
  const [accountOpen, setAccountOpen] = useState(false);
  const [draft, setDraft] = useState('');
  const [avatarDraft, setAvatarDraft] = useState('');
  const [notice, setNotice] = useState('');
  const busy = pendingAction !== null;
  const name = profile?.nickname || '观潮家用户';
  const trimmed = draft.trim();
  const invalid =
    !trimmed || [...trimmed].length > 32 || /[\u0000-\u001f\u007f-\u009f]/.test(trimmed);
  function edit() {
    setDraft(profile?.nickname || '');
    setAvatarDraft('');
    setNotice('');
    setEditing(true);
  }
  async function save(submitted?: string) {
    const nameToSave = (submitted ?? draft).trim();
    if (
      busy ||
      !nameToSave ||
      [...nameToSave].length > 32 ||
      /[\u0000-\u001f\u007f-\u009f]/.test(nameToSave)
    ) {
      setNotice('请填写有效昵称，最多 32 个字符');
      return;
    }
    const saved = await props.onSaveNickname(nameToSave, avatarDraft || undefined);
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
    <View
      className={`profile-page__body${accountOpen ? ' profile-page__body--account' : ' profile-page__body--landing'}`}
    >
      <Button
        className='profile-page__identity profile-page__identity-button'
        disabled={busy}
        onClick={() => {
          if (profile) setAccountOpen(true);
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
            <View className='profile-page__person' aria-hidden>
              <View className='profile-page__person-head' />
              <View className='profile-page__person-body' />
            </View>
          )}
        </View>
        <View className='profile-page__identity-copy'>
          <Text className='profile-page__name'>{profile ? name : '登录/注册'}</Text>
        </View>
      </Button>
      {profile && accountOpen ? (
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
              <Form onSubmit={(event) => void save(String(event.detail.value?.nickname ?? ''))}>
                {supportsWechatLogin && (
                  <>
                    <Button
                      className='profile-page__avatar-picker'
                      openType='chooseAvatar'
                      disabled={busy}
                      onChooseAvatar={(event) => {
                        const detail = event.detail as { avatarUrl?: string };
                        if (detail.avatarUrl) {
                          setAvatarDraft(detail.avatarUrl);
                          setNotice('');
                        }
                      }}
                    >
                      {(avatarDraft || profile.avatarSource) && (
                        <Image
                          className='profile-page__saved-avatar'
                          src={avatarDraft || profile.avatarSource || ''}
                          mode='aspectFill'
                        />
                      )}
                      <Text>{avatarDraft ? '重新选择头像' : '选择微信头像'}</Text>
                    </Button>
                    <Text className='profile-page__hint'>
                      头像和昵称仅用于观潮家个人资料展示，选择后点击保存。可从关于页查看隐私政策。
                    </Text>
                  </>
                )}

                <Text className='profile-page__label'>昵称</Text>
                <Input
                  className='profile-page__input'
                  aria-label='昵称'
                  name='nickname'
                  type={supportsWechatLogin ? 'nickname' : 'text'}
                  value={draft}
                  maxlength={32}
                  placeholder='输入你的昵称'
                  disabled={busy}
                  focus
                  onInput={(event) => setDraft(event.detail.value)}
                  confirmType='done'
                  onBlur={(event) => setDraft(event.detail.value)}
                />
                <Text className='profile-page__hint'>可选用微信昵称或自行填写，最多 32 个字符</Text>
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
                    disabled={
                      busy ||
                      (!supportsWechatLogin && invalid) ||
                      (!supportsWechatLogin && trimmed === profile.nickname && !avatarDraft)
                    }
                    formType='submit'
                  >
                    {pendingAction === 'nickname' ? '保存中…' : '保存修改'}
                  </Button>
                </View>
              </Form>
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
      ) : null}
      {accountOpen && !editing && (
        <Button className='profile-page__phone-login' onClick={() => setAccountOpen(false)}>
          返回我的
        </Button>
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
      {!accountOpen && (
        <Button
          className='profile-page__about-row'
          onClick={() => props.onOpenInformation('about')}
        >
          <Text className='profile-page__about-mark'>观</Text>
          <Text>关于观潮家</Text>
          <Text className='profile-page__about-arrow'>›</Text>
        </Button>
      )}
    </View>
  );
}
