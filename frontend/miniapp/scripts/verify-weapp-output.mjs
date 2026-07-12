import { createHash } from 'node:crypto'
import { readFile, stat } from 'node:fs/promises'
import { resolve } from 'node:path'

const root = resolve(import.meta.dirname, '..')
const appConfig = JSON.parse(await readFile(resolve(root, 'dist/app.json'), 'utf8'))
const stylesheet = resolve(root, 'dist/pages/index/index.wxss')
const image = resolve(root, 'dist/assets/home-header-sea.jpg')
const expectedImageHash = '667dcd64bcfb7c3d40e4f5f5a6d0b9be1f88a90824e5e3db88527f08703b6fdc'

if (appConfig.pages[0] !== 'pages/index/index') throw new Error('今日观潮不是默认页面')
if (appConfig.tabBar.list[0]?.pagePath !== 'pages/index/index') throw new Error('今日观潮不是首个 tab')
if (appConfig.tabBar.list[0]?.text !== '首页') throw new Error('首个 tab 文案不是首页')
if (appConfig.tabBar.selectedColor !== '#2563eb') throw new Error('选中态不是 canonical 蓝色')

const stylesheetSize = (await stat(stylesheet)).size
if (stylesheetSize >= 64 * 1024) throw new Error(`首页 WXSS 体积过大: ${stylesheetSize} bytes`)

const imageBuffer = await readFile(image)
const imageHash = createHash('sha256').update(imageBuffer).digest('hex')
if (imageHash !== expectedImageHash) throw new Error('构建产物海面图指纹不匹配')

process.stdout.write(`weapp output verified: wxss=${stylesheetSize} bytes, image=${imageHash}\n`)
