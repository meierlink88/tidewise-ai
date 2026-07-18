import { access, readFile, stat } from 'node:fs/promises'
import { resolve } from 'node:path'

const root = resolve(import.meta.dirname, '..')
const appConfig = JSON.parse(await readFile(resolve(root, 'dist/app.json'), 'utf8'))
const stylesheet = resolve(root, 'dist/pages/index/index.wxss')
const avatar = resolve(root, 'dist/assets/nav-avatar.png')
const legacySeaImage = resolve(root, 'dist/assets/home-header-sea.jpg')

if (JSON.stringify(appConfig.pages) !== JSON.stringify(['pages/index/index'])) throw new Error('微信首页必须只注册新版观潮首页')
if ('tabBar' in appConfig) throw new Error('微信首页 shell 不得包含 tabBar')
if (appConfig.window?.navigationStyle !== 'custom') throw new Error('微信首页必须使用自定义导航以适配原生状态栏和胶囊')

const stylesheetSize = (await stat(stylesheet)).size
if (stylesheetSize >= 64 * 1024) throw new Error(`首页 WXSS 体积过大: ${stylesheetSize} bytes`)

const avatarSize = (await stat(avatar)).size
if (avatarSize >= 128 * 1024) throw new Error(`导航头像体积过大: ${avatarSize} bytes`)

try {
  await access(legacySeaImage)
  throw new Error('新版首页构建产物不得包含旧 home-header-sea.jpg')
} catch (error) {
  if (error instanceof Error && error.message.includes('不得包含')) throw error
  if (error && typeof error === 'object' && 'code' in error && error.code !== 'ENOENT') throw error
}

process.stdout.write(`weapp output verified: wxss=${stylesheetSize} bytes, avatar=${avatarSize} bytes\n`)
