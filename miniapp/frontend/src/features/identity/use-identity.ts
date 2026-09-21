import { useEffect, useRef, useState } from 'react';
import { useDidShow } from '@tarojs/taro';
import { clearSession, readSession, requestWechatCode, saveSession } from '../../platform/identity';
import * as api from './api';
import type { Profile, Session } from './session';

export function useIdentity() {
  const [profile, setProfile] = useState<Profile | null>(null);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState('');
  const session = useRef<Session | null>(null);
  const pending = useRef(false);
  const mounted = useRef(true);
  useEffect(() => {
    mounted.current = true;
    return () => {
      mounted.current = false;
    };
  }, []);
  async function run(action: () => Promise<void>) {
    if (pending.current) return;
    pending.current = true;
    setBusy(true);
    setError('');
    try {
      await action();
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
    } finally {
      pending.current = false;
      if (mounted.current) setBusy(false);
    }
  }
  function refresh() {
    return run(async () => {
      session.current = readSession();
      if (!session.current) {
        if (mounted.current) setProfile(null);
        return;
      }
      const result = await api.me(session.current.session_token);
      if (mounted.current) setProfile(result);
    });
  }
  useDidShow(() => {
    void refresh();
  });
  return {
    profile,
    busy,
    error,
    refresh,
    login: () =>
      run(async () => {
        const code = await requestWechatCode();
        const result = await api.login(code, session.current?.session_token);
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
      run(async () => {
        if (session.current) await api.logout(session.current.session_token);
        session.current = null;
        if (mounted.current) setProfile(null);
        clearSession();
      }),
    saveNickname: (nickname: string) =>
      run(async () => {
        if (!session.current) throw new api.IdentityError('请先登录', true);
        const result = await api.updateNickname(session.current.session_token, nickname.trim());
        if (mounted.current) setProfile(result);
      })
  };
}
