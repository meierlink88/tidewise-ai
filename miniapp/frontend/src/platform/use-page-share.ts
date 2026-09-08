import Taro, { useShareAppMessage, useShareTimeline } from '@tarojs/taro';
import { useMemo } from 'react';

interface PageShare {
  title: string;
  path: string;
  query: string;
}

/** 微信朋友圈单页不能使用跨页导航或导航栏 API。 */
export function usePageShare(share: PageShare): boolean {
  const isSinglePage = useMemo(
    () => process.env.TARO_ENV === 'weapp' && Taro.getLaunchOptionsSync().scene === 1154,
    []
  );
  useShareAppMessage(() => ({ title: share.title, path: share.path }));
  useShareTimeline(() => ({ title: share.title, query: share.query }));
  return isSinglePage;
}
