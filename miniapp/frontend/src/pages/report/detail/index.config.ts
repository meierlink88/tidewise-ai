export default definePageConfig({
  ...(process.env.TARO_ENV === 'weapp'
    ? {
        enableShareAppMessage: true,
        enableShareTimeline: true,
        singlePage: { navigationBarFit: 'squeezed' }
      }
    : {}),
  navigationBarTitleText: '深度分析',
  navigationBarBackgroundColor: '#ffffff',
  navigationBarTextStyle: 'black',
  navigationStyle: 'custom',
  enablePullDownRefresh: false,
  backgroundColor: '#f7f5ef',
  backgroundTextStyle: 'dark'
});
