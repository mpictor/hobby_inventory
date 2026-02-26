package db

import (
	"database/sql"
	"fmt"
	"strings"
	"testing"
)

// a very trivial db mock that stores queries and exec's
type mockDB struct {
	queries []string
	execs   []string
}

var _ DB = (*mockDB)(nil)

func (m *mockDB) Query(query string, args ...any) (dbRows, error) {
	if len(args) > 0 {
		v := "%v" + strings.Repeat(",%v", len(args)-1)
		a := append([]any{query}, args...)
		query = fmt.Sprintf("%s {"+v+"}", a...)
	}
	m.queries = append(m.queries, query)
	return &mdRows{}, nil
}
func (m *mockDB) Exec(query string, args ...any) (sql.Result, error) {
	if len(args) > 0 {
		v := "%v" + strings.Repeat(",%v", len(args)-1)
		a := append([]any{query}, args...)
		query = fmt.Sprintf("%s {"+v+"}", a...)
	}
	m.execs = append(m.execs, query)
	return mdResult{rows: 1}, nil
}

func (m *mockDB) checkStatements(t testing.TB, stmts []string) {
	var s int
outer:
	for s < len(stmts) {
		for q := 0; q < len(m.queries); q++ {
			if stmts[s] == m.queries[q] {
				// remove from both lists
				stmts = append(stmts[:s], stmts[s+1:]...)
				m.queries = append(m.queries[:q], m.queries[q+1:]...)
				continue outer
			}
		}
		for x := 0; x < len(m.execs); x++ {
			if stmts[s] == m.execs[x] {
				// remove from both lists
				stmts = append(stmts[:s], stmts[s+1:]...)
				m.execs = append(m.execs[:x], m.execs[x+1:]...)
				continue outer
			}
		}
		// no match on this one
		s++
	}
	if len(stmts) > 0 {
		t.Errorf("missing expected statements:\n%s", strings.Join(stmts, "\n"))
	}
	if len(m.queries) > 0 {
		t.Errorf("unexpected queries:\n%s", strings.Join(m.queries, "\n"))
	}
	if len(m.execs) > 0 {
		t.Errorf("unexpected execs:\n%s", strings.Join(m.execs, "\n"))
	}
}

type mdResult struct{ id, rows int64 }

func (m mdResult) LastInsertId() (int64, error) { return m.id, nil }
func (m mdResult) RowsAffected() (int64, error) { return m.rows, nil }

type mdRows struct{}

var _ dbRows = (*mdRows)(nil)

func (m *mdRows) Close() error           { return nil }
func (m *mdRows) Next() bool             { return false }
func (m *mdRows) Scan(dest ...any) error { return nil }
func (m mdRows) Err() error              { return nil }
