import { readFileSync } from 'node:fs';
import { join } from 'node:path';
import { describe, expect, it } from 'vitest';

describe('Minimal Dashboard conformance', () => {
  it('does not depend on Ant Design in the admin package', () => {
    const packageJSON = readFileSync(join(process.cwd(), 'package.json'), 'utf8');

    expect(packageJSON).not.toContain('"antd"');
    expect(packageJSON).not.toContain('@ant-design');
  });

  it('keeps Minimal Dashboard tokens as the source for admin styling', () => {
    const tokens = readFileSync(join(process.cwd(), 'src/styles/tokens.css'), 'utf8');

    expect(tokens).toContain('--minimal-background');
    expect(tokens).toContain('--brand-900: #18181b');
    expect(tokens).toContain('--text-400: #858585');
    expect(tokens).toContain('--icon-800: #262626');
    expect(tokens).toContain('--font-sans: Geist');
    expect(tokens).toContain('--font-serif: "DM Serif Display"');
    expect(tokens).toContain('--font-mono: "Geist Mono"');
    expect(tokens).toContain('--space-4: var(--spacing)');
    expect(tokens).toContain('--radius-md: 25.2px');
    expect(tokens).toContain('--shadow-sm: 0px 0px 0px 0px rgba(0, 0, 0, 0)');
  });

  it('uses Minimal Dashboard sidebar anatomy and icon menu affordances', () => {
    const shell = readFileSync(join(process.cwd(), 'src/layouts/AdminShell.tsx'), 'utf8');

    expect(shell).toContain('admin-brand-mark');
    expect(shell).toContain('admin-nav-icon');
    expect(shell).toContain('数据采集中心');
    expect(shell).toContain('SYSTEM NOTE');
  });

  it('defines table, tabs, pagination, and status pill primitives in the app stylesheet', () => {
    const stylesheet = readFileSync(join(process.cwd(), 'src/styles/app.css'), 'utf8');

    expect(stylesheet).toContain('.ui-tabs');
    expect(stylesheet).toContain('.ui-data-table');
    expect(stylesheet).toContain('.ui-pagination');
    expect(stylesheet).toContain('.ui-status::before');
  });

  it('uses full-width content surfaces without retired scheduler-specific styling', () => {
    const stylesheet = readFileSync(join(process.cwd(), 'src/styles/app.css'), 'utf8');

    expect(stylesheet).toContain('max-width: none;');
    expect(stylesheet).not.toContain('.scheduler-settings');
    expect(stylesheet).not.toContain('.scheduler-form-grid');
    expect(stylesheet).not.toContain('.scheduler-grid');
    expect(stylesheet).not.toContain('.scheduler-form-card');
    expect(stylesheet).not.toContain('grid-template-columns: minmax(0, 1fr) minmax(340px, 0.72fr);');
  });

  it('uses a fixed shell with header, footer, sidebar, and bounded workspace', () => {
    const shell = readFileSync(join(process.cwd(), 'src/layouts/AdminShell.tsx'), 'utf8');
    const stylesheet = readFileSync(join(process.cwd(), 'src/styles/app.css'), 'utf8');

    expect(shell).toContain('admin-footer');
    expect(stylesheet).toContain('background: var(--background-50);');
    expect(stylesheet).toContain('height: 100vh;');
    expect(stylesheet).toContain('overflow: hidden;');
    expect(stylesheet).toContain('grid-template-rows: auto minmax(0, 1fr) auto;');
    expect(stylesheet).toContain('.admin-content {\n  min-height: 0;\n  overflow: hidden;');
  });

  it('keeps data ingestion tabs outside the only scrolling content area', () => {
    const page = readFileSync(join(process.cwd(), 'src/pages/DataIngestionCenter.tsx'), 'utf8');
    const stylesheet = readFileSync(join(process.cwd(), 'src/styles/app.css'), 'utf8');

    expect(page).toContain('data-ingestion-tabs-bar');
    expect(page).toContain('data-ingestion-scroll-area');
    expect(stylesheet).toContain('grid-template-rows: auto minmax(0, 1fr);');
    expect(stylesheet).toContain('.data-ingestion-tabs-bar');
    expect(stylesheet).toContain('z-index: 10;');
    expect(stylesheet).toContain('.data-ingestion-scroll-area');
    expect(stylesheet).toContain('overflow-y: auto;');
  });
});
