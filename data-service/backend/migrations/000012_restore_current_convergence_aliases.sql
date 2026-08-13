-- +goose Up
WITH current_manifest AS (
    SELECT MAX(manifest_version) AS manifest_version
    FROM entity_convergence_manifests
), current_aliases AS (
    SELECT am.to_entity_id, array_agg(DISTINCT am.alias ORDER BY am.alias) AS aliases
    FROM entity_convergence_alias_moves am
    JOIN entity_convergences c ON c.id = am.convergence_id
    JOIN current_manifest cm ON cm.manifest_version = c.manifest_version
    GROUP BY am.to_entity_id
), repaired AS (
    SELECT n.id,
           ARRAY(SELECT DISTINCT value FROM unnest(n.aliases || ca.aliases) AS value ORDER BY value) AS aliases
    FROM entity_nodes n
    JOIN current_aliases ca ON ca.to_entity_id = n.id
)
UPDATE entity_nodes n
SET aliases = repaired.aliases,
    updated_at = NOW()
FROM repaired
WHERE n.id = repaired.id
  AND n.aliases IS DISTINCT FROM repaired.aliases
  AND EXISTS (
      SELECT 1 FROM unnest(repaired.aliases) AS value
      WHERE NOT (value = ANY(n.aliases))
  );

-- +goose Down
SELECT 'convergence alias repair rollback requires a reviewed forward migration or restored backup';
