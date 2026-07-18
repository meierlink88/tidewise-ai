import assert from 'node:assert/strict'
import { mkdtemp, mkdir, readFile, writeFile } from 'node:fs/promises'
import { tmpdir } from 'node:os'
import { join } from 'node:path'
import test from 'node:test'
import { getDefaultPreviewDir, publishWeapp } from './publish-weapp.mjs'

test('defaults preview publishing under the user Documents directory', () => {
  assert.equal(getDefaultPreviewDir('/Users/example'), '/Users/example/Documents/WeChatProjects/tidewise-ai-preview')
})

test('replaces stale target contents and writes build provenance', async () => {
  const root = await mkdtemp(join(tmpdir(), 'tidewise-publish-'))
  const sourceDir = join(root, 'source')
  const targetDir = join(root, 'preview')
  await mkdir(sourceDir)
  await mkdir(targetDir)
  await writeFile(join(sourceDir, 'app.json'), '{"pages":["pages/index/index"]}')
  await writeFile(join(sourceDir, 'fresh.js'), 'fresh')
  await writeFile(join(targetDir, 'stale.js'), 'stale')
  await writeFile(join(root, 'outside.txt'), 'preserve')

  await publishWeapp({
    sourceDir,
    targetDir,
    provenance: { branch: 'codex/test', commit: 'abc123', builtAt: '2026-07-12T00:00:00.000Z', buildTarget: 'weapp' }
  })

  await assert.rejects(readFile(join(targetDir, 'stale.js')))
  assert.equal(await readFile(join(targetDir, 'fresh.js'), 'utf8'), 'fresh')
  assert.equal(await readFile(join(root, 'outside.txt'), 'utf8'), 'preserve')
  const marker = JSON.parse(await readFile(join(targetDir, 'tidewise-build.json'), 'utf8'))
  assert.equal(marker.branch, 'codex/test')
  assert.equal(marker.commit, 'abc123')
  assert.equal(marker.buildTarget, 'weapp')
  assert.match(marker.sourceAppJsonSha256, /^[a-f0-9]{64}$/)
})

test('rejects unsafe targets before touching files', async () => {
  const root = await mkdtemp(join(tmpdir(), 'tidewise-publish-boundary-'))
  const sourceDir = join(root, 'source')
  await mkdir(sourceDir)
  await writeFile(join(sourceDir, 'app.json'), '{}')

  await assert.rejects(
    publishWeapp({ sourceDir, targetDir: '/', provenance: { branch: 'x', commit: 'y', builtAt: 'z', buildTarget: 'weapp' } }),
    /unsafe preview target/
  )
  assert.equal(await readFile(join(sourceDir, 'app.json'), 'utf8'), '{}')
})
