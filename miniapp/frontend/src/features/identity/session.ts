export interface Session {
  session_token: string;
  expires_at: string;
}
export interface Profile {
  user_id: string;
  nickname: string;
  status: 'active';
  expires_at: string;
}
export function isSession(value: unknown): value is Session {
  if (!value || typeof value !== 'object') return false;
  const s = value as Session;
  return (
    typeof s.session_token === 'string' &&
    /^[A-Za-z0-9_-]{43}$/.test(s.session_token) &&
    typeof s.expires_at === 'string' &&
    Date.parse(s.expires_at) > Date.now()
  );
}
export function isProfile(value: unknown): value is Profile {
  if (!value || typeof value !== 'object') return false;
  const p = value as Profile;
  return (
    typeof p.user_id === 'string' &&
    /^[0-9a-f-]{36}$/i.test(p.user_id) &&
    typeof p.nickname === 'string' &&
    p.status === 'active' &&
    typeof p.expires_at === 'string' &&
    Number.isFinite(Date.parse(p.expires_at))
  );
}
