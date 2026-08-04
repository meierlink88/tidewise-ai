import { access, readFile, stat } from 'node:fs/promises';
import { resolve } from 'node:path';

const root = resolve(import.meta.dirname, '..');
const outputRoot = resolve(root, 'dist/weapp');
const appConfig = JSON.parse(await readFile(resolve(outputRoot, 'app.json'), 'utf8'));
const projectConfig = JSON.parse(
  await readFile(resolve(outputRoot, 'project.config.json'), 'utf8')
);
const stylesheet = resolve(outputRoot, 'common.wxss');
const reasoningTreeStylesheet = resolve(
  outputRoot,
  'pages/research-theme/reasoning-trees/index.wxss'
);
const avatar = resolve(outputRoot, 'assets/nav-avatar.png');
const legacySeaImage = resolve(outputRoot, 'assets/home-header-sea.jpg');

if (
  JSON.stringify(appConfig.pages) !==
  JSON.stringify([
    'pages/index/index',
    'pages/research-theme/history/index',
    'pages/research-theme/reasoning-trees/index'
  ])
) {
  throw new Error('微信构建必须注册今日主题、历史主题与推理树页');
}
if ('tabBar' in appConfig) throw new Error('微信首页 shell 不得包含 tabBar');
if (appConfig.window?.navigationStyle !== 'custom')
  throw new Error('微信首页必须使用自定义导航以适配原生状态栏和胶囊');
if (projectConfig.miniprogramRoot !== './')
  throw new Error('构建产物必须能作为微信小程序根目录直接导入');
if (projectConfig.compileType !== 'miniprogram') throw new Error('微信项目类型必须为 miniprogram');
if (typeof projectConfig.appid !== 'string' || projectConfig.appid.length === 0)
  throw new Error('微信项目必须声明 appid');

const stylesheetSize = (await stat(stylesheet)).size;
if (stylesheetSize >= 64 * 1024) throw new Error(`首页 WXSS 体积过大: ${stylesheetSize} bytes`);

const reasoningTreeStyles = await readFile(reasoningTreeStylesheet, 'utf8');
const reasoningTreeStylesWithoutComments = reasoningTreeStyles.replace(/\/\*[\s\S]*?\*\//g, '');
const reasoningTreeSelectors = [...reasoningTreeStylesWithoutComments.matchAll(/([^{}]+)\{/g)].map(
  ([, selector]) => selector
);
if (
  reasoningTreeSelectors.some((selector) => /(^|[\s,>+~])\*(?=$|[\s,.#:[\]>+~])/.test(selector))
) {
  throw new Error('推理树 WXSS 不得使用微信编译器不支持的通配选择器');
}
if (!/\.reasoning-theme-hero__title\s*\{[^}]*font-size:\s*19px;/s.test(reasoningTreeStyles)) {
  throw new Error('推理树标题必须按定稿原型保留 19px，不得缩放为 rpx');
}
if (!/\.reasoning-chain-node\s*\{[^}]*width:\s*158px;/s.test(reasoningTreeStyles)) {
  throw new Error('推理树节点必须按定稿原型保留 158px，不得缩放为 rpx');
}

const avatarSize = (await stat(avatar)).size;
if (avatarSize >= 128 * 1024) throw new Error(`导航头像体积过大: ${avatarSize} bytes`);

try {
  await access(legacySeaImage);
  throw new Error('新版首页构建产物不得包含旧 home-header-sea.jpg');
} catch (error) {
  if (error instanceof Error && error.message.includes('不得包含')) throw error;
  if (error && typeof error === 'object' && 'code' in error && error.code !== 'ENOENT') throw error;
}

process.stdout.write(
  `weapp output verified: project=${projectConfig.projectname}, wxss=${stylesheetSize} bytes, avatar=${avatarSize} bytes\n`
);
