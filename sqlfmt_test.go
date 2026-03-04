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
