package db

import (
	"context"
	"fmt"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
)

// SQLHooks satisfies the sqlhook.SQLHooks interface
type SQLHooks struct {
	evts  []HookEvent
	trace bool
}
type HookEvent struct {
	begin, end time.Time
	query      string
	nargs      int
	stack      string
}

// ctx key
type hbegin struct{}

// Before hook will print the query with its args and return the context with the timestamp
func (h *SQLHooks) Before(ctx context.Context, query string, args ...interface{}) (context.Context, error) {
	return context.WithValue(ctx, hbegin{}, time.Now()), nil
}

// After hook will get the timestamp registered on the Before hook and print the elapsed time
func (h *SQLHooks) After(ctx context.Context, query string, args ...interface{}) (context.Context, error) {
	end := time.Now()
	begin := ctx.Value(hbegin{}).(time.Time)

	if len(args) > 0 {
		v := "%v" + strings.Repeat(",%v", len(args)-1)
		a := append([]any{query}, args...)
		query = fmt.Sprintf("%s {"+v+"}", a...)
	}

	evt := HookEvent{
		begin: begin,
		end:   end,
		query: query,
		nargs: len(args),
	}
	if h.trace {
		// evt.stack = debug.Stack()
		pc := make([]uintptr, 32)
		n := runtime.Callers(2, pc)
		pc = pc[:n]
		fs := runtime.CallersFrames(pc)
		found := false
		for {
			f, more := fs.Next()
			if strings.Contains(f.File, "pkg/db") {
				found = true
				evt.stack += fmt.Sprintf("%s:%d +++ %s\n", f.File, f.Line, f.Function)
				// TODO
				// break
			} else if found {
				break
			}
			if !more {
				break
			}
		}
		// runtime.Callers()
	}
	h.evts = append(h.evts, evt)
	return ctx, nil
}
func (h *SQLHooks) Clear() { h.evts = nil }

func (h *SQLHooks) checkStatements(t testing.TB, wantStatements []string) {
	t.Helper()
	var gotStatements []string
	for _, q := range h.evts {
		gotStatements = append(gotStatements, q.query)
	}
	if d := cmp.Diff(wantStatements, gotStatements); len(d) > 0 {
		t.Errorf("result differs: --want ++got\n%s", d)
	}
}

func (h *SQLHooks) dump(t testing.TB) {
	t.Helper()
	var gotStatements []string
	for _, q := range h.evts {
		if h.trace {
			gotStatements = append(gotStatements, fmt.Sprintf("%s >> %s <<", q.query, string(q.stack)))
		} else {
			gotStatements = append(gotStatements, q.query)
		}
	}
	t.Logf("sql statements:\n  %s", strings.Join(gotStatements, "\n  "))
}
