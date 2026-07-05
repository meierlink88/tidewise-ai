## 1. Project Configuration

- [ ] 1.1 Create `package.json` with Mini Program TypeScript development scripts and dev dependencies.
- [ ] 1.2 Create `tsconfig.json` for Mini Program TypeScript compilation.
- [ ] 1.3 Create lint and formatting configuration files for the source engineering root.
- [ ] 1.4 Create `apps/miniprogram/project.config.json` and `apps/miniprogram/sitemap.json` with safe development defaults.

## 2. App Shell

- [ ] 2.1 Create `apps/miniprogram/app.ts`, `apps/miniprogram/app.json`, and `apps/miniprogram/app.wxss`.
- [ ] 2.2 Configure five tabBar entries for feed, index, AI assistant, sectors, and subscribe.
- [ ] 2.3 Add global style imports and base design variables for the Mini Program shell.

## 3. Page Skeletons

- [ ] 3.1 Create feed page files with a minimal page title and placeholder content.
- [ ] 3.2 Create index page files with a minimal page title and placeholder content.
- [ ] 3.3 Create AI assistant page files with a minimal page title and safety disclaimer placeholder.
- [ ] 3.4 Create sectors page files with a minimal page title and placeholder content.
- [ ] 3.5 Create subscribe page files with a minimal page title and placeholder content.

## 4. Shared Source Directories

- [ ] 4.1 Create reusable component directories for market card, event card, sector card, insight panel, confidence bar, tag list, and empty state.
- [ ] 4.2 Create domain model files for event, market, sector, graph, report, subscription, and AI message types.
- [ ] 4.3 Create utility, constants, store, styles, and assets directories with minimal placeholder modules where needed.

## 5. Mock Data and Service Boundary

- [ ] 5.1 Create mock data modules for events, markets, sectors, graph, and subscriptions.
- [ ] 5.2 Create request wrapper and service modules for event, market, sector, AI, report, and subscription domains.
- [ ] 5.3 Ensure page skeletons can rely on service boundaries rather than importing prototype files or browser-only logic.

## 6. Validation

- [ ] 6.1 Verify generated files do not modify `prototype` or `doc` directories.
- [ ] 6.2 Verify Mini Program configuration references all tab pages and existing page files.
- [ ] 6.3 Run available TypeScript, lint, or configuration validation commands and record any unavailable tooling.
- [ ] 6.4 Confirm no secrets, model credentials, payment credentials, direct database access, or backend execution logic are present in the Mini Program source.
