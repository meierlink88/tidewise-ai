import { readFile } from 'node:fs/promises'
import { resolve } from 'node:path'
import { afterEach, describe, expect, it, vi } from 'vitest'

async function loadConfig(target?: string) {
  vi.resetModules()
  if (target) vi.stubEnv('TARO_ENV', target)
  else vi.unstubAllEnvs()

  const config = (await import('./index')).default
  if (typeof config === 'function') throw new Error('Expected an object-based Taro config')

  return config
}

afterEach(() => {
  vi.unstubAllEnvs()
})

describe('platform output directories', () => {
  it.each(['weapp', 'tt', 'h5'])('isolates %s build output', async (target) => {
    const config = await loadConfig(target)

    expect(config.outputRoot).toBe(`dist/${target}`)
  })

  it('defaults to the WeChat output directory outside a Taro build', async () => {
    const config = await loadConfig()

    expect(config.outputRoot).toBe('dist/weapp')
  })

  it('points the source WeChat project descriptor at the WeChat output', async () => {
    const projectConfig = JSON.parse(await readFile(resolve(import.meta.dirname, '..', 'project.config.json'), 'utf8'))

    expect(projectConfig.miniprogramRoot).toBe('./dist/weapp')
  })
})
