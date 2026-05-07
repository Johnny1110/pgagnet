package db

import (
	"database/sql"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/mattn/go-runewidth"
)

type alignment uint8

const (
	alignLeft alignment = iota
	alignRight
)

const (
	cellPad     = 1
	maxCellChar = 120
)

func renderRows(rows *sql.Rows, w io.Writer) error {
	cols, err := rows.Columns()
	if err != nil {
		return err
	}
	colTypes, err := rows.ColumnTypes()
	if err != nil {
		return err
	}

	aligns := make([]alignment, len(cols))
	formatters := make([]func(any) string, len(cols))
	for i, ct := range colTypes {
		aligns[i] = alignmentFor(ct.DatabaseTypeName())
		formatters[i] = formatterFor(ct.DatabaseTypeName())
	}

	vals := make([]any, len(cols))
	ptrs := make([]any, len(cols))
	for i := range vals {
		ptrs[i] = &vals[i]
	}

	cells := [][]string{}
	for rows.Next() {
		if err := rows.Scan(ptrs...); err != nil {
			return err
		}
		row := make([]string, len(cols))
		for i, v := range vals {
			row[i] = truncate(formatters[i](v), maxCellChar)
		}
		cells = append(cells, row)
	}
	if err := rows.Err(); err != nil {
		return err
	}

	widths := make([]int, len(cols))
	for i, h := range cols {
		widths[i] = runewidth.StringWidth(h)
	}
	for _, row := range cells {
		for i, c := range row {
			if n := runewidth.StringWidth(c); n > widths[i] {
				widths[i] = n
			}
		}
	}

	drawTable(w, cols, cells, widths, aligns)
	fmt.Fprintf(w, "(%d %s)\n", len(cells), pluralize("row", len(cells)))
	return nil
}

func drawTable(w io.Writer, headers []string, rows [][]string, widths []int, aligns []alignment) {
	border := func(l, m, r string) {
		var b strings.Builder
		b.WriteString(l)
		for i, width := range widths {
			if i > 0 {
				b.WriteString(m)
			}
			b.WriteString(strings.Repeat("─", width+cellPad*2))
		}
		b.WriteString(r)
		b.WriteString("\n")
		_, _ = io.WriteString(w, b.String())
	}

	writeRow := func(cells []string, isHeader bool) {
		var b strings.Builder
		b.WriteString("│")
		for i, c := range cells {
			b.WriteString(strings.Repeat(" ", cellPad))
			a := aligns[i]
			if isHeader {
				a = alignLeft
			}
			b.WriteString(padCell(c, widths[i], a))
			b.WriteString(strings.Repeat(" ", cellPad))
			b.WriteString("│")
		}
		b.WriteString("\n")
		_, _ = io.WriteString(w, b.String())
	}

	border("╭", "┬", "╮")
	writeRow(headers, true)
	border("├", "┼", "┤")
	for _, r := range rows {
		writeRow(r, false)
	}
	border("╰", "┴", "╯")
}

func padCell(s string, width int, a alignment) string {
	gap := width - runewidth.StringWidth(s)
	if gap <= 0 {
		return s
	}
	pad := strings.Repeat(" ", gap)
	if a == alignRight {
		return pad + s
	}
	return s + pad
}

func truncate(s string, maxChars int) string {
	if runewidth.StringWidth(s) <= maxChars {
		return s
	}
	return runewidth.Truncate(s, maxChars, "…")
}

func pluralize(word string, n int) string {
	if n == 1 {
		return word
	}
	return word + "s"
}

func alignmentFor(dbType string) alignment {
	switch strings.ToUpper(dbType) {
	case "INT2", "INT4", "INT8", "SMALLINT", "INTEGER", "BIGINT",
		"FLOAT4", "FLOAT8", "REAL", "DOUBLE PRECISION",
		"NUMERIC", "DECIMAL", "MONEY", "OID":
		return alignRight
	}
	return alignLeft
}

func formatterFor(dbType string) func(any) string {
	switch strings.ToUpper(dbType) {
	case "DATE":
		return func(v any) string {
			if v == nil {
				return "NULL"
			}
			if t, ok := v.(time.Time); ok {
				return t.Format("2006-01-02")
			}
			return formatValue(v)
		}
	case "TIMESTAMP", "TIMESTAMPTZ", "TIMESTAMP WITHOUT TIME ZONE", "TIMESTAMP WITH TIME ZONE":
		return func(v any) string {
			if v == nil {
				return "NULL"
			}
			if t, ok := v.(time.Time); ok {
				return t.Format("2006-01-02 15:04:05")
			}
			return formatValue(v)
		}
	}
	return formatValue
}
