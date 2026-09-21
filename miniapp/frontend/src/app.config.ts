export default defineAppConfig({
  pages: ['pages/index/index', 'pages/report/detail/index', 'pages/profile/index'],
  ...(process.env.TARO_ENV === 'weapp' ? { lazyCodeLoading: 'requiredComponents' as const } : {}),
  tabBar: {
    color: '#85857e',
    selectedColor: '#142d40',
    backgroundColor: '#fffdf8',
    borderStyle: 'white',
    list: [
      {
        pagePath: 'pages/index/index',
        text: '推理',
        iconPath: 'assets/tab/reasoning-normal.png',
        selectedIconPath: 'assets/tab/reasoning-active.png'
      },
      {
        pagePath: 'pages/profile/index',
        text: '我的',
        iconPath: 'assets/tab/profile-normal.png',
        selectedIconPath: 'assets/tab/profile-active.png'
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
