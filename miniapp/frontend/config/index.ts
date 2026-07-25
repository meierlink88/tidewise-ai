import { defineConfig } from '@tarojs/cli';
import path from 'node:path';

const buildTarget = process.env.TARO_ENV ?? 'weapp';
const researchSource = process.env.TARO_APP_RESEARCH_SOURCE;
const miniappApiBaseUrl = process.env.TARO_APP_MINIAPP_API_BASE_URL ?? '';
const researchWindowHours = process.env.TARO_APP_RESEARCH_WINDOW_HOURS ?? '24';
const h5ApiProxyTarget = process.env.TARO_APP_H5_API_PROXY_TARGET ?? 'http://127.0.0.1:9012';

if (researchSource !== 'mock' && researchSource !== 'api') {
  throw new Error('TARO_APP_RESEARCH_SOURCE must explicitly be mock or api');
}
if (!/^\d+$/.test(researchWindowHours) || Number(researchWindowHours) < 1 || Number(researchWindowHours) > 168) {
  throw new Error('TARO_APP_RESEARCH_WINDOW_HOURS must be an integer between 1 and 168');
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
    TARO_APP_MINIAPP_API_BASE_URL: JSON.stringify(miniappApiBaseUrl),
    TARO_APP_RESEARCH_WINDOW_HOURS: JSON.stringify(researchWindowHours)
  },
  plugins: [
    '@tarojs/plugin-platform-weapp',
    '@tarojs/plugin-platform-tt',
    '@tarojs/plugin-platform-h5'
  ],
  framework: 'react',
  compiler: {
    type: 'webpack5',
    prebundle: {
      enable: false
    }
  },
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
  h5: {
    webpack: {
      watchOptions: {
        ignored: /node_modules/,
        poll: 1000
      }
    },
    devServer: {
      port: 10086,
      proxy: {
        '/api': {
          target: h5ApiProxyTarget,
          changeOrigin: true
        }
      }
    }
  }
});

export default config;
