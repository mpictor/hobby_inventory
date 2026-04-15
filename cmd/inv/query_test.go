package main

import (
	"context"
	"testing"
)

func Test_query(t *testing.T) {
	args := []string{"q", "pn=%2222%"}
	if err := execQuery(context.Background(), args); err != nil {
		t.Fatal(err)
	}
}
