package tsukuyomi_test

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/dDan4a/tsukuyomi"
)

type status string

const (
	newOrder  status = "new"
	paid      status = "paid"
	shipped   status = "shipped"
	cancelled status = "cancelled"
)

func orders() *tsukuyomi.Machine[int, status] {
	return tsukuyomi.New[int, status]().
		Allow(newOrder, paid, cancelled).
		Allow(paid, shipped)
}

func TestTransition(t *testing.T) {
	ctx := context.Background()
	store := tsukuyomi.NewMemoryStore[int, status]()
	store.Set(1, newOrder)
	m := orders()

	if err := m.Transition(ctx, store, 1, paid); err != nil {
		t.Fatal(err)
	}
	if err := m.Transition(ctx, store, 1, cancelled); !errors.Is(err, tsukuyomi.ErrInvalidTransition) {
		t.Fatalf("paid -> cancelled: got %v, want ErrInvalidTransition", err)
	}
	if err := m.Transition(ctx, store, 2, paid); !errors.Is(err, tsukuyomi.ErrNotFound) {
		t.Fatalf("unknown id: got %v, want ErrNotFound", err)
	}
	if got, _ := store.State(ctx, 1); got != paid {
		t.Fatalf("state = %v, want paid", got)
	}
}

func TestGuardAndHook(t *testing.T) {
	ctx := context.Background()
	store := tsukuyomi.NewMemoryStore[int, status]()
	store.Set(1, newOrder)

	errUnpaid := errors.New("unpaid")
	var entered []tsukuyomi.Transition[int, status]
	m := orders().
		Guard(newOrder, paid, func(_ context.Context, tr tsukuyomi.Transition[int, status]) error {
			if tr.ID == 1 {
				return errUnpaid
			}
			return nil
		}).
		OnEnter(cancelled, func(_ context.Context, tr tsukuyomi.Transition[int, status]) { entered = append(entered, tr) })

	if err := m.Transition(ctx, store, 1, paid); !errors.Is(err, errUnpaid) {
		t.Fatalf("guard: got %v, want errUnpaid", err)
	}
	if err := m.Transition(ctx, store, 1, cancelled); err != nil {
		t.Fatal(err)
	}
	want := tsukuyomi.Transition[int, status]{ID: 1, From: newOrder, To: cancelled}
	if len(entered) != 1 || entered[0] != want {
		t.Fatalf("hook calls = %v, want [%v]", entered, want)
	}
}

func TestGuardOnUnknownTransitionPanics(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Fatal("expected panic")
		}
	}()
	orders().Guard(shipped, newOrder, func(context.Context, tsukuyomi.Transition[int, status]) error { return nil })
}

// Many goroutines race to move the same order out of "new": exactly one may win.
func TestConcurrentTransitions(t *testing.T) {
	ctx := context.Background()
	store := tsukuyomi.NewMemoryStore[int, status]()
	store.Set(1, newOrder)
	m := orders()

	var wins atomic.Int32
	var wg sync.WaitGroup
	for i := range 100 {
		target := paid
		if i%2 == 0 {
			target = cancelled
		}
		wg.Go(func() {
			err := m.Transition(ctx, store, 1, target)
			switch {
			case err == nil:
				wins.Add(1)
			case errors.Is(err, tsukuyomi.ErrConflict), errors.Is(err, tsukuyomi.ErrInvalidTransition):
			default:
				t.Error(err)
			}
		})
	}
	wg.Wait()

	if wins.Load() != 1 {
		t.Fatalf("wins = %d, want 1", wins.Load())
	}
}

func TestMermaid(t *testing.T) {
	want := "stateDiagram-v2\n    new --> paid\n    new --> cancelled\n    paid --> shipped\n"
	if got := orders().Mermaid(); got != want {
		t.Fatalf("got:\n%s\nwant:\n%s", got, want)
	}
}
