export default definePageConfig({
  ...(process.env.TARO_ENV === 'weapp'
    ? {
        enableShareAppMessage: true,
        enableShareTimeline: true,
        singlePage: { navigationBarFit: 'squeezed' }
      }
    : {}),
  navigationBarTitleText: '推理详情',
  navigationBarBackgroundColor: '#071735',
  navigationBarTextStyle: 'white',
  navigationStyle: 'default',
  enablePullDownRefresh: true,
  backgroundColor: '#f8fafc',
  backgroundTextStyle: 'dark'
});
