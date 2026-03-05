package sqlfmt

import (
	"strings"
	"testing"

	"github.com/cockroachdb/cockroachdb-parser/pkg/sql/sem/tree"
)

func TestFmtSQLFormatsSingleQuotedDoBlock(t *testing.T) {
	cfg := tree.DefaultPrettyCfg()
	cfg.UseTabs = true
	cfg.TabWidth = 4

	got, err := FmtSQL(cfg, []string{`DO 'BEGIN
  IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = ''wallet_type'') THEN
    CREATE TYPE wallet_type AS ENUM (''CRYPTO'', ''E_MONEY'');
  END IF;
END';`})
	if err != nil {
		t.Fatalf("FmtSQL returned error: %v", err)
	}

	want := "DO 'BEGIN\n\tIF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = ''wallet_type'') THEN\n\t\tCREATE TYPE wallet_type AS ENUM (''CRYPTO'', ''E_MONEY'');\n\tEND IF;\nEND';"
	if got != want {
		t.Fatalf("unexpected formatted DO block:\nwant:\n%s\n\ngot:\n%s", want, got)
	}
}

func TestFmtSQLStillFormatsRegularStatements(t *testing.T) {
	cfg := tree.DefaultPrettyCfg()
	got, err := FmtSQL(cfg, []string{"SELECT 1;"})
	if err != nil {
		t.Fatalf("FmtSQL returned error: %v", err)
	}
	if !strings.Contains(got, "SELECT 1;") {
		t.Fatalf("expected SELECT output, got %q", got)
	}
}

func TestFmtSQLReattachesStandaloneComments(t *testing.T) {
	cfg := tree.DefaultPrettyCfg()
	cfg.UseTabs = false
	cfg.TabWidth = 4
	cfg.LineWidth = 120

	got, err := FmtSQL(cfg, []string{`CREATE TABLE IF NOT EXISTS sample_records (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  -- Ownership
  -- External account reference
  account_reference TEXT,
  -- Classification
  category sample_category NOT NULL,
  -- Request fields
  source_name TEXT NOT NULL,
  source_path TEXT NOT NULL,
  payload TEXT,
  response_text TEXT,
  -- HTTP details
  -- Method name only (e.g. GET, POST, PUT, DELETE)
  request_method TEXT,
  response_code INT,
  -- Queue details
  queue_name TEXT,
  worker_name TEXT,
  -- Status
  is_archived BOOLEAN NOT NULL DEFAULT false
);`})
	if err != nil {
		t.Fatalf("FmtSQL returned error: %v", err)
	}

	wantContains := []string{
		"    -- Ownership\n    -- External account reference\n    account_reference",
		"    -- Classification\n    category",
		"    -- Request fields\n    source_name",
		"    -- HTTP details\n    -- Method name only (e.g. GET, POST, PUT, DELETE)\n    request_method",
		"    -- Queue details\n    queue_name",
		"    -- Status\n    is_archived",
	}
	for _, want := range wantContains {
		if !strings.Contains(got, want) {
			t.Fatalf("expected formatted SQL to contain:\n%s\n\ngot:\n%s", want, got)
		}
	}
}

func TestFmtSQLReattachesInlineComments(t *testing.T) {
	cfg := tree.DefaultPrettyCfg()
	cfg.UseTabs = false
	cfg.TabWidth = 4
	cfg.LineWidth = 120

	got, err := FmtSQL(cfg, []string{`CREATE TABLE IF NOT EXISTS sample_entities (
  entity_id UUID NOT NULL PRIMARY KEY, -- External primary identifier
  owner_id TEXT NOT NULL, -- Upstream owner reference
  payload BYTEA NOT NULL -- Binary payload blob
);`})
	if err != nil {
		t.Fatalf("FmtSQL returned error: %v", err)
	}

	wantContains := []string{
		"    -- External primary identifier\n    entity_id",
		"    -- Upstream owner reference\n    owner_id",
		"    -- Binary payload blob\n    payload",
	}
	for _, want := range wantContains {
		if !strings.Contains(got, want) {
			t.Fatalf("expected formatted SQL to contain:\n%s\n\ngot:\n%s", want, got)
		}
	}
}

func TestFmtSQLPreservesPostgresColumnTypes(t *testing.T) {
	cfg := tree.DefaultPrettyCfg()
	cfg.UseTabs = false
	cfg.TabWidth = 2
	cfg.LineWidth = 120

	got, err := FmtSQL(cfg, []string{`CREATE TABLE IF NOT EXISTS sample_types (
  col_text TEXT,
  col_int INT,
  col_bool BOOLEAN,
  col_bigint BIGINT,
  col_integer integer,
  col_bytes bytea,
  col_json json,
  col_jsonb jsonb,
  col_xml xml,
  col_money money,
  col_serial serial,
  col_bigserial bigserial,
  col_smallserial smallserial,
  col_numeric_plain NUMERIC,
  col_numeric_precision NUMERIC(8),
  col_numeric NUMERIC(10,2),
  col_character_varying_255 CHARACTER VARYING(255),
  col_character_varying CHARACTER VARYING(30),
  col_character_12 CHARACTER(12),
  col_character CHARACTER(5),
  col_double_precision DOUBLE PRECISION,
  col_real REAL,
  col_float FLOAT,
  col_float_24 FLOAT(24),
  col_float_53 FLOAT(53),
  col_smallint SMALLINT,
  col_bpchar BPCHAR,
  col_name NAME,
  col_citext CITEXT,
  col_uuid UUID,
  col_varchar VARCHAR(10),
  col_char_plain CHAR(3),
  col_bit BIT,
  col_bit_8 BIT(8),
  col_varbit VARBIT(8),
  col_int2 INT2,
  col_int4 INT4,
  col_int8 INT8,
  col_bool2 BOOL,
  col_timetz TIMETZ,
  col_timestamp_plain TIMESTAMP,
  col_timestamptz TIMESTAMPTZ,
  col_date DATE,
  col_time_plain TIME,
  col_time_3 TIME(3),
  col_timestamp_3 TIMESTAMP(3),
  col_interval INTERVAL,
  col_interval_year INTERVAL YEAR,
  col_interval_month INTERVAL MONTH,
  col_interval_day INTERVAL DAY,
  col_interval_hour INTERVAL HOUR,
  col_interval_minute INTERVAL MINUTE,
  col_interval_second INTERVAL SECOND,
  col_interval_year_to_month INTERVAL YEAR TO MONTH,
  col_interval_day_to_second INTERVAL DAY TO SECOND,
  col_bit_varying BIT VARYING(8),
  col_dec DEC,
  col_dec_precision DEC(8,2),
  col_time_with_tz TIME WITH TIME ZONE,
  col_time_with_tz_precision TIME(3) WITH TIME ZONE,
  col_time_without_tz TIME WITHOUT TIME ZONE,
  col_timestamp_with_tz TIMESTAMP WITH TIME ZONE,
  col_timestamp_with_tz_precision TIMESTAMP(3) WITH TIME ZONE,
  col_timestamp_without_tz TIMESTAMP WITHOUT TIME ZONE
);`})
	if err != nil {
		t.Fatalf("FmtSQL returned error: %v", err)
	}

	wantContains := []string{
		"col_text\n    TEXT,",
		"col_int\n    INT,",
		"col_bigint\n    BIGINT,",
		"col_integer\n    INT,",
		"col_smallint\n    SMALLINT,",
		"col_bytes\n    BYTEA,",
		"col_json\n    JSON,",
		"col_jsonb\n    JSONB,",
		"col_xml\n    XML,",
		"col_money\n    MONEY,",
		"col_serial\n    SERIAL,",
		"col_bigserial\n    BIGSERIAL,",
		"col_smallserial\n    SMALLSERIAL,",
		"col_numeric_plain\n    NUMERIC,",
		"col_numeric_precision\n    NUMERIC(8),",
		"col_numeric\n    NUMERIC(10,2),",
		"col_float\n    FLOAT,",
		"col_float_24\n    FLOAT(24),",
		"col_float_53\n    FLOAT(53),",
		"col_bpchar\n    BPCHAR,",
		"col_name\n    NAME,",
		"col_citext\n    citext,",
		"col_uuid\n    UUID,",
		"col_varchar\n    VARCHAR(10),",
		"col_char_plain\n    CHAR(3),",
		"col_bit\n    BIT,",
		"col_bit_8\n    BIT(8),",
		"col_varbit\n    VARBIT(8),",
		"col_int2\n    INT2,",
		"col_int4\n    INT4,",
		"col_int8\n    INT8,",
		"col_bool2\n    BOOL,",
		"col_timetz\n    TIMETZ,",
		"col_timestamp_plain\n    TIMESTAMP,",
		"col_timestamptz\n    TIMESTAMPTZ,",
		"col_date\n    DATE,",
		"col_time_plain\n    TIME,",
		"col_time_3\n    TIME(3),",
		"col_timestamp_3\n    TIMESTAMP(3),",
		"col_interval\n    INTERVAL,",
		"col_interval_year\n    INTERVAL YEAR,",
		"col_interval_month\n    INTERVAL MONTH,",
		"col_interval_day\n    INTERVAL DAY,",
		"col_interval_hour\n    INTERVAL HOUR,",
		"col_interval_minute\n    INTERVAL MINUTE,",
		"col_interval_second\n    INTERVAL SECOND,",
		"col_interval_year_to_month\n    INTERVAL YEAR TO MONTH,",
		"col_interval_day_to_second\n    INTERVAL DAY TO SECOND,",
	}
	for _, want := range wantContains {
		if !strings.Contains(got, want) {
			t.Fatalf("expected formatted SQL to contain:\n%s\n\ngot:\n%s", want, got)
		}
	}

	wantNormalized := []string{
		"col_bool\n    BOOL,",
		"col_character_varying_255\n    VARCHAR(255),",
		"col_character_varying\n    VARCHAR(30),",
		"col_character_12\n    CHAR(12),",
		"col_character\n    CHAR(5),",
		"col_double_precision\n    FLOAT8,",
		"col_real\n    FLOAT4,",
		"col_bit_varying\n    VARBIT(8),",
		"col_dec\n    DECIMAL,",
		"col_dec_precision\n    DECIMAL(8,2),",
		"col_time_with_tz\n    TIMETZ,",
		"col_time_with_tz_precision\n    TIMETZ(3),",
		"col_time_without_tz\n    TIME,",
		"col_timestamp_with_tz\n    TIMESTAMPTZ,",
		"col_timestamp_with_tz_precision\n    TIMESTAMPTZ(3),",
		"col_timestamp_without_tz\n    TIMESTAMP",
	}
	for _, want := range wantNormalized {
		if !strings.Contains(got, want) {
			t.Fatalf("expected formatted SQL to contain normalized type:\n%s\n\ngot:\n%s", want, got)
		}
	}
}
