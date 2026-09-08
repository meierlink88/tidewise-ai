export default definePageConfig({
  ...(process.env.TARO_ENV === 'weapp'
    ? {
        enableShareAppMessage: true,
        enableShareTimeline: true,
        singlePage: { navigationBarFit: 'squeezed' }
      }
    : {}),
  navigationBarTitleText: '观潮家',
  navigationBarBackgroundColor: '#071735',
  navigationBarTextStyle: 'white',
  navigationStyle: 'custom',
  enablePullDownRefresh: true,
  backgroundColor: '#f8fafc',
  backgroundTextStyle: 'dark'
});
