import { access, readdir } from 'node:fs/promises';
import { resolve, relative } from 'node:path';
import { spawnSync } from 'node:child_process';

// Taro compilation does not execute WeChat's WXSS parser. Run the installed
// developer-tool compiler against all emitted styles, including imported files.
const compiler =
  process.env.WXSS_COMPILER ||
  '/Applications/wechatwebdevtools.app/Contents/Resources/app.asar.unpacked/node_modules/wcc-exec/wcsc';
try {
  await access(compiler);
} catch {
  throw new Error(
    'WeChat stylesheet compiler missing. Set WXSS_COMPILER to the installed wcsc executable.'
  );
}
const root = resolve(import.meta.dirname, '../dist/weapp');
const files = [];
async function collect(directory) {
  for (const entry of await readdir(directory, { withFileTypes: true })) {
    const path = resolve(directory, entry.name);
    if (entry.isDirectory()) await collect(path);
    else if (entry.isFile() && entry.name.endsWith('.wxss')) files.push(relative(root, path));
  }
}
await collect(root);
if (!files.length) throw new Error('No WXSS output found. Build weapp first.');
const result = spawnSync(compiler, files.sort(), {
  cwd: root,
  encoding: 'utf8',
  stdio: ['ignore', 'ignore', 'pipe'],
  timeout: 30000
});
if (result.error) throw result.error;
if (result.status !== 0) throw new Error(`WeChat WXSS compilation failed:\n${result.stderr}`);
console.log(`WeChat WXSS compiler verified ${files.length} stylesheets.`);
