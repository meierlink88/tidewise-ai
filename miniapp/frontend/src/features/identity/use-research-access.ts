import { useEffect, useRef, useState } from 'react';
import { useDidHide, useDidShow } from '@tarojs/taro';
import {
  clearSession,
  readSession,
  requireResearchLogin,
  type ResearchDestination
} from '../../platform/identity';
import { IdentityError, me } from './api';

export function useResearchAccess(destination: ResearchDestination) {
  const [allowed, setAllowed] = useState(false);
  const [error, setError] = useState('');
  const sequence = useRef(0);
  const expiry = useRef<ReturnType<typeof setTimeout>>();
  const pending = useRef(false);
  async function check() {
    if (pending.current) return;
    pending.current = true;
    const seq = ++sequence.current;
    clearTimeout(expiry.current);
    setAllowed(false);
    setError('');
    const session = readSession();
    try {
      if (!session) {
        await requireResearchLogin(destination);
        return;
      }
      await me(session.session_token);
      if (seq !== sequence.current) return;
      if (readSession()?.session_token !== session.session_token) {
        await requireResearchLogin(destination);
        return;
      }
      setAllowed(true);
      // Recheck at expiry, also for a page left open without another onShow.
      expiry.current = setTimeout(
        () => void check(),
        Math.min(2147483647, Math.max(0, Date.parse(session.expires_at) - Date.now()))
      );
    } catch (e) {
      if (seq !== sequence.current) return;
      if (e instanceof IdentityError && e.expired) {
        if (readSession()?.session_token === session?.session_token) {
          try {
            clearSession();
          } catch {
            /* Rejected identity is never displayed. */
          }
        }
        try {
          await requireResearchLogin(destination);
        } catch {
          if (seq === sequence.current) setError('暂时无法打开登录页，请重试');
        }
      } else setError(e instanceof Error ? e.message : '暂时无法检查登录状态，请重试');
    } finally {
      if (seq === sequence.current) pending.current = false;
    }
  }
  function invalidate() {
    ++sequence.current;
    pending.current = false;
    clearTimeout(expiry.current);
  }
  useDidShow(() => void check());
  useDidHide(() => {
    invalidate();
    setAllowed(false);
  });
  useEffect(() => invalidate, []);
  return { allowed, error, retry: check };
}
