# User migration ledger

This is a separate Goose forward-only ledger for the private User database. Run the
`cmd/dbmigrate` binary with `USER_DATABASE_URL` (DDL role) and explicit
`USER_DATABASE_NAME`. No Data Service migrations or user data are imported.

Apply the complete ledger on an empty disposable database in CI, then rerun to
verify a no-op. Runtime credentials need DML on the three business tables and
SELECT on goose_db_version and configuration; server startup never migrates. Run only one migration
job at a time. Do not run down; roll back the application and retain the database.

Risk classification is in `migration-risk.tsv`; User is not yet enabled in the
existing Data UAT migration manifest or release pipeline.

After migration 3, import the WeChat dictionary entry with configure-wechat before starting the server. Migration never seeds credentials.
