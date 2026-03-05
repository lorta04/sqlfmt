package sqlfmt

import (
	"fmt"
	"regexp"
	"sort"
	"strings"

	"github.com/cockroachdb/cockroachdb-parser/pkg/sql/parser"
	"github.com/cockroachdb/cockroachdb-parser/pkg/sql/parser/statements"
	"github.com/cockroachdb/cockroachdb-parser/pkg/sql/sem/tree"
)

var (
	commentLineRE = regexp.MustCompile(`^--.*\s*`)
	xmlTypeRE     = regexp.MustCompile(`(?i)\bXML\b`)
	moneyTypeRE   = regexp.MustCompile(`(?i)\bMONEY\b`)
)

func parseStatement(stmt string) (stmts statements.Statements, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("parser panic for statement %q: %v", truncateForError(stmt), r)
		}
	}()
	return parser.Parse(stmt)
}

func formatSpecialStatement(cfg tree.PrettyCfg, stmt string) (string, bool, error) {
	trimmed := strings.TrimSpace(strings.TrimSuffix(stmt, ";"))
	if !strings.HasPrefix(strings.ToUpper(trimmed), "DO") {
		return "", false, nil
	}

	body, suffix, quote, ok, err := parseDoStatement(trimmed)
	if !ok || err != nil {
		return "", ok, err
	}

	formattedBody := formatDoBody(cfg, body)
	switch quote {
	case "'":
		return "DO '" + strings.ReplaceAll(formattedBody, "'", "''") + "'" + suffix, true, nil
	case "$$":
		return "DO $$" + formattedBody + "$$" + suffix, true, nil
	default:
		return "", false, nil
	}
}

func parseDoStatement(stmt string) (body string, suffix string, quote string, ok bool, err error) {
	rest := strings.TrimSpace(stmt[2:])
	if rest == "" {
		return "", "", "", false, nil
	}

	switch {
	case strings.HasPrefix(rest, "'"):
		var b strings.Builder
		for i := 1; i < len(rest); i++ {
			if rest[i] != '\'' {
				b.WriteByte(rest[i])
				continue
			}
			if i+1 < len(rest) && rest[i+1] == '\'' {
				b.WriteByte('\'')
				i++
				continue
			}
			return b.String(), strings.TrimSpace(rest[i+1:]), "'", true, nil
		}
		return "", "", "", true, fmt.Errorf("unterminated DO string literal")
	case strings.HasPrefix(rest, "$$"):
		end := strings.LastIndex(rest, "$$")
		if end <= 1 {
			return "", "", "", true, fmt.Errorf("unterminated DO dollar-quoted body")
		}
		return rest[2:end], strings.TrimSpace(rest[end+2:]), "$$", true, nil
	default:
		return "", "", "", false, nil
	}
}

func formatDoBody(cfg tree.PrettyCfg, body string) string {
	body = strings.ReplaceAll(body, "\r\n", "\n")
	body = strings.TrimSpace(body)
	if body == "" {
		return body
	}

	lines := strings.Split(body, "\n")
	var formatted []string
	indent := 0
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			formatted = append(formatted, "")
			continue
		}

		if closesDoBlock(trimmed) && indent > 0 {
			indent--
		}
		formatted = append(formatted, indentString(cfg, indent)+trimmed)
		if opensDoBlock(trimmed) {
			indent++
		}
	}
	return strings.Join(formatted, "\n")
}

func closesDoBlock(line string) bool {
	upper := strings.ToUpper(line)
	return upper == "END" ||
		upper == "END;" ||
		strings.HasPrefix(upper, "END ") ||
		strings.HasPrefix(upper, "END;") ||
		strings.HasPrefix(upper, "ELSE") ||
		strings.HasPrefix(upper, "ELSIF") ||
		strings.HasPrefix(upper, "EXCEPTION")
}

func opensDoBlock(line string) bool {
	upper := strings.ToUpper(line)
	return upper == "BEGIN" ||
		strings.HasSuffix(upper, " THEN") ||
		strings.HasSuffix(upper, " LOOP") ||
		strings.HasSuffix(upper, " CASE")
}

func indentString(cfg tree.PrettyCfg, depth int) string {
	if depth <= 0 {
		return ""
	}
	if cfg.UseTabs {
		return strings.Repeat("\t", depth)
	}
	return strings.Repeat(" ", cfg.TabWidth*depth)
}

func truncateForError(stmt string) string {
	stmt = strings.Join(strings.Fields(stmt), " ")
	if len(stmt) <= 80 {
		return stmt
	}
	return stmt[:77] + "..."
}

type commentAttachment struct {
	anchor   string
	comments []string
}

func reattachStandaloneComments(original string, formatted string) string {
	attachments := collectStandaloneComments(original)
	if len(attachments) == 0 {
		return formatted
	}

	lines := strings.Split(formatted, "\n")
	anchorCounts := map[string]int{}
	used := make([]bool, len(attachments))
	var out []string

	for _, line := range lines {
		anchor := commentAnchor(line)
		if anchor != "" {
			anchorCounts[anchor]++
			ordinal := anchorCounts[anchor]
			for i, attachment := range attachments {
				if used[i] || attachment.anchor != anchor || countAnchorBefore(attachments, i) != ordinal {
					continue
				}
				indent := leadingWhitespace(line)
				for _, comment := range attachment.comments {
					out = append(out, indent+comment)
				}
				used[i] = true
				break
			}
		}
		out = append(out, line)
	}

	result := strings.Join(out, "\n")
	for i, attachment := range attachments {
		if used[i] {
			continue
		}
		re := regexp.MustCompile(`(?i)\b` + regexp.QuoteMeta(attachment.anchor) + `\b`)
		idx := re.FindStringIndex(result)
		if idx == nil {
			continue
		}
		var block strings.Builder
		for _, c := range attachment.comments {
			block.WriteString("\n    ")
			block.WriteString(c)
		}
		block.WriteString("\n    ")
		result = result[:idx[0]] + block.String() + result[idx[0]:]
	}
	return result
}

func collectStandaloneComments(sql string) []commentAttachment {
	lines := strings.Split(strings.ReplaceAll(sql, "\r\n", "\n"), "\n")
	var pending []string
	var attachments []commentAttachment

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		switch {
		case trimmed == "":
			pending = nil
		case strings.HasPrefix(trimmed, "--"):
			pending = append(pending, trimmed)
		default:
			if len(pending) > 0 {
				if anchor := commentAnchor(trimmed); anchor != "" {
					attachments = append(attachments, commentAttachment{
						anchor:   anchor,
						comments: append([]string(nil), pending...),
					})
				}
				pending = nil
			}
			// Also keep trailing inline comments (e.g. "col TYPE -- note").
			if idx := inlineCommentIndex(line); idx >= 0 {
				sqlPart := strings.TrimSpace(line[:idx])
				if anchor := commentAnchor(sqlPart); anchor != "" {
					attachments = append(attachments, commentAttachment{
						anchor:   anchor,
						comments: []string{strings.TrimSpace(line[idx:])},
					})
				}
			}
		}
	}

	return attachments
}

func inlineCommentIndex(line string) int {
	inSingle := false
	inDouble := false
	for i := 0; i+1 < len(line); i++ {
		ch := line[i]
		next := line[i+1]
		if !inDouble && ch == '\'' {
			inSingle = !inSingle
			continue
		}
		if !inSingle && ch == '"' {
			inDouble = !inDouble
			continue
		}
		if !inSingle && !inDouble && ch == '-' && next == '-' {
			return i
		}
	}
	return -1
}

func commentAnchor(line string) string {
	trimmed := strings.TrimSpace(line)
	trimmed = strings.TrimLeft(trimmed, "(,")
	trimmed = strings.TrimSpace(trimmed)
	if trimmed == "" || strings.HasPrefix(trimmed, "--") || trimmed == ")" {
		return ""
	}

	fields := strings.Fields(trimmed)
	if len(fields) == 0 {
		return ""
	}
	return strings.Trim(fields[0], `"`)
}

func countAnchorBefore(attachments []commentAttachment, idx int) int {
	count := 0
	for i := 0; i <= idx; i++ {
		if attachments[i].anchor == attachments[idx].anchor {
			count++
		}
	}
	return count
}

func leadingWhitespace(s string) string {
	var i int
	for i < len(s) && (s[i] == ' ' || s[i] == '\t') {
		i++
	}
	return s[:i]
}

func preserveOriginalColumnTypes(original string, formatted string) string {
	typeByColumn := collectOriginalColumnTypes(original)
	if len(typeByColumn) == 0 {
		return formatted
	}

	columns := make([]string, 0, len(typeByColumn))
	for col := range typeByColumn {
		columns = append(columns, col)
	}
	sort.Slice(columns, func(i, j int) bool {
		return len(columns[i]) > len(columns[j])
	})

	out := formatted
	for _, col := range columns {
		pgType := typeByColumn[col]
		for _, normalizedType := range normalizedTypeCandidates(pgType) {
			re := regexp.MustCompile(`(?i)(\b` + regexp.QuoteMeta(col) + `\b\s+)` + regexp.QuoteMeta(normalizedType))
			out = re.ReplaceAllString(out, `${1}`+pgType)
		}
	}
	return out
}

func collectOriginalColumnTypes(sql string) map[string]string {
	lines := strings.Split(strings.ReplaceAll(sql, "\r\n", "\n"), "\n")
	typeByColumn := map[string]string{}
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "--") {
			continue
		}
		if strings.HasPrefix(trimmed, ")") {
			break
		}
		fields := strings.Fields(strings.TrimSuffix(trimmed, ","))
		if len(fields) < 2 {
			continue
		}
		switch strings.ToUpper(fields[0]) {
		case "CONSTRAINT", "PRIMARY", "FOREIGN", "UNIQUE", "CHECK", "EXCLUDE":
			continue
		}
		col := strings.Trim(fields[0], `"`)
		if col == "" {
			continue
		}
		if typ, ok := extractOriginalType(fields[1:]); ok {
			typeByColumn[col] = typ
		}
	}
	return typeByColumn
}

func extractOriginalType(fields []string) (string, bool) {
	if len(fields) == 0 {
		return "", false
	}
	head := strings.ToUpper(fields[0])
	switch head {
	case "TEXT", "BYTEA", "JSON", "FLOAT", "XML", "MONEY", "SERIAL", "BIGSERIAL", "SMALLSERIAL":
		return head, true
	case "INT", "INTEGER":
		return "INT", true
	case "BIGINT", "SMALLINT":
		return head, true
	}

	if head == "NUMERIC" {
		if len(fields) > 1 && strings.HasPrefix(fields[1], "(") {
			return "NUMERIC" + strings.ToUpper(fields[1]), true
		}
		return "NUMERIC", true
	}
	if strings.HasPrefix(head, "NUMERIC(") {
		return head, true
	}
	if strings.HasPrefix(head, "FLOAT(") {
		return head, true
	}

	return "", false
}

func normalizedTypeCandidates(originalType string) []string {
	switch {
	case originalType == "TEXT":
		return []string{"STRING"}
	case originalType == "BYTEA":
		return []string{"BYTES"}
	case originalType == "JSON":
		return []string{"JSONB"}
	case originalType == "XML":
		return []string{"STRING"}
	case originalType == "MONEY":
		return []string{"DECIMAL"}
	case originalType == "SERIAL":
		return []string{"SERIAL8"}
	case originalType == "BIGSERIAL":
		return []string{"SERIAL8"}
	case originalType == "SMALLSERIAL":
		return []string{"SERIAL2"}
	case originalType == "FLOAT":
		return []string{"FLOAT8"}
	case originalType == "FLOAT(24)":
		return []string{"FLOAT4"}
	case originalType == "FLOAT(53)":
		return []string{"FLOAT8"}
	case originalType == "INT", originalType == "INTEGER", originalType == "BIGINT":
		return []string{"INT8"}
	case originalType == "SMALLINT":
		return []string{"INT2"}
	case originalType == "NUMERIC":
		return []string{"DECIMAL"}
	case strings.HasPrefix(originalType, "NUMERIC("):
		return []string{"DECIMAL" + strings.TrimPrefix(originalType, "NUMERIC")}
	default:
		return []string{originalType}
	}
}

func preprocessUnsupportedTypes(sql string) string {
	upper := strings.ToUpper(sql)
	if !strings.Contains(upper, "CREATE TABLE") {
		return sql
	}
	// Cockroach parser in this module doesn't support XML/MONEY type tokens in table defs.
	sql = xmlTypeRE.ReplaceAllString(sql, "TEXT")
	sql = moneyTypeRE.ReplaceAllString(sql, "DECIMAL")
	return sql
}
