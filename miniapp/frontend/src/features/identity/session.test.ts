import { describe, expect, it } from 'vitest';
import { isSession, isProfile } from './session';

describe('stored identity boundary', () => {
  it('rejects expired and malformed credentials', () => {
    const valid = { session_token: 'A'.repeat(43), expires_at: '2099-01-01T00:00:00Z' };
    expect(isSession(valid)).toBe(true);
    expect(isSession({ ...valid, expires_at: '2000-01-01T00:00:00Z' })).toBe(false);
    expect(isSession({ ...valid, session_token: 'anything' })).toBe(false);
    expect(isSession(null)).toBe(false);
  });
  it('does not accept disabled or incomplete profiles', () => {
    const valid = {
      user_id: 'f466d548-4ac3-4d3f-b994-cbeeb6eb3ef2',
      nickname: '观潮用户',
      status: 'active',
      expires_at: '2099-01-01T00:00:00Z'
    };
    expect(isProfile(valid)).toBe(true);
    expect(isProfile({ ...valid, status: 'disabled' })).toBe(false);
    expect(isProfile({ user_id: valid.user_id })).toBe(false);
  });
});
