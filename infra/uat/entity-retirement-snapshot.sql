BEGIN TRANSACTION ISOLATION LEVEL REPEATABLE READ READ ONLY;
SELECT '__migration__', max(version_id), 'ledger' FROM public.goose_db_version WHERE is_applied;
SELECT format(
  'SELECT %L, count(*), md5(coalesce(string_agg(md5(to_jsonb(t)::text), '''' ORDER BY md5(to_jsonb(t)::text)), '''')) FROM public.%I t;',
  tablename, tablename
)
FROM pg_tables
WHERE schemaname = 'public' AND tablename <> 'goose_db_version'
ORDER BY tablename
\gexec
COMMIT;
