import { expect, it, vi } from 'vitest';
import {
  confirmLogout,
  openProfile,
  leaveProfile,
  openProfileEditor,
  leaveProfileEditor,
  parseResearchDestination,
  requireResearchLogin,
  leaveResearchLogin
} from './identity';

const {
  showModal,
  switchTab,
  navigateTo,
  navigateBack,
  reLaunch,
  getCurrentPages,
  showToast,
  redirectTo
} = vi.hoisted(() => ({
  showModal: vi.fn(),
  redirectTo: vi.fn(),
  switchTab: vi.fn(),
  navigateTo: vi.fn(),
  navigateBack: vi.fn(),
  reLaunch: vi.fn(),
  getCurrentPages: vi.fn(),
  showToast: vi.fn()
}));
vi.mock('@tarojs/taro', () => ({
  default: {
    showModal,
    switchTab,
    navigateTo,
    navigateBack,
    reLaunch,
    getCurrentPages,
    showToast,
    redirectTo
  }
}));
it('only resumes allowlisted research routes and cancels to the public home', async () => {
  expect(parseResearchDestination('https://example.com')).toBeUndefined();
  expect(parseResearchDestination('__proto__')).toBeUndefined();
  expect(parseResearchDestination('company')).toBe('company');
  for (const target of ['macro', 'industry', 'company'] as const) {
    await requireResearchLogin(target);
    expect(redirectTo).toHaveBeenLastCalledWith({ url: `/pages/login/index?research=${target}` });
    expect(await leaveResearchLogin(target, true)).toBe(true);
    expect(switchTab).toHaveBeenLastCalledWith({ url: `/pages/${target}/index` });
  }
  await leaveResearchLogin('company', false);
  expect(switchTab).toHaveBeenLastCalledWith({ url: '/pages/index/index' });
  await leaveResearchLogin('companyReport', true);
  expect(redirectTo).toHaveBeenLastCalledWith({ url: '/pages/company/report/index' });
});
it('requires an explicit confirmation and treats cancellation as no logout', async () => {
  showModal.mockResolvedValueOnce({ confirm: false, cancel: true });
  expect(await confirmLogout()).toBe(false);
  showModal.mockResolvedValueOnce({ confirm: true, cancel: false });
  expect(await confirmLogout()).toBe(true);
  expect(showModal).toHaveBeenLastCalledWith(
    expect.objectContaining({ title: '退出登录？', confirmText: '退出登录', cancelText: '取消' })
  );
});

it('opens profile as a tab and returns through the existing stack', async () => {
  await openProfile();
  expect(switchTab).toHaveBeenCalledWith({ url: '/pages/profile/index' });
  getCurrentPages.mockReturnValue([{}, {}]);
  await leaveProfile();
  expect(navigateBack).toHaveBeenCalledWith({ delta: 1 });
  expect(reLaunch).not.toHaveBeenCalled();
});
it('returns directly opened profile to home and reports navigation failure', async () => {
  getCurrentPages.mockReturnValue([{}]);
  await leaveProfile();
  expect(reLaunch).toHaveBeenCalledWith({ url: '/pages/index/index' });
  switchTab.mockRejectedValueOnce(new Error('navigation failed'));
  await openProfile();
  expect(showToast).toHaveBeenCalledWith({ title: '打开失败，请重试', icon: 'none' });
});

it('opens a non-tab editor and returns direct entries to My', async () => {
  await openProfileEditor();
  expect(navigateTo).toHaveBeenCalledWith({ url: '/pages/profile/edit/index' });
  getCurrentPages.mockReturnValue([{}, {}]);
  await leaveProfileEditor();
  expect(navigateBack).toHaveBeenLastCalledWith({ delta: 1 });
  getCurrentPages.mockReturnValue([{}]);
  await leaveProfileEditor();
  expect(switchTab).toHaveBeenLastCalledWith({ url: '/pages/profile/index' });
});

it('retains company identity through login without allowing arbitrary return URLs', async () => {
  await requireResearchLogin('companyReport', '000001.SZ');
  expect(redirectTo).toHaveBeenLastCalledWith({
    url: '/pages/login/index?research=companyReport&symbol=000001.SZ'
  });
  await leaveResearchLogin('companyReport', true, '000001.SZ');
  expect(redirectTo).toHaveBeenLastCalledWith({
    url: '/pages/company/report/index?symbol=000001.SZ'
  });
  await leaveResearchLogin('companyReport', true, '603179.SH');
  expect(redirectTo).toHaveBeenLastCalledWith({
    url: '/pages/company/report/index?symbol=603179.SH'
  });
  await leaveResearchLogin('companyReport', true, 'https://example.com');
  expect(redirectTo).toHaveBeenLastCalledWith({ url: '/pages/company/report/index' });
  await leaveResearchLogin('companyReport', false, '000001.SZ');
  expect(switchTab).toHaveBeenLastCalledWith({ url: '/pages/index/index' });
});

it('preserves Chinese company labels across login without changing the allowlisted symbol', async () => {
  const names = { stockName: '众泰汽车', companyName: '众泰汽车股份有限公司' };
  await requireResearchLogin('companyReport', '000980.SZ', names);
  const loginQuery = new URLSearchParams(redirectTo.mock.calls.at(-1)![0].url.split('?')[1]);
  expect(loginQuery.get('research')).toBe('companyReport');
  expect(loginQuery.get('symbol')).toBe('000980.SZ');
  expect(JSON.parse(loginQuery.get('display')!)).toEqual(names);
  await leaveResearchLogin('companyReport', true, '000980.SZ', names);
  const reportQuery = new URLSearchParams(redirectTo.mock.calls.at(-1)![0].url.split('?')[1]);
  expect(reportQuery.get('symbol')).toBe('000980.SZ');
  expect(JSON.parse(reportQuery.get('display')!)).toEqual(names);
});
