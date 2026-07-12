export const appPages = [
  'pages/index/index',
  'pages/feed/index',
  'pages/ai/index',
  'pages/sectors/index',
  'pages/subscribe/index'
] as const;

export const appTabBar = {
  color: '#64748b',
  selectedColor: '#2563eb',
  backgroundColor: '#ffffff',
  borderStyle: 'black' as const,
  list: [
    { pagePath: 'pages/index/index', text: '首页' },
    { pagePath: 'pages/feed/index', text: '行情' },
    { pagePath: 'pages/ai/index', text: 'AI 助手' },
    { pagePath: 'pages/sectors/index', text: '板块' },
    { pagePath: 'pages/subscribe/index', text: '订阅' }
  ]
};
