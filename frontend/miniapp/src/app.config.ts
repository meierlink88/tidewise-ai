export default defineAppConfig({
  pages: [
    'pages/feed/index',
    'pages/index/index',
    'pages/ai/index',
    'pages/sectors/index',
    'pages/subscribe/index'
  ],
  window: {
    backgroundTextStyle: 'light',
    navigationBarBackgroundColor: '#0f172a',
    navigationBarTitleText: '观潮家',
    navigationBarTextStyle: 'white'
  },
  tabBar: {
    color: '#64748b',
    selectedColor: '#0f766e',
    backgroundColor: '#ffffff',
    borderStyle: 'black',
    list: [
      {
        pagePath: 'pages/feed/index',
        text: '行情'
      },
      {
        pagePath: 'pages/index/index',
        text: '指数'
      },
      {
        pagePath: 'pages/ai/index',
        text: 'AI 助手'
      },
      {
        pagePath: 'pages/sectors/index',
        text: '板块'
      },
      {
        pagePath: 'pages/subscribe/index',
        text: '订阅'
      }
    ]
  }
});
