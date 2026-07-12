import { createHash } from 'node:crypto'
import { execFileSync } from 'node:child_process'
import { cp, mkdir, readFile, rename, rm, stat, writeFile } from 'node:fs/promises'
import { homedir } from 'node:os'
import { basename, dirname, isAbsolute, join, parse, resolve } from 'node:path'
import { fileURLToPath } from 'node:url'

const scriptDir = dirname(fileURLToPath(import.meta.url))
const miniappRoot = resolve(scriptDir, '..')

export function getDefaultPreviewDir(home = homedir()) {
  return join(home, 'Documents/WeChatProjects/tidewise-ai-preview')
}

function assertSafeTarget(targetDir, sourceDir) {
  const target = resolve(targetDir)
  const source = resolve(sourceDir)
  if (!isAbsolute(targetDir) || target === parse(target).root || target === resolve(homedir()) || target === source) {
    throw new Error(`unsafe preview target: ${target}`)
  }
}

async function exists(path) {
  try {
    await stat(path)
    return true
  } catch (error) {
    if (error?.code === 'ENOENT') return false
    throw error
  }
}

export async function publishWeapp({ sourceDir, targetDir, provenance }) {
  assertSafeTarget(targetDir, sourceDir)
  const source = resolve(sourceDir)
  const target = resolve(targetDir)
  const appJson = await readFile(join(source, 'app.json'))
  const sourceAppJsonSha256 = createHash('sha256').update(appJson).digest('hex')
  const parent = dirname(target)
  const nonce = `${process.pid}-${Date.now()}`
  const staging = join(parent, `.${basename(target)}.staging-${nonce}`)
  const backup = join(parent, `.${basename(target)}.backup-${nonce}`)

  await mkdir(parent, { recursive: true })
  await rm(staging, { recursive: true, force: true })
  await cp(source, staging, { recursive: true })
  await writeFile(join(staging, 'tidewise-build.json'), `${JSON.stringify({ ...provenance, sourceAppJsonSha256 }, null, 2)}\n`)

  let movedExisting = false
  try {
    if (await exists(target)) {
      await rename(target, backup)
      movedExisting = true
    }
    await rename(staging, target)
    if (movedExisting) await rm(backup, { recursive: true, force: true })
  } catch (error) {
    await rm(staging, { recursive: true, force: true })
    if (movedExisting && !(await exists(target))) await rename(backup, target)
    throw error
  }

  return { targetDir: target, marker: { ...provenance, sourceAppJsonSha256 } }
}

function gitValue(args) {
  return execFileSync('git', args, { cwd: resolve(miniappRoot, '../..'), encoding: 'utf8' }).trim()
}

async function main() {
  const sourceDir = resolve(miniappRoot, 'dist')
  const targetDir = resolve(process.env.TIDEWISE_WEAPP_PREVIEW_DIR || getDefaultPreviewDir())
  const result = await publishWeapp({
    sourceDir,
    targetDir,
    provenance: {
      branch: gitValue(['branch', '--show-current']),
      commit: gitValue(['rev-parse', 'HEAD']),
      builtAt: new Date().toISOString(),
      buildTarget: 'weapp'
    }
  })
  process.stdout.write(`weapp preview published: ${result.targetDir}\n`)
}

if (resolve(process.argv[1] || '') === fileURLToPath(import.meta.url)) await main()
