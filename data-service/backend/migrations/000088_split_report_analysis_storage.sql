-- +goose Up
-- Schema only. Stop old readers/writers; run report-storage maintenance before new readers.
ALTER TABLE reports RENAME TO report_archive;
CREATE TABLE report_publications (
 id varchar(39) PRIMARY KEY REFERENCES report_archive(id),
 publisher_report_id text NOT NULL UNIQUE,
 content_hash text NOT NULL CHECK (content_hash ~ '^[0-9a-f]{64}$'),
 report jsonb NOT NULL CHECK (jsonb_typeof(report)='object'),
 published_at timestamptz NOT NULL,
 evidence_counts jsonb,
 has_geopolitics boolean NOT NULL,
 has_macroeconomics boolean NOT NULL,
 industry_chain_count integer NOT NULL CHECK (industry_chain_count>=0)
);
CREATE INDEX report_publications_order ON report_publications(published_at DESC,id ASC);
CREATE TABLE report_summary (
 id varchar(39) PRIMARY KEY CHECK (id ~ '^RPA[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$'),
 report_id varchar(39) NOT NULL REFERENCES report_publications(id),
 analysis_kind text NOT NULL CHECK (analysis_kind IN ('geopolitical_stories','macroeconomic_stories','concept_analyses','industry_chain_analyses','company_analyses')),
 local_key text NOT NULL CHECK (btrim(local_key)<>''),
 source_id text NOT NULL CHECK (btrim(source_id)<>''),
 title text NOT NULL CHECK (btrim(title)<>''),
 ordinal integer NOT NULL CHECK (ordinal>0),
 summary_data jsonb NOT NULL CHECK (jsonb_typeof(summary_data)='object'),
 UNIQUE(report_id,analysis_kind,local_key),
 UNIQUE(report_id,analysis_kind,ordinal)
);
CREATE INDEX report_summary_source ON report_summary(report_id,analysis_kind,source_id);
CREATE TABLE report_detail (
 id varchar(39) PRIMARY KEY REFERENCES report_summary(id),
 detail_data jsonb NOT NULL CHECK (jsonb_typeof(detail_data)='object')
);
CREATE TRIGGER report_publications_immutable BEFORE UPDATE OR DELETE OR TRUNCATE ON report_publications FOR EACH STATEMENT EXECUTE FUNCTION prevent_report_mutation();
CREATE TRIGGER report_summary_immutable BEFORE UPDATE OR DELETE OR TRUNCATE ON report_summary FOR EACH STATEMENT EXECUTE FUNCTION prevent_report_mutation();
CREATE TRIGGER report_detail_immutable BEFORE UPDATE OR DELETE OR TRUNCATE ON report_detail FOR EACH STATEMENT EXECUTE FUNCTION prevent_report_mutation();
COMMENT ON TABLE report_archive IS 'Original immutable publication for audit/replay and legacy wire projection only; normalized product reads use split storage.';
COMMENT ON TABLE report_detail IS 'One complete normalized analysis unit per matching report_summary.id; includes all its chain details.';
-- +goose Down
-- +goose StatementBegin
DO $$ BEGIN RAISE EXCEPTION 'Report storage cutover is forward-only; restore reviewed backup with matching application'; END $$;
-- +goose StatementEnd
