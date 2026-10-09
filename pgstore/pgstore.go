// Package pgstore is a tsukuyomi.Store that keeps the state in a column of any Postgres table.
package pgstore

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/dDan4a/tsukuyomi"
)

// DB is satisfied by *pgxpool.Pool, *pgx.Conn and pgx.Tx, so a transition can run inside the caller's transaction.
type DB interface {
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
}

// Table points the store at the entity table, e.g. {Name: "orders", IDColumn: "id", StateColumn: "status"}.
// Name may be schema-qualified ("billing.payments").
type Table struct {
	Name        string
	IDColumn    string
	StateColumn string
}

type Store[ID, S comparable] struct {
	db         DB
	selectStmt string
	updateStmt string
}

func New[ID, S comparable](db DB, t Table) *Store[ID, S] {
	table := pgx.Identifier(strings.SplitN(t.Name, ".", 2)).Sanitize()
	id := pgx.Identifier{t.IDColumn}.Sanitize()
	state := pgx.Identifier{t.StateColumn}.Sanitize()
	return &Store[ID, S]{
		db:         db,
		selectStmt: fmt.Sprintf("SELECT %s FROM %s WHERE %s = $1", state, table, id),
		updateStmt: fmt.Sprintf("UPDATE %s SET %s = $3 WHERE %s = $1 AND %s = $2", table, state, id, state),
	}
}

func (s *Store[ID, S]) State(ctx context.Context, id ID) (S, error) {
	var state S
	err := s.db.QueryRow(ctx, s.selectStmt, id).Scan(&state)
	if errors.Is(err, pgx.ErrNoRows) {
		return state, tsukuyomi.ErrNotFound
	}
	return state, err
}

func (s *Store[ID, S]) CompareAndSwap(ctx context.Context, id ID, from, to S) (bool, error) {
	tag, err := s.db.Exec(ctx, s.updateStmt, id, from, to)
	if err != nil {
		return false, err
	}
	return tag.RowsAffected() == 1, nil
}
