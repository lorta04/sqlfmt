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
