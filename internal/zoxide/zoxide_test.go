package zoxide

import (
	"context"
	"strings"
	"testing"

	"github.com/rtuszik/sex/internal/execx"
)

func TestCommands(t *testing.T) {
	var got []string
	run := execx.RunnerFunc(func(_ context.Context, cmd execx.Cmd) ([]byte, error) {
		got = append(got, cmd.Name+" "+strings.Join(cmd.Args, " "))
		return []byte("/home/me/src\n"), nil
	})
	c := New(run)
	if err := c.Add(context.Background(), "/a b"); err != nil {
		t.Fatal(err)
	}
	path, err := c.Query(context.Background(), "src")
	if err != nil || path != "/home/me/src" {
		t.Fatalf("path %q, err %v", path, err)
	}
	if strings.Join(got, ",") != "zoxide add /a b,zoxide query src" {
		t.Fatalf("got %v", got)
	}
}
