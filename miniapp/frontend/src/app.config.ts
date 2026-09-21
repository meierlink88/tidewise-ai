export default defineAppConfig({
  pages: [
    'pages/index/index',
    'pages/report/detail/index',
    'pages/profile/index',
    'pages/profile/information/index',
    'pages/login/index',
    'pages/tracking/index'
  ],
  ...(process.env.TARO_ENV === 'weapp' ? { lazyCodeLoading: 'requiredComponents' as const } : {}),
  tabBar: {
    color: '#858780',
    selectedColor: '#123343',
    backgroundColor: '#fffefa',
    borderStyle: 'white',
    list: [
      {
        pagePath: 'pages/index/index',
        text: '推理',
        iconPath: 'assets/tab/reasoning-normal.png',
        selectedIconPath: 'assets/tab/reasoning-active.png'
      },
      {
        pagePath: 'pages/tracking/index',
        text: '跟踪',
        iconPath: 'assets/tab/tracking-normal.png',
        selectedIconPath: 'assets/tab/tracking-active.png'
      }
    ]
  },
  window: {
    backgroundTextStyle: 'light',
    navigationBarBackgroundColor: '#071735',
    navigationBarTitleText: '观潮家',
    navigationBarTextStyle: 'white',
    navigationStyle: 'custom',
    backgroundColor: '#f8fafc'
  }
});
