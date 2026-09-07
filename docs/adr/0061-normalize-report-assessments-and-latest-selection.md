---
status: accepted
date: 2026-09-07
issue: 432
supersedes_in_part: 0060-version-report-story-and-concept-analyses.md
---

# Normalize Report assessments and stop hiding newer formats

The user approved the v7 authoring baseline. Data accepts it as report-publication/v4,
with one canonical detail assessment for each summary target, shared chain/node
assessments, explicit forecast windows, separate support/counterfacts/buffers/gaps,
and observation-only empty states. Data validates author conclusions and never
infers them. The complete contract is [Report v4](../contexts/data/report-publication-v4.md).

Legacy and v3 immutable snapshots, hashes and replay identities remain unchanged.
The publication envelope and transaction boundary remain the same. Evidence IDs
are published per scope and read back only as report-bound opaque tokens.
Migration 86 expands the Evidence scope CHECK without rewriting history.

The default Report list now includes all supported formats, ordered by publication
time and ID. Explicit legacy/v3/v4 selectors remain available. The default cursor
is bound to `all`, so a historical implicit-legacy cursor cannot silently continue
under the new selection. This supersedes ADR-0060's default legacy filter.

Miniapp must consume the selected report's version, not silently request an older
report. Consumer adoption and UAT deployment are separate tasks; Data changes alone
do not make the current Miniapp understand v4. Deploy Data before publishing v4 and
retain v4 read capability after publication. Application rollback does not reverse
immutable snapshots or guarantee that an old consumer can read a new format.
