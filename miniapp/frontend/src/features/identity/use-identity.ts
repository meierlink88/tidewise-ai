import { useEffect, useRef, useState } from 'react';
import { useDidShow } from '@tarojs/taro';
import { clearSession, readSession, requestWechatCode, saveSession } from '../../platform/identity';
import { readChosenAvatar, displayAvatar } from '../../platform/avatar';
import * as api from './api';
import type { Profile, Session } from './session';

export type IdentityAction = 'refresh' | 'login' | 'nickname' | 'logout';

export function useIdentity() {
  const [profile, setProfile] = useState<Profile | null>(null);
  const [pendingAction, setPendingAction] = useState<IdentityAction | null>(null);
  const busy = pendingAction !== null;
  const [error, setError] = useState('');
  const session = useRef<Session | null>(null);
  const pending = useRef(false);
  const mounted = useRef(true);
  const avatarSlot = useRef(0);
  async function withAvatar(p: Profile, token: string): Promise<Profile> {
    const data = await api.avatar(token);
    return { ...p, avatarSource: await displayAvatar(data, p.user_id, ++avatarSlot.current) };
  }
  useEffect(() => {
    mounted.current = true;
    return () => {
      mounted.current = false;
    };
  }, []);
  async function run(kind: IdentityAction, action: () => Promise<void>): Promise<boolean> {
    if (pending.current) return false;
    pending.current = true;
    setPendingAction(kind);
    setError('');
    try {
      await action();
      return true;
    } catch (e) {
      if (e instanceof api.IdentityError && e.expired) {
        session.current = null;
        try {
          clearSession();
        } catch {
          /* The server remains authoritative. */
        }
        if (mounted.current) setProfile(null);
      }
      if (mounted.current) setError(e instanceof Error ? e.message : '操作未完成，请重试');
      return false;
    } finally {
      pending.current = false;
      if (mounted.current) setPendingAction(null);
    }
  }
  function refresh() {
    return run('refresh', async () => {
      session.current = readSession();
      if (!session.current) {
        if (mounted.current) setProfile(null);
        return;
      }
      const result = await api.me(session.current.session_token);
      if (mounted.current) setProfile(result);
      const complete = await withAvatar(result, session.current.session_token);
      if (mounted.current) setProfile(complete);
    });
  }
  useDidShow(() => {
    void refresh();
  });
  return {
    profile,
    busy,
    pendingAction,
    error,
    refresh,
    login: (phoneCode?: string) =>
      run('login', async () => {
        const code = await requestWechatCode();
        const result = await api.login(code, session.current?.session_token, phoneCode);
        const next = { session_token: result.session_token, expires_at: result.expires_at };
        try {
          saveSession(next);
        } catch {
          await api.logout(next.session_token).catch(() => undefined);
          throw new Error('无法保存登录状态，请检查设备存储后重试');
        }
        session.current = next;
        if (mounted.current) setProfile(result);
      }),
    logout: () =>
      run('logout', async () => {
        if (session.current) await api.logout(session.current.session_token);
        session.current = null;
        if (mounted.current) setProfile(null);
        clearSession();
      }),
    saveNickname: (nickname: string, avatarPath?: string) =>
      run('nickname', async () => {
        if (!session.current) throw new api.IdentityError('请先登录', true);
        const data = avatarPath ? await readChosenAvatar(avatarPath) : undefined;
        const result = await api.updateNickname(
          session.current.session_token,
          nickname.trim(),
          data
        );
        if (mounted.current)
          setProfile({ ...result, avatarSource: avatarPath || profile?.avatarSource });
      })
  };
}
