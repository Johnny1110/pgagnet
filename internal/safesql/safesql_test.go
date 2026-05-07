package safesql

import "testing"

func TestEnsureReadOnly_Allowed(t *testing.T) {
	cases := []string{
		`SELECT 1`,
		`SELECT * FROM kyc_personal WHERE id = 12231`,
		`  WITH x AS (SELECT 1) SELECT * FROM x`,
		`EXPLAIN SELECT * FROM t`,
		`SHOW search_path`,
		`SELECT 'DROP TABLE x' AS note`,
		`SELECT col -- DROP TABLE x` + "\n" + `FROM t`,
		`SELECT col /* INSERT INTO x */ FROM t`,
		`SELECT * FROM t;`,
	}
	for _, q := range cases {
		if err := EnsureReadOnly(q); err != nil {
			t.Errorf("expected allowed, got %v for %q", err, q)
		}
	}
}

func TestEnsureReadOnly_Rejected(t *testing.T) {
	cases := []string{
		``,
		`INSERT INTO t VALUES (1)`,
		`UPDATE t SET a = 1`,
		`DELETE FROM t`,
		`DROP TABLE t`,
		`SELECT * INTO new_t FROM t`,
		`SET search_path = public`,
		`BEGIN; SELECT 1; COMMIT;`,
		`CREATE TABLE t (id int)`,
		`TRUNCATE t`,
		`GRANT SELECT ON t TO u`,
		`COPY t FROM '/tmp/x.csv'`,
		`CALL my_proc()`,
		`SELECT 1; SELECT 2`,
		`SELECT 1; DROP TABLE t`,
	}
	for _, q := range cases {
		if err := EnsureReadOnly(q); err == nil {
			t.Errorf("expected rejection for %q", q)
		}
	}
}

func TestAddDefaultLimit_Injects(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		{`SELECT * FROM sales`, `SELECT * FROM sales LIMIT 500`},
		{`SELECT * FROM sales;`, `SELECT * FROM sales LIMIT 500`},
		{`SELECT * FROM sales;   `, `SELECT * FROM sales LIMIT 500`},
		{`WITH x AS (SELECT 1) SELECT * FROM x`, `WITH x AS (SELECT 1) SELECT * FROM x LIMIT 500`},
		{`SELECT * FROM (SELECT * FROM s LIMIT 5) q`, `SELECT * FROM (SELECT * FROM s LIMIT 5) q LIMIT 500`},
		{`WITH x AS (SELECT * FROM t LIMIT 10) SELECT * FROM x`, `WITH x AS (SELECT * FROM t LIMIT 10) SELECT * FROM x LIMIT 500`},
		{`SELECT 'has LIMIT inside string' AS s`, `SELECT 'has LIMIT inside string' AS s LIMIT 500`},
		{`TABLE sales`, `TABLE sales LIMIT 500`},
	}
	for _, c := range cases {
		got, injected := AddDefaultLimit(c.in, DefaultRowLimit)
		if !injected {
			t.Errorf("expected injection for %q, got none", c.in)
			continue
		}
		if got != c.want {
			t.Errorf("for %q\n  got:  %q\n  want: %q", c.in, got, c.want)
		}
	}
}

func TestAddDefaultLimit_LeavesAlone(t *testing.T) {
	cases := []string{
		`SELECT * FROM t LIMIT 10`,
		`SELECT * FROM t LIMIT 10;`,
		`SELECT * FROM t FETCH FIRST 10 ROWS ONLY`,
		`SELECT * FROM t LIMIT 10 OFFSET 5`,
		`EXPLAIN SELECT * FROM t`,
		`SHOW search_path`,
	}
	for _, q := range cases {
		got, injected := AddDefaultLimit(q, DefaultRowLimit)
		if injected {
			t.Errorf("did not expect injection for %q (got %q)", q, got)
		}
		if got != q {
			t.Errorf("query unexpectedly modified: in=%q out=%q", q, got)
		}
	}
}
