import { defineConfig } from '@tarojs/cli';
import path from 'node:path';

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
  outputRoot: 'dist',
  alias: {
    '@': path.resolve(__dirname, '..', 'src')
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
