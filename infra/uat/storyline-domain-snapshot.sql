BEGIN TRANSACTION ISOLATION LEVEL REPEATABLE READ READ ONLY;
SELECT '__migration__', max(version_id), 'ledger' FROM public.goose_db_version WHERE is_applied;
-- Compare every retained field, excluding only the scalar columns being moved.
SELECT format(
  'SELECT %L, count(*), md5(coalesce(string_agg(md5((%s)::text), '''' ORDER BY md5((%s)::text)), '''')) FROM public.%I t;',
  tablename,
  CASE tablename WHEN 'geopolitic_rivalries' THEN 'to_jsonb(t) - ''geopolitic_domain_id'''
                 WHEN 'macro_economics' THEN 'to_jsonb(t) - ''macro_economics_domain_id'''
                 ELSE 'to_jsonb(t)' END,
  CASE tablename WHEN 'geopolitic_rivalries' THEN 'to_jsonb(t) - ''geopolitic_domain_id'''
                 WHEN 'macro_economics' THEN 'to_jsonb(t) - ''macro_economics_domain_id'''
                 ELSE 'to_jsonb(t)' END,
  tablename
)
FROM pg_tables WHERE schemaname='public' AND tablename <> 'goose_db_version' ORDER BY tablename
\gexec
-- Snapshot original and target relationships in exactly the same endpoint format.
SELECT CASE WHEN EXISTS (SELECT 1 FROM information_schema.columns WHERE table_schema='public' AND table_name='geopolitic_rivalries' AND column_name='geopolitic_domain_id') THEN
  'SELECT ''__geo_memberships__'', count(*), md5(coalesce(string_agg(id || ''|'' || geopolitic_domain_id, '''' ORDER BY id, geopolitic_domain_id), '''')) FROM public.geopolitic_rivalries'
ELSE
  'SELECT ''__geo_memberships__'', count(*), md5(coalesce(string_agg(geopolitic_rivalry_id || ''|'' || geopolitic_domain_id, '''' ORDER BY geopolitic_rivalry_id, geopolitic_domain_id), '''')) FROM public.geopolitic_rivalry_domain_links'
END
\gexec
SELECT CASE WHEN EXISTS (SELECT 1 FROM information_schema.columns WHERE table_schema='public' AND table_name='macro_economics' AND column_name='macro_economics_domain_id') THEN
  'SELECT ''__macro_memberships__'', count(*), md5(coalesce(string_agg(id || ''|'' || macro_economics_domain_id, '''' ORDER BY id, macro_economics_domain_id), '''')) FROM public.macro_economics'
ELSE
  'SELECT ''__macro_memberships__'', count(*), md5(coalesce(string_agg(macro_economic_id || ''|'' || macro_economic_domain_id, '''' ORDER BY macro_economic_id, macro_economic_domain_id), '''')) FROM public.macro_economic_domain_links'
END
\gexec
COMMIT;
