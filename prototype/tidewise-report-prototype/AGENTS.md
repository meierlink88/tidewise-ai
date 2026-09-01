# Mobile Prototype Agent Guide

## Prototype Instructions

In ChatGPT Work Mode, run `sites-preview start "$PWD"`, open `http://terminal.local:4173/` in the cloud browser, and verify the rendered app and its primary interactions. Keep that preview open and tell the user to inspect it in the cloud browser; do not present the local URL as a user-facing chat link. In Codex Desktop, run the local server yourself, open the preview in the in-app browser, and provide the clickable local URL. Do not deploy to Sites unless the user explicitly asks to share, publish, or deploy. Do not give the user server-start instructions when you can run it.

Before planning or implementing any mobile-app change, read this `AGENTS.md` in full. It is the source of truth for the template's runtime and component guidance.

Before making substantial visual changes, use the Product Design plugin's `get-context` skill when the visual source is unclear or no longer matches the current goal. When the user gives durable prototype-specific design feedback, preferences, or decisions, record them in `AGENTS.md`.

## Investment Report Projection Contract

- Treat the four list tabs as projections of the same fixed-structure reasoning report, not as separate reports.
- One report can publish at most one geopolitics card and one macroeconomics card. Each shows only its own one-sentence conclusion; downstream layers are represented by compact related-object and text-status tags.
- A geopolitics card projects `geopolitics anchors -> macro anchors -> related industry chains`; it does not show chain nodes.
- A macroeconomics card projects `macro anchors -> related industry chains`; it does not repeat geopolitics or show chain nodes.
- The industry-chain tab renders one card per published chain. Each card shows its own conclusion and an ordered node preview with node result states.
- Company cards are reserved for a later contract and should remain an explicit empty boundary in this prototype.
- Statuses must use text in addition to color (`升温`, `降温`, `分化`, `待验证`). Cards show only the current layer's prose conclusion; detailed why, evidence IDs, and reasoning steps belong in the detail view.
- The list page keeps the current Miniapp home hierarchy: a single `今日推理` heading followed immediately by the four projection tabs. Do not add a second per-tab list eyebrow or `{layer}报告` section title above the cards.
- The four home tab nouns are fixed as `政治`, `经济`, `产业链`, and `企业`; full report-layer terminology remains `地缘政治`, `宏观经济`, `产业链`, and `公司层` inside report content and detail semantics.
- Every layer or chain exposes its directly related Evidence at the stage that owns it. Card entry copy is simply `证据`: do not show Evidence counts or IDs, and do not label Metric Observation or Counterevidence records as Events. Preserve Evidence IDs only in the in-memory/report contract for traceability.
- Do not show horizon, evidence count, or Evidence Gap as a generic stats bar on geopolitical or macro cards. Evidence Gaps belong in reasoning detail under uncertainty, missing verification, or checkpoints.
- Treat the Markdown report as the only semantic source for the prototype. Every conclusion, anchor/result state, chain, node, and Evidence entry shown in a card must map to an explicit report field or Claim Ledger entry; presentation labels may describe navigation but must not introduce a new research claim.
- Multi-level cards express one ordered projection: current-layer conclusion → current-layer anchors → downstream-layer conclusion and anchors → related chains. Industry cards express chain conclusion → chain anchor → affected nodes. Do not flatten these into parallel summary blocks.
- Card typography follows semantic priority: conclusions first, anchor or node names second, result chips third, and Evidence entry text last. Evidence entry copy is simply `证据`; do not expose counts on cards.
- Do not show internal report, conclusion, chain, anchor, node, or Evidence IDs in list cards or Evidence sheets. Keep IDs only in the in-memory/report contract for traceability. Anchor chips are text plus result state; do not add a leading decorative pin or category icon.
- Evidence access on cards is an icon-only highlighted document control with an accessible `查看证据` label; do not render a text-sized `证据` button.
- Evidence sheets start directly with a newest-first timeline. Each card projects only `published_at`, `summary`, and ordered `keywords` from the Evidence contract; omit layer-specific sheet titles, source/type metadata, relationship stance, and evidence-boundary labels from this compact view.
- Evidence `keywords` render as compact blue-tinted chips with a visible border; they are visually subordinate to publication time and summary but must not collapse into plain inline text.
- Evidence ordering is communicated by newest-first publication time, not by decorative timeline dots or connector lines.
- Geopolitics and macro detail screens do not use a report-title route hero. After displaying their report-owned downward-transmission section, both end with all industry chains published by the same report: chain name plus chain result. The downward-transmission section communicates whether an upper-layer path is direct, inferred, or pending; it must not filter the report's industry-chain list. Upper-layer screens never embed chain nodes or the chain reasoning tree. Selecting a chain pushes that single chain's independent reasoning-detail screen.
- Industry-chain detail starts with the report-owned chain name, one-sentence conclusion, chain result, time window, and confidence. Do not repeat chain path, chain status, or accepted-hypothesis prose above the graph. The node graph lays every node on one horizontal canvas and draws only explicit report Mermaid edges; unconnected peers must not be visually serialized into invented relationships. Relationship paths use orthogonal right-angle routing: adjacent nodes use a horizontal segment, while long edges receive separate horizontal lanes and distinct source/target ports. Every graph node and the selected-node card show result, confidence, time window, and conclusion nature (`直接证据` / `推理假设` / `待验证`). The selected-node card additionally shows report-owned impact and transmission logic, followed by the chain's counterevidence gap and stop condition.

When implementing from a selected generated mock, treat that image as the source of truth for layout, component anatomy, density, spacing, color, typography, visible content, and hierarchy.

## Editing Boundary

- Build app-specific UI in `src/Prototype.tsx` and `src/prototype.css`.
- Treat `src/App.tsx`, `src/main.tsx`, `src/styles.css`, `src/mobile/`, `public/assets/iphone/`, `public/assets/android/`, `public/assets/status/`, `vite.config.ts`, `worker/index.js`, and `scripts/prepare-sites-build.mjs` as protected runtime files. Do not edit, replace, remove, or recreate them unless the user explicitly asks to change the mobile runtime itself. For an explicit runtime change, update the affected lock hashes only after verifying the new runtime behavior.
- Run `npm run check:runtime` before preview or handoff. If it fails, restore the protected runtime instead of weakening or bypassing the check.
- `npm run build` preserves the mobile runtime and prepares the static Cloudflare Worker output required by Sites. Before a Sites handoff, confirm `dist/client/index.html`, `dist/server/index.js`, `dist/.openai/hosting.json`, and source `.openai/hosting.json` exist, then run `npm run test:sites`. Do not replace this project with a Vinext starter.

## Runtime Contract

- Preserve the mobile device runtime unless the user's task explicitly asks otherwise. Do not replace it with a standalone page. Visual fidelity applies to app-owned content inside the device screen, not to template-owned device chrome.
- Keep `App` composed around `PhoneFrame` -> `KeyboardProvider`, with `StatusBar`, app content, `HomeIndicator`, and `KeyboardDock` mounted inside the phone frame. `StatusBar` and the iOS home indicator are overlaid device chrome. When the Android keyboard is closed, the app viewport reserves the protected navigation-bar region instead of painting behind it. When the Android keyboard is open, preserve the current full-screen keyboard layout: its asset includes the IME navigation strip and the separate black navigation bar is hidden. iOS screens continue to paint behind the home-indicator area and own their safe-area content padding.
- Preserve the `iPhone` / `Pixel 10` device picker and both calibrated device presets. The Pixel screen is `427 x 952`; its `32 x 32` camera circle and `public/assets/android/navigation-bar.svg` bottom navigation bar are protected device chrome, not app content.
- Preserve the device picker's intentionally lightweight Codex styling in the top-right corner: its trigger wrapper is borderless and transparent, its trigger sizes to content, and its right-aligned menu uses the compact 3px inset plus the specified hairline and elevation shadow layers. Keep the prototype root and default app screen white.
- Preserve `StatusBar` as live device chrome, including its platform-specific typography, source status-icon assets, and spacing. Pixel 10 uses Roboto, Android indicators, and 32px top, left, and right padding. iPhone uses its iOS indicators, system typography, and calibrated spacing. Do not hardcode screenshot times like `9:41` into the status bar, replace its real-time clock, or move status bar content into app markup unless the user explicitly asks for a fixed/mock device time.
- `PhoneFrame` owns the calibrated device frame, screen portal, device picker, camera cutout, and custom cursor. Keep device assets in `public/assets/iphone/` and `public/assets/android/`; if an asset fails to load, repair the asset path or restore the asset instead of removing the frame, keyboard, or image render.
- Use `MobileScroll` directly for simple single-screen prototypes. Use `FlowStack` for conventional multi-screen flows whose routes can own their fixed header and footer; when using it, define each route as a `FlowScreen`: `{ id, header?, headerHeight?, footer?, footerHeight?, render }`, and use `flow.push(screen)`, `flow.pop()`, and `flow.replace(screen)` from `FlowStack` render callbacks or `useFlow()` instead of introducing another router.
- Use `Carousel` for a carousel, horizontal rail, swipeable cards, image or media strip, horizontally scrollable cards, chip rail, or other horizontal collection.
- For a layered app shell—such as a persistent composer, independently presented sheet, pushed/peek sidebar, or app-wide transition—compose directly in `Prototype.tsx` rather than forcing it through `FlowStack`. Keep app-owned fixed chrome as sibling layers outside `MobileScroll`.
- When using `FlowScreen`, put route-owned fixed headers or footers in `FlowScreen.header` or `FlowScreen.footer`. Set `headerHeight` to the visible app-toolbar height; `FlowStack` adds the device's top safe-area/status-bar inset automatically. Do not include `StatusBar` or its height in the header. Set `footerHeight` to the full app-footer height. `FlowScreen.footer` is an overlay, not reserved layout space; screens using it must add their own bottom content padding such as `padding-bottom: calc(var(--flow-footer-height) + var(--mobile-safe-area-height) + 24px)` so final content can scroll above the footer while still painting behind it.
- Render only scrollable content inside `MobileScroll`; it is for content that should move with scroll and rubber-band overscroll. Keep app-owned headers, nav bars, tabs, composers, and overlays outside it. This keeps scroll physics, safe areas, keyboard insets, scrollbars, and drag click suppression active without letting content paint under fixed chrome.
- Buttons, links, cards, and images inside `MobileScroll` should still allow drag scrolling when the pointer moves beyond tap slop. Use `data-scroll-drag="ignore"` only for rare controls that must own the drag gesture themselves.
- Do not add `var(--keyboard-height)` to ordinary screen/content padding inside `MobileScroll`; the scroll viewport already shrinks above the simulated keyboard. For custom fixed composers, search bars, or toast chrome, use `useKeyboardInsets().bottomInset`. It is relative to the app viewport: Android returns `0` while the closed-keyboard viewport already reserves navigation, then returns the keyboard height while open; iOS continues to clear the home indicator while closed and ride directly above the keyboard while open. Do not pin custom bottom chrome to `bottom: 0` or only `keyboardHeight`.
- Use `KeyboardInput`, `KeyboardTextarea`, or `MobileTextField` for every text-entry control. A raw `input` or `textarea` disconnects focus, keyboard animation, safe-area insets, and attached surfaces.
- Use `BottomSheet` for phone-scoped sheets. Its props are `open`, `onOpenChange`, `title`, optional `description`, optional `snap`, and `children`; it renders through the phone screen portal and dismisses the keyboard before opening.

## Horizontal Carousels

- Use `Carousel` for horizontally draggable cards, images, media, chips, or other horizontal collections. Do not recreate these with `overflow-x`, custom pointer handlers, or a generic div.
- `Carousel` can be nested directly inside `MobileScroll`. It owns horizontal gestures and automatically yields vertical gestures to the parent.
- Never put `data-scroll-drag="ignore"` on or around a `Carousel`; doing so prevents vertical parent scrolling when a gesture begins inside it.
- Do not add CSS scroll snapping to `Carousel`; its runtime owns momentum and release motion.
- Use `data-scroll-drag="ignore"` only when a control must prevent parent scrolling in every drag direction.

See `src/mobile/COMPONENTS.md` for the full component and gesture contract.

## Keyboard Rule

The simulated keyboard is a separate top-layer component. Before presenting anything that behaves like iOS navigation or modal UI, dismiss it first.

Call `keyboard.hide()` before:

- pushing, popping, or replacing FlowStack routes
- opening bottom sheets, action sheets, dialogs, menus, or navigation sheets
- starting transitions where the destination should not inherit text-input focus

`FlowStack` already hides the keyboard for `push`, `pop`, and `replace`. `BottomSheet` already hides it before opening. If you add new modal/sheet/navigation primitives, follow the same rule.

When a composer, search surface, or other keyboard-attached component closes, call `keyboard.hide()` in the same event before changing that component's open state. Position attached surfaces from `useKeyboardInsets()` rather than a separate timer or visibility flag so both dismiss together.

When any text-entry control loses focus, dismiss the simulated keyboard. If the control is custom or does not use the runtime's keyboard-aware fields, handle its blur event and call `keyboard.hide()` explicitly. Keep the keyboard open only when focus is moving directly to another text-entry control that should share the same keyboard session.

## Interaction Rules

- Do not trigger buttons or inputs after a pointer has become a drag. Preserve the drag suppression behavior in `MobileScroll`.
- Do not allow native browser image/file dragging inside the phone frame. Preserve the phone-level `dragstart` suppression and non-draggable image styles so scroll drags that begin on images still scroll the prototype.
- Use `KeyboardInput`, `KeyboardTextarea`, or `MobileTextField` for text entry so the simulated keyboard and safe-area insets stay connected.
- Fixed phone chrome should not animate with pushed screens. Screen content can animate; the status bar, camera cutout, and preview chrome should stay put.
- Keep the keyboard below the home indicator/safe area layer in z-index, and above ordinary app UI while visible.
- Keep the home indicator as the topmost safe-area layer in the z-index above everything else in the prototype.
