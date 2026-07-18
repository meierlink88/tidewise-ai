# Curated Taro Sources

Use this catalog instead of rescanning the NervJS organization.

## Primary Sources

### NervJS/taro

- Repository: https://github.com/NervJS/taro
- Current examples: https://github.com/NervJS/taro/tree/main/examples
- Use for current framework behavior, platform plugins, build behavior, and exact feature examples.
- Relevant examples include custom React TabBar, mini-program basics, split chunks, independent subpackages, and list rendering.
- Do not copy Taro's framework monorepo structure into the application.

### Taro 4.x documentation and NervJS/taro-docs

- Documentation: https://docs.taro.zone/docs/
- Source: https://github.com/NervJS/taro-docs
- Use for project organization, React contracts, component styling, routing, lifecycle, APIs, platform support, performance, CompileMode, and virtual lists.
- Documentation contracts take priority over old sample code.

## Conditional Sources

Use each source only for the listed purpose.

| Source | Use | Constraint |
|---|---|---|
| `NervJS/taro-rfcs` | Understand the rationale behind unclear framework behavior | Not an application template |
| `NervJS/taro-plugin-mock` | Reference the idea that mock and real adapters share one contract | Old Taro 2/3 plugin; do not install by default |
| `NervJS/taro-test-utils` | Reference behavior-focused component testing | Current package targets Taro 3.6; do not install without renewed compatibility proof |
| `NervJS/taro-doctor` | Diagnose framework or environment failures | Diagnostic tool only, not a mandatory validation step |
| `NervJS/postcss-pxtransform` | Understand design width and pixel conversion | Already provided through Taro tooling |
| `NervJS/taro-benchmark` | Borrow performance measurement ideas | Not a current app architecture benchmark |
| `NervJS/taro-user-cases` | Observe product outcomes | Case index only; no reusable architecture |

## Known Non-Baselines

Do not use these as implementation baselines unless a future investigation proves a narrowly relevant, current use:

- `taro-project-templates`: old JavaScript, Nerv, Redux/MobX, and wxcloud templates.
- `taro-sample-weapp`: WeChat native-component mixing example; unsuitable as a cross-platform default.
- `taro-ui-sample`: component-library publishing example, not an application architecture.
- `taro-v2ex-hooks`, `taro-todomvc-hooks`, `taro-zhihu-sample`: historical Taro examples.
- Old Taro UI, WeUI, Vant, and Ant Design compatibility repositories: conflict with the custom Tidewise design and target old Taro versions.

## Project Defaults

- Framework: Taro 4 + React + TypeScript.
- Platforms: WeChat first; Douyin compatibility retained.
- UI: custom Tidewise design; no generic UI library by default.
- Backend: independent Go Miniapp Backend; no uniCloud or arbitrary frontend-to-domain-service access.
- State: local React state and small typed modules until shared state is justified.
- Data: mock and real API adapters implement the same typed port.
- Performance: paginate first, limit update scope, optimize images, then consider CompileMode or virtual lists based on measurement.

For the full one-time NervJS survey, see `docs/research/nervjs-repositories-for-miniapp.md`.
