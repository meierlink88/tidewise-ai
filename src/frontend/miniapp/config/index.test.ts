import { readFile } from 'node:fs/promises'
import { resolve } from 'node:path'
import { afterEach, describe, expect, it, vi } from 'vitest'

async function loadConfig(target?: string, source: 'mock' | 'api' | null = 'mock', baseUrl = '') {
  vi.resetModules()
  vi.unstubAllEnvs()
  if (target) vi.stubEnv('TARO_ENV', target)
  if (source !== null) vi.stubEnv('TARO_APP_RESEARCH_SOURCE', source)
  if (baseUrl !== '') vi.stubEnv('TARO_APP_MINIAPP_API_BASE_URL', baseUrl)

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

  it('injects an explicit homepage data source and public Miniapp BFF URL', async () => {
    const config = await loadConfig('weapp', 'api', 'https://miniapp.example.test')

    expect(config.env).toMatchObject({
      TARO_APP_RESEARCH_SOURCE: JSON.stringify('api'),
      TARO_APP_MINIAPP_API_BASE_URL: JSON.stringify('https://miniapp.example.test')
    })
  })

  it('injects the frontend-owned mock adapter when explicitly selected', async () => {
    const config = await loadConfig('weapp', 'mock')

    expect(config.env).toMatchObject({
      TARO_APP_RESEARCH_SOURCE: JSON.stringify('mock'),
      TARO_APP_MINIAPP_API_BASE_URL: JSON.stringify('')
    })
  })

  it('rejects a build without an explicit homepage data source', async () => {
    await expect(loadConfig('weapp', null)).rejects.toThrow(
      'TARO_APP_RESEARCH_SOURCE must explicitly be mock or api'
    )
  })
})
