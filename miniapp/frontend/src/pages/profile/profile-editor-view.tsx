import { useRef, useState } from 'react';
import { Button, Form, Image, Input, Text, View } from '@tarojs/components';
import { supportsWechatLogin, profileSubmitOnClick } from '../../platform/identity';
import type { Profile } from '../../features/identity/session';
import type { IdentityAction } from '../../features/identity/use-identity';

export interface ProfileEditorViewProps {
  profile: Profile;
  pendingAction: IdentityAction | null;
  error: string;
  onSaveNickname: (nickname: string, avatarPath?: string) => Promise<boolean>;
  onSaved: () => void;
  onCancel: () => void;
}

export function ProfileEditorView(props: Readonly<ProfileEditorViewProps>) {
  const { profile, pendingAction, error } = props;
  const [draft, setDraft] = useState(profile.nickname);
  const [avatarDraft, setAvatarDraft] = useState('');
  const [notice, setNotice] = useState('');
  const submitting = useRef(false);
  const busy = pendingAction !== null;
  const trimmed = draft.trim();
  const invalid =
    !trimmed || [...trimmed].length > 32 || /[\u0000-\u001f\u007f-\u009f]/.test(trimmed);
  async function save(submitted: string) {
    if (busy || submitting.current) return;
    const nickname = submitted.trim();
    if (!nickname || [...nickname].length > 32 || /[\u0000-\u001f\u007f-\u009f]/.test(nickname)) {
      setNotice('请填写有效昵称，最多 32 个字符');
      return;
    }
    submitting.current = true;
    try {
      if (await props.onSaveNickname(nickname, avatarDraft || undefined)) props.onSaved();
    } finally {
      submitting.current = false;
    }
  }
  const portrait = (
    <View className='profile-editor__portrait'>
      {avatarDraft || profile.avatarSource ? (
        <Image
          className='profile-editor__avatar'
          src={avatarDraft || profile.avatarSource || ''}
          mode='aspectFill'
        />
      ) : (
        <View className='profile-page__person' aria-hidden />
      )}
    </View>
  );
  return (
    <Form onSubmit={(event) => void save(String(event.detail.value?.nickname ?? ''))}>
      <View className='profile-editor__avatar-area'>
        {supportsWechatLogin ? (
          <Button
            className='tidewise-button profile-editor__avatar-picker'
            openType='chooseAvatar'
            disabled={busy || undefined}
            onChooseAvatar={(event) => {
              const detail = event.detail as { avatarUrl?: string };
              if (detail.avatarUrl) {
                setAvatarDraft(detail.avatarUrl);
                setNotice('');
              }
            }}
          >
            {portrait}
            <Text className='profile-editor__avatar-label'>
              {avatarDraft ? '重新选择头像' : '选择微信头像'}
            </Text>
          </Button>
        ) : (
          <>
            {portrait}
            <Text className='profile-editor__avatar-label'>请在微信小程序中修改头像</Text>
          </>
        )}
      </View>
      <Text className='profile-editor__label'>昵称</Text>
      <Input
        className='profile-editor__input'
        aria-label='昵称'
        name='nickname'
        type={supportsWechatLogin ? 'nickname' : 'text'}
        value={draft}
        maxlength={32}
        placeholder='输入你的昵称'
        disabled={busy || undefined}
        confirmType='done'
        onInput={(event) => setDraft(event.detail.value)}
        onBlur={(event) => setDraft(event.detail.value)}
      />
      <Text className='profile-editor__hint'>最多 32 个字符，保存后在我的页面展示</Text>
      {(error || notice) && (
        <View className='profile-editor__error' role='alert'>
          {error || notice}
        </View>
      )}
      <Button
        className='tidewise-button profile-editor__save'
        formType={profileSubmitOnClick ? undefined : 'submit'}
        onClick={profileSubmitOnClick ? () => void save(draft) : undefined}
        disabled={busy || (!supportsWechatLogin && invalid) || undefined}
      >
        {pendingAction === 'nickname' ? '保存中…' : '保存修改'}
      </Button>
      <Button
        className='tidewise-button profile-editor__cancel'
        disabled={busy || undefined}
        onClick={props.onCancel}
      >
        放弃修改
      </Button>
    </Form>
  );
}
