---
status: accepted
date: 2026-09-07
issue: 441
supersedes_in_part: 0061-normalize-report-assessments-and-latest-selection.md
---

# Publish entity-local signals and bounded report judgments

The approved AgentOS v8 report is published as report-publication/v5. Data preserves entity-local signals, direct/inferred judgment origin, publisher provenance, assessed-node-only graphs, additional industry-chain units and standalone company judgments. The detailed contract is [Report v5](../contexts/data/report-publication-v5.md).

Data owns strict publication, immutable JSONB, Evidence existence/link/counts and read projections. AgentOS owns signal adoption, inference and coverage auditing. Variable and Signal UUIDs and Event IDs are provenance snapshots, not live joins or foreign keys. Reads replace Evidence IDs with report-bound scope tokens, including every signal and company scope.

Reuse the normalized typed structures with optional, version-gated extension fields. Legacy/v3/v4 serialization and canonical hashes remain unchanged; only v5 requires the new fields. Bounded pending judgments are permitted in v5 with the remaining inference requirements intact. Unsupported observation-only anchors and extra graph nodes are rejected.

The existing schema accommodates these changes without a migration. Existing analysis routes add industry_chain_analyses and company_analyses; company results use an explicit company projection rather than inventing a storyline summary. All supported versions remain visible in default report selection. Data deployment and v5 publication precede any consumer rollout; old Miniapp compatibility is not implied. Retain a v5-capable reader after publication, including rollback.
