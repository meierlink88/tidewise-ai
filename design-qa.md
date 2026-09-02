# Report Miniapp Homepage Design QA

## QA scope

- Scope is limited to the existing Miniapp homepage shell and the Report content below `今日观潮`.
- The existing avatar, `观潮家` title, search field, and send action are retained; their Taro component and style implementation was not replaced by prototype code.
- Detail pages are outside this acceptance pass. The homepage `看传导` action is checked only as a navigation boundary.

## QA setup

- Reference URL: `http://127.0.0.1:4174/?variant=A`
- Implementation URL: `http://127.0.0.1:4175/?qa=deliverable-2130#/pages/index/index`
- Viewport: 393 x 852 CSS pixels; implementation DPR 2.
- State: the Report mock port contains the exact homepage projection of `investment-reasoning-report-2026-09-01-transmission-hypotheses.md` for visual QA.
- Data boundary: this is not a production data-chain acceptance. Migration `000079_add_report_publications.sql` exists in source, but the runtime PostgreSQL database has not yet been migrated and populated with this report.

### 2026-09-02 fixed-header spacing regression pass

- Source visual truth: `/Users/meierlink/.codex/visualizations/2026/09/01/01a05c0f-87f2-7a23-abb0-f508ae72afee/tidewise-report-homepage-rework/prototype-home-final-reference.png`.
- User-reported before state: `/Users/meierlink/.codex/visualizations/2026/09/02/report-home-spacing/before-home-spacing.png`.
- Revised implementation: `/Users/meierlink/.codex/visualizations/2026/09/02/report-home-spacing/after-home-spacing.png`.
- Implementation screenshot: 515 x 853 pixels, 515 x 853 CSS viewport, DPR 2; the browser capture is CSS-pixel normalized.
- State: homepage at the top, fixed application header visible, Report mock ready.
- Focused comparison: the title/header boundary was reviewed together with the archived reference and the revised implementation. No additional focused region was required because this pass changes only that boundary.

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
- The fixed header ends before the `今日观潮` content block. The content keeps the prototype's `34rpx` top inset, equal to 17 CSS pixels at the 375px design width, instead of retaining the obsolete `-40rpx` overlap.

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
- The fixed-header spacer keeps its static `rpx` height in the page stylesheet and applies the status/navigation-bar height as a separate platform-pixel padding. This avoids an invalid runtime `px + rpx` expression and keeps the static geometry in one named style value.
- H5, WeChat Mini Program, and Douyin Mini Program builds complete. WeChat and Douyin output verification scripts pass.
- Frontend tests, TypeScript type checking, and ESLint pass.
- Runtime layout and console checks in this pass cover H5. WeChat and Douyin runtime spacing remain target-platform acceptance items; their successful builds are not treated as visual proof.

## Spacing comparison history

- Earlier P2: `margin-top: -40rpx` cancelled the heading's `34rpx` top padding after the header became fixed, leaving the title visually attached to the header. The H5 runtime also discarded the mixed-unit dynamic spacer, which could place content underneath the fixed header.
- Fix: removed the obsolete negative content margin and separated the spacer's stylesheet `rpx` height from its platform-pixel safe-area padding.
- Post-fix evidence: the current H5 viewport reports `position: fixed`, `margin-top: 0`, a valid spacer height, and no console warning/error. The revised screenshot restores a clear title boundary without changing typography, colors, assets, copy, cards, or interactions.

## Intentional source-of-truth differences

- The product heading is `今日观潮`, following the user's latest explicit product copy decision.
- The implementation displays Data Service `published_at` (`2026.09.01 12:45`) as the publication fact. It does not substitute the report-generation time (`12:39`).
- The original Miniapp shell is retained exactly as requested instead of adopting the standalone prototype's simulated device shell.

No actionable homepage P0, P1, or P2 issue remains in the H5 mock-data acceptance pass. WeChat and Douyin runtime visual acceptance remain pending.

final result: H5 passed; WeChat/Douyin runtime visual acceptance pending

---

# Report detail visual QA

## Visual truth

- Layer heading: `/var/folders/51/02rqhzzj69sbg4m15_kfh8q40000gn/T/codex-clipboard-5bdd0116-3309-4a50-bc6e-d431c155b7a7.png`
- Downward transmission: `/var/folders/51/02rqhzzj69sbg4m15_kfh8q40000gn/T/codex-clipboard-5d7c78e7-d844-4293-b71a-f680e9182e4b.png`
- Chain boundaries: `/var/folders/51/02rqhzzj69sbg4m15_kfh8q40000gn/T/codex-clipboard-8b56a932-b0b1-42ad-9ffd-1aaf4b6326b9.png`

## Implementation state

- Preview: `http://127.0.0.1:4175/?qa=detail-54#/pages/report/detail/index?reportId=RPT11111111-1111-4111-8111-111111111111&targetType=layer&targetKey=geopolitics`
- Browser: Codex in-app browser
- Viewport: 758 x 1100 CSS px
- Data source: approved report prototype snapshot, 54 industry chains
- Detail navigation check: the final row `chn-54` opens the wind-power chain detail.

## Combined comparisons

- Header: `/Users/meierlink/.codex/visualizations/2026/09/02/report-detail-qa/compare-header.png`
- Transmission cards: `/Users/meierlink/.codex/visualizations/2026/09/02/report-detail-qa/compare-transmission.png`
- Chain boundaries: `/Users/meierlink/.codex/visualizations/2026/09/02/report-detail-qa/compare-boundaries.png`

## Findings and iterations

1. The identity row inherited `space-between`, pushing the layer title toward the evidence button. The identity row now explicitly uses `justify-content: flex-start`.
2. Published transmission paths and candidate mechanisms lacked the leading link asset. Both headings now render the same Radix `Link2Icon` asset.
3. The mock port exposed only four manually authored chain details. A generated typed fixture now exposes all 54 report chains to the related-chain list and detail route, while the home showcase remains limited to the approved four cards.
4. The chain gap and stop-condition blocks lacked semantic assets and used mismatched colors. They now use Radix info/warning assets with the neutral and amber prototype treatments.

## Final result

Passed for the four requested discrepancies: heading alignment, transmission link icons, full 54-chain list/detail loading, and chain boundary icons.

---

# Industry-chain detail metric and node QA

## Visual truth

- Summary metrics: `/var/folders/51/02rqhzzj69sbg4m15_kfh8q40000gn/T/codex-clipboard-ccad2f2b-d0fb-4317-a618-8e28b1f6c9f0.png`
- Direct-evidence nodes: `/var/folders/51/02rqhzzj69sbg4m15_kfh8q40000gn/T/codex-clipboard-fc92f3af-d453-4163-8f26-d9feb8d8244c.png`
- Direct/hypothesis comparison: `/var/folders/51/02rqhzzj69sbg4m15_kfh8q40000gn/T/codex-clipboard-e3bc1727-b20a-4ed0-842e-e35bd8bd0c48.png`

## Implementation state

- Preview: `http://127.0.0.1:4175/?qa=chain-node-style-before#/pages/report/detail/index?reportId=RPT11111111-1111-4111-8111-111111111111&targetType=industry_chain&targetKey=chn-01`
- Browser: Codex in-app browser
- Viewport: 724 x 1100 CSS px
- Data source: approved Report mock snapshot

## Combined comparisons

- Summary and metric pills: `/Users/meierlink/.codex/visualizations/2026/09/02/report-chain-detail-qa/compare-summary.png`
- Direct-evidence node cards: `/Users/meierlink/.codex/visualizations/2026/09/02/report-chain-detail-qa/compare-direct-node.png`
- Reasoning-hypothesis node card: `/Users/meierlink/.codex/visualizations/2026/09/02/report-chain-detail-qa/compare-hypothesis-node.png`

## Findings and iterations

1. The chain result, time window, and confidence were presented as a segmented grid. They now render as three independent compact pill cards with their existing semantic icons and values.
2. Node result, confidence, and time remain grouped as factual signals while the nature label has a dedicated bottom-centered position.
3. `direct_evidence` uses a blue nature chip, `reasoning_hypothesis` uses an amber nature chip, and `pending_validation` retains a neutral treatment. The visible text remains present, so the distinction does not depend on color alone.
4. Node width and minimum height were increased to prevent long names and delayed-window copy from crowding the centered nature label.

final result: passed
