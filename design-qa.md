# Report Miniapp Homepage Design QA

## QA scope

- Scope is limited to the existing Miniapp homepage shell and the Report content below `今日推理`.
- The existing avatar, `观潮` title, search field, and send action are retained; their Taro component and style implementation was not replaced by prototype code.
- Detail pages are outside this acceptance pass. The homepage `看传导` action is checked only as a navigation boundary.

## QA setup

- Reference URL: `http://127.0.0.1:4174/?variant=A`
- Implementation URL: `http://127.0.0.1:4175/?qa=deliverable-2130#/pages/index/index`
- Viewport: 393 x 852 CSS pixels; implementation DPR 2.
- State: the Report mock port contains the exact homepage projection of `investment-reasoning-report-2026-09-01-transmission-hypotheses.md` for visual QA.
- Data boundary: this is not a production data-chain acceptance. Migration `000079_add_report_publications.sql` exists in source, but the runtime PostgreSQL database has not yet been migrated and populated with this report.

## Same-input visual comparisons

The reference and implementation were captured in the same in-app Browser session, emitted together, and reviewed at the same viewport and scroll state.

| Surface               | Reference                                                                                                                                                             | Taro implementation                                                                                                                                              | Result |
| --------------------- | --------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------- | ------ |
| Homepage first screen | `/Users/meierlink/.codex/visualizations/2026/09/01/01a05c0f-87f2-7a23-abb0-f508ae72afee/tidewise-report-homepage-rework/prototype-home-final-reference.png`           | `/Users/meierlink/.codex/visualizations/2026/09/01/01a05c0f-87f2-7a23-abb0-f508ae72afee/tidewise-report-homepage-rework/implementation-home-final.png`           | passed |
| Industry-chain cards  | `/Users/meierlink/.codex/visualizations/2026/09/01/01a05c0f-87f2-7a23-abb0-f508ae72afee/tidewise-report-homepage-rework/prototype-industry-final-reference.png`       | `/Users/meierlink/.codex/visualizations/2026/09/01/01a05c0f-87f2-7a23-abb0-f508ae72afee/tidewise-report-homepage-rework/implementation-industry-final.png`       | passed |
| Card Evidence sheet   | `/Users/meierlink/.codex/visualizations/2026/09/01/01a05c0f-87f2-7a23-abb0-f508ae72afee/tidewise-report-homepage-rework/prototype-evidence-sheet-final-reference.png` | `/Users/meierlink/.codex/visualizations/2026/09/01/01a05c0f-87f2-7a23-abb0-f508ae72afee/tidewise-report-homepage-rework/implementation-evidence-sheet-final.png` | passed |

## Visual review

- The report group uses one compact publication row and does not expose the report title.
- Geopolitics and macroeconomics headings sit outside their conclusion cards.
- Each card begins with the one-sentence conclusion, then renders one anchor or node per full-width row.
- Result, confidence, and time window stay right-aligned on the same row. Only the result uses a semantic color pill; confidence and time remain neutral inline signals.
- The industry section shows `54 条真实产业链 · 首页展示 4 条`, a green chain accent, the chain identity row, and four chain cards.
- The card footer contains one `依据` action and one `看传导` action at card level. No Evidence control appears inside an anchor or node row.
- Local SVG assets are rendered through Taro `Image`; no prototype HTML, inline SVG, CSS drawing, emoji, or text-symbol icon was copied into the Miniapp.

## Content and interaction review

- Homepage impact rows: 21 total (`2 + 2 + 5 + 5 + 5 + 2`).
- Card Evidence actions: 6 total. Anchor/node Evidence actions: 0.
- Whole-card navigation roles: 0. Dedicated `看传导` actions: 6.
- Opening `依据` keeps the homepage URL unchanged and draws a content-sized Taro bottom sheet from the bottom, capped at 76vh.
- The sheet loads the selected persisted Evidence scope through `ReportPort`, and exposes only publication time, summary, and keywords; it does not show Event, Evidence ID, source metadata, or counts.
- `看传导` navigates to the expected report detail route with `reportId`, `targetType`, and `targetKey`.
- The card projection has an automated field-level difference count of 0 against the prototype report-data JSON. The six card Evidence lists also use the same referenced objects and order; geopolitics renders all five persisted objects.

## Taro implementation review

- Page and sheet UI use `@tarojs/components` (`View`, `Text`, `Image`, `Button`, and `ScrollView`) and Taro navigation/platform APIs.
- The implementation remains in the existing page/component/module structure; the standalone React/Vite prototype implementation was not copied.
- New report colors and shadows are declared in `src/styles/tokens.scss`; page styles consume those tokens instead of scattering a second palette.
- H5, WeChat Mini Program, and Douyin Mini Program builds complete. WeChat and Douyin output verification scripts pass.
- Frontend tests, TypeScript type checking, and ESLint pass.

## Intentional source-of-truth differences

- The product heading remains `今日推理`, as required by the current Miniapp context, while the standalone reference prototype displays `今日观潮`.
- The implementation displays Data Service `published_at` (`2026.09.01 12:45`) as the publication fact. It does not substitute the report-generation time (`12:39`).
- The original Miniapp shell is retained exactly as requested instead of adopting the standalone prototype's simulated device shell.

No actionable homepage P0, P1, or P2 visual or interaction issue remains in this mock-data acceptance pass.

final result: passed
