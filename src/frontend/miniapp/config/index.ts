import { defineConfig } from '@tarojs/cli';
import path from 'node:path';

const buildTarget = process.env.TARO_ENV ?? 'weapp';
const researchSource = process.env.TARO_APP_RESEARCH_SOURCE;
const miniappApiBaseUrl = process.env.TARO_APP_MINIAPP_API_BASE_URL ?? '';

if (researchSource !== 'mock' && researchSource !== 'api') {
  throw new Error('TARO_APP_RESEARCH_SOURCE must explicitly be mock or api');
}

const config = defineConfig({
  projectName: 'tidewise-miniapp',
  date: '2026-07-05',
  designWidth: 750,
  deviceRatio: {
    640: 2.34,
    750: 1,
    828: 1.81
  },
  sourceRoot: 'src',
  outputRoot: `dist/${buildTarget}`,
  alias: {
    '@': path.resolve(__dirname, '..', 'src')
  },
  env: {
    TARO_APP_RESEARCH_SOURCE: JSON.stringify(researchSource),
    TARO_APP_MINIAPP_API_BASE_URL: JSON.stringify(miniappApiBaseUrl)
  },
  plugins: ['@tarojs/plugin-platform-weapp', '@tarojs/plugin-platform-tt'],
  framework: 'react',
  compiler: 'webpack5',
  mini: {
    postcss: {
      pxtransform: {
        enable: true,
        config: {}
      },
      cssModules: {
        enable: false,
        config: {
          namingPattern: 'module',
          generateScopedName: '[name]__[local]___[hash:base64:5]'
        }
      }
    }
  },
  h5: {}
});

export default config;
