---
status: accepted
date: 2026-08-24
issue: 334
supersedes_in_part: 0019-database-independent-domain-object-identities.md, 0022-independent-industry-and-concept.md
---

# Retire Entity external identifiers and redirects

## Context

`entity_external_identifiers` was introduced as a generic vendor-taxonomy mapping for Entity objects.
Its remaining rows represent legacy Eastmoney and THS Industry, Concept, and ChainNode mappings, but no
current Data API, Adapter, import command, or consumer owns their lifecycle.

`entity_redirects` stores generic merge or reclassification redirects across Data objects. It likewise
has no current application owner. PostgreSQL still uses the table in Redirect validation and shared
object owner delete/truncate guards, so dropping only the table would leave active owner triggers calling
functions that query a missing relation.

## Decision

- Migration `000071` deletes all Entity External Identifier and Entity Redirect rows and drops both
  tables without `CASCADE`. Existing external identifiers are not converted into Company identifiers or
  another mapping store; Redirects are not converted into aliases or Entity Relations.
- The Redirect-only `entity_nodes` trigger, its identity-protection function, and the Redirect validation
  function are removed.
- Shared Data object delete and truncate guards are replaced in the same transaction so they continue to
  protect generic Entity Relations and typed IndustryChain Links without querying retired tables.
- The unused `EntityExternalIdentifier` Biz vocabulary is removed. `EEI` is removed from the current
  closed ID registry; historical migration references remain immutable ledger history.
- `TestEntityExternalIdentifierValidation` is deleted with classification `obsolete`: its owning Biz
  concept and validation contract are retired by this decision. Registry rejection remains covered by
  `TestRetiredEntityExternalIdentifierKindIsRejected`.
- No public API, DTO, replacement model, import path, or cross-service contract is introduced.

## Release and rollback

This is a high-risk, forward-only destructive migration. Operators stop Data and direct writers, capture
a current PostgreSQL recovery point, and confirm `000071` is the only pending migration before applying
it with the candidate image. They then verify ledger version `71`, absence of both tables and Redirect-only
functions, successful execution of retained object-reference guards, and unchanged retained fact counts;
only then does the candidate Data binary resume traffic.

No old/new binary mixed traffic is allowed during DDL. The previous Data binary has no runtime Adapter or
API for either retired store and remains physically schema-compatible, but application-only rollback does
not restore deleted mappings or redirects. Full rollback restores the pre-migration PostgreSQL recovery
point together with the previous application; no down migration is provided.
