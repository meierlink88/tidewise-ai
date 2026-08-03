# Analyst snapshot V3 prepared fixture

`01-uat-at01-prepared-request.json` is the canonical Data publication request prepared from
Theme Analyst UAT Theme `AT-01` in run
`uat-analyst-first-cached-20260730t0300-0700-20260803t110116z`.

The fixture intentionally contains no Entity, IndustryChain, VariableDefinition,
VariableSignal, DirectImpact, Relation, or GraphEdge IDs. It is a Theme Analyst-owned
presentation preparation, not a mechanical copy of `analysis.json`.

Field ownership:

- UAT source fields: Theme title/conclusion, focus-node keys and names, Event/Evidence IDs,
  Tree keys/display names/conclusions, Node keys/display names/variable states/judgments/why,
  transmission mechanisms/conditions, boundaries, and checkpoint prose.
- Theme Analyst presentation preparation: publication and batch keys, Theme impact role/order,
  Miniapp investment guidance/time horizon/strength/stage fields, normalized lowercase local
  keys, Tree titles/orders, structured signal keys/roles/orders, and checkpoint type enums.
- Data Service: no semantic completion. It validates, atomically stores, hashes, replays, and
  reads back the submitted snapshot.
