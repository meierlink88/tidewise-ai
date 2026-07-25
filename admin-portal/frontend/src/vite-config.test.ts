// @vitest-environment node

import { describe, expect, it } from 'vitest';
import viteConfig from '../vite.config';

describe('vite dev proxy', () => {
  it('proxies admin api requests to the local admin api server', () => {
    expect(viteConfig.server?.proxy?.['/admin']).toMatchObject({
      target: 'http://127.0.0.1:9013',
      changeOrigin: true
    });
  });
});
