package pgstore_test

import (
	"context"
	"errors"
	"os"
	"testing"

	"github.com/jackc/pgx/v5"

	"github.com/dDan4a/tsukuyomi"
	"github.com/dDan4a/tsukuyomi/pgstore"
)

type status string

// Runs against a real Postgres: TSUKUYOMI_PG_DSN=postgres://user:pass@localhost:5432/db go test ./...
func TestStore(t *testing.T) {
	dsn := os.Getenv("TSUKUYOMI_PG_DSN")
	if dsn == "" {
		t.Skip("TSUKUYOMI_PG_DSN is not set")
	}
	ctx := context.Background()
	conn, err := pgx.Connect(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close(ctx) //nolint:errcheck // test cleanup

	tx, err := conn.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck // test cleanup

	if _, err := tx.Exec(ctx, `CREATE TEMP TABLE "order" (id bigint PRIMARY KEY, status text NOT NULL);
		INSERT INTO "order" VALUES (1, 'new')`); err != nil {
		t.Fatal(err)
	}

	store := pgstore.New[int64, status](tx, pgstore.Table{Name: "order", IDColumn: "id", StateColumn: "status"})
	m := tsukuyomi.New[int64, status]().Allow("new", "paid")

	if err := m.Transition(ctx, store, 1, "paid"); err != nil {
		t.Fatal(err)
	}
	if got, _ := store.State(ctx, 1); got != "paid" {
		t.Fatalf("state = %v, want paid", got)
	}
	if ok, _ := store.CompareAndSwap(ctx, 1, "new", "paid"); ok {
		t.Fatal("CAS with stale state succeeded")
	}
	if _, err := store.State(ctx, 2); !errors.Is(err, tsukuyomi.ErrNotFound) {
		t.Fatalf("got %v, want ErrNotFound", err)
	}
}
