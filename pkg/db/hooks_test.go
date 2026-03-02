package db

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
)

// SQLHooks satisfies the sqlhook.SQLHooks interface
type SQLHooks []HookEvent
type HookEvent struct {
	begin, end time.Time
	query      string
	nargs      int
}

// ctx key
type hbegin struct{}

// Before hook will print the query with it's args and return the context with the timestamp
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

	*h = append(*h, HookEvent{
		begin: begin,
		end:   end,
		query: query,
		nargs: len(args),
	})
	return ctx, nil
}

func (h *SQLHooks) checkStatements(t testing.TB, wantStatements []string) {
	t.Helper()
	var gotStatements []string
	for _, q := range *h {
		gotStatements = append(gotStatements, q.query)
	}
	if d := cmp.Diff(wantStatements, gotStatements); len(d) > 0 {
		t.Errorf("result differs: --want ++got\n%s", d)
	}
}
