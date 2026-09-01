# Report Miniapp Design QA

## QA setup

- Reference URL: `http://127.0.0.1:4174/?variant=A`
- Implementation URL: `http://127.0.0.1:4175/`
- State: the same 2026-09-01 investment reasoning report sample, using the Report mock port only for visual QA.
- Viewport: 393 x 852 CSS pixels, DPR 1.
- Reference capture: 1400 x 1200 source screenshots; the prototype device viewport was measured at 393 x 852 and normalized to that size for comparison.
- Implementation capture: 393 x 852 screenshots from the latest H5 mock build.

## Same-input comparisons

| Surface | Reference | Implementation | Result |
| --- | --- | --- | --- |
| Today reasoning home | `/Users/meierlink/.codex/visualizations/2026/09/01/01a05c0f-87f2-7a23-abb0-f508ae72afee/tidewise-report-qa/source-home-mobile.jpg` | `/Users/meierlink/.codex/visualizations/2026/09/01/01a05c0f-87f2-7a23-abb0-f508ae72afee/tidewise-report-qa/implementation-home-final.png` | passed |
| Geopolitics detail | `/Users/meierlink/.codex/visualizations/2026/09/01/01a05c0f-87f2-7a23-abb0-f508ae72afee/tidewise-report-qa/source-detail-mobile.png` | `/Users/meierlink/.codex/visualizations/2026/09/01/01a05c0f-87f2-7a23-abb0-f508ae72afee/tidewise-report-qa/implementation-geo-detail.png` | passed |
| Industry-chain detail | `/Users/meierlink/.codex/visualizations/2026/09/01/01a05c0f-87f2-7a23-abb0-f508ae72afee/tidewise-report-qa/source-chain-mobile.jpg` | `/Users/meierlink/.codex/visualizations/2026/09/01/01a05c0f-87f2-7a23-abb0-f508ae72afee/tidewise-report-qa/implementation-chain-detail.png` | passed |
| Related Evidence page | `/Users/meierlink/.codex/visualizations/2026/09/01/01a05c0f-87f2-7a23-abb0-f508ae72afee/tidewise-report-qa/source-evidence-mobile.jpg` | `/Users/meierlink/.codex/visualizations/2026/09/01/01a05c0f-87f2-7a23-abb0-f508ae72afee/tidewise-report-qa/implementation-evidence-detail.png` | passed |

The four pairs were opened together at the same normalized viewport after the final fixes. The chain edge was also checked in the live DOM: its final H5 style is `top:3.25rem;left:6.9rem;width:1.4rem`.

## Visual review

- Typography: hierarchy, weights, line lengths, and Chinese body-copy density follow the prototype's conclusion-first reading order. No clipped business copy or broken glyphs were found.
- Spacing: card padding, one-object-per-row impact lists, compact right-aligned result/confidence/time signals, and section gaps match the reference rhythm. No overlap or unintended crop remains.
- Colors: navy shell, blue layer accents, green chain state, neutral surfaces, borders, and semantic warming/cooling/diverging chips remain consistent with the reference.
- Image and asset quality: visible UI icons use local official Radix Icons v1.3.2 SVG assets. The chain arrows are data-visualization geometry, not decorative placeholder art. Assets render sharply at DPR 1.
- Copy: the report sample drives every card, detail, node, and Evidence row. The implementation keeps Evidence terminology and does not surface Event data, internal Evidence IDs, roles, ordering, or counts.

## Interaction and runtime review

- Home card -> geopolitics detail: passed.
- Geopolitics downward transmission -> industry-chain detail: passed.
- Chain node -> related Evidence registered page: passed.
- Evidence title is decoded once on H5 and displayed as `油品运输服务证据`: passed.
- Evidence page resets both the Taro page container and H5 window scroll; its timestamp row is no longer hidden behind the navigation bar: passed.
- Industry-chain edge and arrow have non-null style strings after H5 rendering: passed.
- Browser error/warning log after the complete click path: empty.

## Fix history

1. Replaced the CSS/div file placeholder with the official Radix FileText asset.
2. Added platform-aware single-level H5 title decoding and preserved already-decoded Weapp/TT titles, including literal percent sequences.
3. Replaced object-valued graph layout styles, which Taro H5 dropped, with `Taro.pxTransform` style strings.
4. Replaced text/CSS icon stand-ins with official Radix ArrowRight, Clock, Globe, BarChart, Layers, and Cube assets.
5. Reset H5 window scroll after route/data changes so the Evidence first row opens below the navigation bar.

## Accepted product-scope differences

- The existing Miniapp shell is intentionally retained, as required; only the Report home content, detail page, and Evidence page were changed.
- The prototype uses a bottom sheet for Evidence, while the approved implementation uses a registered Taro Evidence subpage to provide the requested tap-to-open page behavior across H5, Weapp, and TT.
- Report group title and publication metadata are present so multiple reports published on the same day remain visibly separated instead of being merged.

No actionable P0, P1, or P2 visual issue remains.

final result: passed
