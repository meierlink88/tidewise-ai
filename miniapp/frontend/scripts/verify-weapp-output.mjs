import { access, readFile, stat } from 'node:fs/promises';
import { resolve } from 'node:path';

const platform = process.argv[2] ?? 'weapp';
if (platform !== 'weapp' && platform !== 'tt') {
  throw new Error(`unsupported Miniapp output platform: ${platform}`);
}

const root = resolve(import.meta.dirname, '..');
const outputRoot = resolve(root, `dist/${platform}`);
const appConfig = JSON.parse(await readFile(resolve(outputRoot, 'app.json'), 'utf8'));
const stylesheet = resolve(outputRoot, platform === 'weapp' ? 'app.wxss' : 'app.ttss');
const avatar = resolve(outputRoot, 'assets/nav-avatar.png');

const expectedPages = ['pages/index/index', 'pages/report/detail/index'];
if (JSON.stringify(appConfig.pages) !== JSON.stringify(expectedPages)) {
  throw new Error(`${platform} 构建必须只注册首页和推理详情页`);
}
if (appConfig.pages.some((page) => page.includes('research-theme'))) {
  throw new Error(`${platform} 构建不得包含已退役页面`);
}
if ('tabBar' in appConfig) throw new Error(`${platform} 首页 shell 不得包含 tabBar`);
if (appConfig.window?.navigationStyle !== 'custom') {
  throw new Error(`${platform} 首页必须使用自定义导航以适配原生状态栏`);
}

if (platform === 'weapp') {
  if (appConfig.lazyCodeLoading !== 'requiredComponents') {
    throw new Error('微信构建必须启用组件按需注入 lazyCodeLoading=requiredComponents');
  }

  const projectConfig = JSON.parse(
    await readFile(resolve(outputRoot, 'project.config.json'), 'utf8')
  );
  if (projectConfig.miniprogramRoot !== './') {
    throw new Error('构建产物必须能作为微信小程序根目录直接导入');
  }
  if (projectConfig.compileType !== 'miniprogram') {
    throw new Error('微信项目类型必须为 miniprogram');
  }
  if (typeof projectConfig.appid !== 'string' || projectConfig.appid.length === 0) {
    throw new Error('微信项目必须声明 appid');
  }
} else if ('lazyCodeLoading' in appConfig) {
  throw new Error('抖音构建不得包含微信专用的 lazyCodeLoading 配置');
}

const stylesheetSize = (await stat(stylesheet)).size;
if (stylesheetSize >= 64 * 1024) {
  throw new Error(`${platform} 公共样式体积过大: ${stylesheetSize} bytes`);
}

const avatarSize = (await stat(avatar)).size;
if (avatarSize >= 128 * 1024) {
  throw new Error(`导航头像体积过大: ${avatarSize} bytes`);
}

await assertMissing(resolve(outputRoot, 'pages/research-theme'), `${platform} 旧页面目录`);
await assertPresent(resolve(outputRoot, 'pages/report/detail/index.js'), `${platform} 推理详情页`);
await assertMissing(resolve(outputRoot, 'pages/report/evidences'), `${platform} 旧证据路由目录`);
await assertMissing(resolve(outputRoot, 'assets/home-header-sea.jpg'), '旧首页海面图片');
await assertMissing(resolve(outputRoot, 'assets/icons/theme-history.svg'), '旧历史入口图标');
await assertMissing(resolve(outputRoot, 'assets/icons/today-theme.svg'), '旧今日入口图标');

process.stdout.write(
  `${platform} output verified: pages=${appConfig.pages.length}, styles=${stylesheetSize} bytes, avatar=${avatarSize} bytes\n`
);

async function assertMissing(path, label) {
  try {
    await access(path);
    throw new Error(`${label}不得出现在构建产物中`);
  } catch (error) {
    if (error instanceof Error && error.message.includes('不得出现在')) throw error;
    if (error && typeof error === 'object' && 'code' in error && error.code !== 'ENOENT') {
      throw error;
    }
  }
}

async function assertPresent(path, label) {
  try {
    await access(path);
  } catch {
    throw new Error(`${label}必须出现在构建产物中`);
  }
}
