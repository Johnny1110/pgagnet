package safesql

import (
	"fmt"
	"regexp"
	"strings"
)

const DefaultRowLimit = 25

var forbidden = map[string]struct{}{
	"INSERT": {}, "UPDATE": {}, "DELETE": {}, "MERGE": {}, "UPSERT": {},
	"DROP": {}, "TRUNCATE": {}, "ALTER": {}, "CREATE": {},
	"GRANT": {}, "REVOKE": {}, "COMMENT": {},
	"VACUUM": {}, "ANALYZE": {}, "CLUSTER": {}, "REINDEX": {},
	"COPY": {}, "CALL": {}, "DO": {},
	"LOCK": {}, "REFRESH": {}, "RESET": {}, "SET": {},
	"PREPARE": {}, "EXECUTE": {}, "DEALLOCATE": {},
	"LISTEN": {}, "NOTIFY": {}, "UNLISTEN": {},
	"BEGIN": {}, "COMMIT": {}, "ROLLBACK": {}, "SAVEPOINT": {},
	"INTO": {},
}

var allowedFirst = map[string]struct{}{
	"SELECT": {}, "WITH": {}, "EXPLAIN": {}, "SHOW": {}, "VALUES": {}, "TABLE": {},
}

var limitable = map[string]struct{}{
	"SELECT": {}, "WITH": {}, "VALUES": {}, "TABLE": {},
}

var (
	reLineComment  = regexp.MustCompile(`(?m)--[^\n]*`)
	reBlockComment = regexp.MustCompile(`(?s)/\*.*?\*/`)
	reSingleQuote  = regexp.MustCompile(`'([^']|'')*'`)
	reDoubleQuote  = regexp.MustCompile(`"([^"]|"")*"`)
	reDollarTagged = regexp.MustCompile(`(?s)\$([A-Za-z_][A-Za-z0-9_]*)?\$.*?\$([A-Za-z_][A-Za-z0-9_]*)?\$`)
)

// EnsureReadOnly performs a sanity check that the SQL contains no
// data-modifying or session-altering keywords and is a single statement.
// Comments and string literals are stripped before scanning so a literal
// like 'DROP' won't trigger a false positive. The DB connection runs the
// query inside a READ ONLY transaction as the authoritative guard.
func EnsureReadOnly(sql string) error {
	stripped := stripLiteralsAndComments(sql)

	if err := ensureSingleStatement(stripped); err != nil {
		return err
	}

	tokens := tokenize(strings.ToUpper(stripped))
	if len(tokens) == 0 {
		return fmt.Errorf("empty SQL")
	}

	if _, ok := allowedFirst[tokens[0]]; !ok {
		return fmt.Errorf("statement must start with SELECT/WITH/EXPLAIN/SHOW/VALUES/TABLE; got %q", tokens[0])
	}

	for _, tok := range tokens {
		if _, bad := forbidden[tok]; bad {
			return fmt.Errorf("forbidden keyword %q; only read-only queries are permitted", tok)
		}
	}
	return nil
}

// AddDefaultLimit appends `LIMIT n` to the SQL when the top-level
// statement is a SELECT/WITH/VALUES/TABLE that does not already specify
// LIMIT or FETCH at parenthesis depth 0. Subquery LIMITs are ignored.
// Returns the (possibly modified) SQL and a bool indicating injection.
// EnsureReadOnly must be called first; this function trusts that the
// input is a single read-only statement.
func AddDefaultLimit(sql string, limit int) (string, bool) {
	stripped := stripLiteralsAndComments(sql)
	tokens := tokenize(strings.ToUpper(stripped))
	if len(tokens) == 0 {
		return sql, false
	}
	if _, ok := limitable[tokens[0]]; !ok {
		return sql, false
	}
	if hasTopLevelLimit(stripped) {
		return sql, false
	}
	trimmed := strings.TrimRight(sql, " \t\n\r;")
	return fmt.Sprintf("%s LIMIT %d", trimmed, limit), true
}

func stripLiteralsAndComments(sql string) string {
	s := reDollarTagged.ReplaceAllString(sql, " ")
	s = reLineComment.ReplaceAllString(s, " ")
	s = reBlockComment.ReplaceAllString(s, " ")
	s = reSingleQuote.ReplaceAllString(s, " ")
	s = reDoubleQuote.ReplaceAllString(s, " ")
	return s
}

func ensureSingleStatement(stripped string) error {
	trimmed := strings.TrimRight(stripped, " \t\n\r;")
	if strings.Contains(trimmed, ";") {
		return fmt.Errorf("multiple statements are not allowed; submit a single query")
	}
	return nil
}

func hasTopLevelLimit(stripped string) bool {
	upper := strings.ToUpper(stripped)
	depth := 0
	i := 0
	for i < len(upper) {
		c := upper[i]
		switch {
		case c == '(':
			depth++
			i++
		case c == ')':
			depth--
			i++
		case isIdentStart(c):
			j := i + 1
			for j < len(upper) && isIdentPart(upper[j]) {
				j++
			}
			if depth == 0 {
				word := upper[i:j]
				if word == "LIMIT" || word == "FETCH" {
					return true
				}
			}
			i = j
		default:
			i++
		}
	}
	return false
}

func isIdentStart(c byte) bool { return c >= 'A' && c <= 'Z' || c == '_' }
func isIdentPart(c byte) bool {
	return c >= 'A' && c <= 'Z' || c >= '0' && c <= '9' || c == '_'
}

func tokenize(s string) []string {
	return strings.FieldsFunc(s, func(r rune) bool {
		switch {
		case r >= 'A' && r <= 'Z':
			return false
		case r >= '0' && r <= '9':
			return false
		case r == '_':
			return false
		}
		return true
	})
}
