package db

import (
	"context"
	"database/sql"
	"fmt"
	"io"
	"net/url"
	"strconv"
	"time"

	"pg-agent/internal/config"

	_ "github.com/jackc/pgx/v5/stdlib"
)

const queryTimeout = 30 * time.Second

func RunQuery(ctx context.Context, c config.Database, query string, w io.Writer) error {
	pool, err := sql.Open("pgx", buildDSN(c))
	if err != nil {
		return fmt.Errorf("open: %w", err)
	}
	defer pool.Close()
	pool.SetMaxOpenConns(2)

	queryCtx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	tx, err := pool.BeginTx(queryCtx, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return fmt.Errorf("begin read-only tx: %w", err)
	}
	defer tx.Rollback()

	rows, err := tx.QueryContext(queryCtx, query)
	if err != nil {
		return fmt.Errorf("query: %w", err)
	}
	defer rows.Close()

	return renderRows(rows, w)
}

func buildDSN(c config.Database) string {
	port := c.Port
	if port == 0 {
		port = 5432
	}
	sslmode := c.SSLMode
	if sslmode == "" {
		sslmode = "disable"
	}
	u := url.URL{
		Scheme: "postgres",
		User:   url.UserPassword(c.User, c.Password),
		Host:   c.Host + ":" + strconv.Itoa(port),
		Path:   c.DBName,
	}
	q := u.Query()
	q.Set("sslmode", sslmode)
	u.RawQuery = q.Encode()
	return u.String()
}

func formatValue(v any) string {
	if v == nil {
		return "NULL"
	}
	switch val := v.(type) {
	case []byte:
		return string(val)
	case time.Time:
		return val.Format(time.RFC3339Nano)
	case string:
		return val
	default:
		return fmt.Sprintf("%v", val)
	}
}
