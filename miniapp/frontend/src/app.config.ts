export default defineAppConfig({
  pages: [
    'pages/index/index',
    'pages/macro/index',
    'pages/industry/index',
    'pages/company/index',
    'pages/company/report/index',
    'pages/report/detail/index',
    'pages/profile/index',
    'pages/profile/information/index',
    'pages/profile/edit/index',
    'pages/login/index',
    'pages/tracking/index'
  ],
  ...(process.env.TARO_ENV === 'weapp' ? { lazyCodeLoading: 'requiredComponents' as const } : {}),
  tabBar: {
    color: '#777e91',
    selectedColor: '#194d87',
    backgroundColor: '#ffffff',
    borderStyle: 'white',
    list: [
      {
        pagePath: 'pages/index/index',
        text: '要闻解读',
        iconPath: 'assets/tab/reasoning-normal.png',
        selectedIconPath: 'assets/tab/reasoning-active.png'
      },
      {
        pagePath: 'pages/macro/index',
        text: '宏观指数',
        iconPath: 'assets/tab/macro-normal.png',
        selectedIconPath: 'assets/tab/macro-active.png'
      },
      {
        pagePath: 'pages/industry/index',
        text: '产业透析',
        iconPath: 'assets/tab/industry-normal.png',
        selectedIconPath: 'assets/tab/industry-active.png'
      },
      {
        pagePath: 'pages/company/index',
        text: '企业洞察',
        iconPath: 'assets/tab/company-normal.png',
        selectedIconPath: 'assets/tab/company-active.png'
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
