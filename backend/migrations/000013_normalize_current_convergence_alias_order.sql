-- +goose Up
WITH current_manifest AS (
    SELECT MAX(manifest_version) AS manifest_version
    FROM entity_convergence_manifests
), ranked_owned_aliases AS (
    SELECT am.to_entity_id,am.alias,am.moved_at,am.id,
           row_number() OVER (PARTITION BY am.to_entity_id,am.alias ORDER BY am.moved_at,am.id) AS alias_rank
    FROM entity_convergence_alias_moves am
    JOIN entity_convergences c ON c.id=am.convergence_id
    JOIN current_manifest cm ON cm.manifest_version=c.manifest_version
), owned_aliases AS (
    SELECT to_entity_id,array_agg(alias ORDER BY moved_at,id) AS aliases
    FROM ranked_owned_aliases WHERE alias_rank=1
    GROUP BY to_entity_id
), desired_aliases AS (
    SELECT n.id,
           COALESCE(array_agg(existing.alias ORDER BY existing.ordinality)
               FILTER (WHERE existing.alias IS NOT NULL AND NOT (existing.alias=ANY(oa.aliases))), '{}'::text[])
           || oa.aliases AS aliases
    FROM entity_nodes n
    JOIN owned_aliases oa ON oa.to_entity_id=n.id
    LEFT JOIN LATERAL unnest(n.aliases) WITH ORDINALITY AS existing(alias,ordinality) ON TRUE
    GROUP BY n.id,oa.aliases
)
UPDATE entity_nodes n
SET aliases=desired.aliases,updated_at=NOW()
FROM desired_aliases desired
WHERE n.id=desired.id AND n.aliases IS DISTINCT FROM desired.aliases;

-- +goose Down
SELECT 'convergence alias order normalization rollback requires a reviewed forward migration or restored backup';
