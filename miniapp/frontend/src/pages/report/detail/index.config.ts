export default definePageConfig({
  ...(process.env.TARO_ENV === 'weapp'
    ? {
        enableShareAppMessage: true,
        enableShareTimeline: true,
        singlePage: { navigationBarFit: 'squeezed' }
      }
    : {}),
  navigationBarTitleText: '深度分析',
  navigationBarBackgroundColor: '#0b2035',
  navigationBarTextStyle: 'white',
  navigationStyle: 'custom',
  enablePullDownRefresh: true,
  backgroundColor: '#f7f5ef',
  backgroundTextStyle: 'dark'
});
