// Package tsukuyomi is a state machine for status fields of domain entities (orders, payments, requests, ...).
//
// The machine describes which transitions are allowed; a Store applies them atomically,
// so two concurrent transitions of the same entity cannot both succeed.
package tsukuyomi

import (
	"context"
	"errors"
	"fmt"
	"strings"
)

var (
	// ErrNotFound is returned by a Store when the entity does not exist.
	ErrNotFound = errors.New("tsukuyomi: entity not found")
	// ErrInvalidTransition means the machine has no transition from the current state to the target one.
	ErrInvalidTransition = errors.New("tsukuyomi: invalid transition")
	// ErrConflict means the state was changed concurrently between the read and the write.
	ErrConflict = errors.New("tsukuyomi: state changed concurrently")
)

// Store keeps the current state of entities.
type Store[ID, S comparable] interface {
	State(ctx context.Context, id ID) (S, error)
	// CompareAndSwap sets the state to `to` only if it currently equals `from` and reports whether it did.
	CompareAndSwap(ctx context.Context, id ID, from, to S) (bool, error)
}

// Transition describes a single state change of an entity.
type Transition[ID, S comparable] struct {
	ID       ID
	From, To S
}

// Guard can veto a transition by returning an error.
type Guard[ID, S comparable] func(ctx context.Context, t Transition[ID, S]) error

// Hook runs after a transition has been committed.
type Hook[ID, S comparable] func(ctx context.Context, t Transition[ID, S])

type edge[S comparable] struct{ from, to S }

// Machine holds the transition graph. Configure it at startup; after that it is safe for concurrent use.
type Machine[ID, S comparable] struct {
	edges   map[edge[S]][]Guard[ID, S]
	order   []edge[S]
	onEnter map[S][]Hook[ID, S]
}

func New[ID, S comparable]() *Machine[ID, S] {
	return &Machine[ID, S]{
		edges:   make(map[edge[S]][]Guard[ID, S]),
		onEnter: make(map[S][]Hook[ID, S]),
	}
}

// Allow permits transitions from `from` to each of `to`.
func (m *Machine[ID, S]) Allow(from S, to ...S) *Machine[ID, S] {
	for _, t := range to {
		e := edge[S]{from, t}
		if _, ok := m.edges[e]; !ok {
			m.edges[e] = nil
			m.order = append(m.order, e)
		}
	}
	return m
}

// Guard adds a check to an allowed transition. It panics if the transition was not allowed: that is a setup bug.
func (m *Machine[ID, S]) Guard(from, to S, g Guard[ID, S]) *Machine[ID, S] {
	e := edge[S]{from, to}
	if _, ok := m.edges[e]; !ok {
		panic(fmt.Sprintf("tsukuyomi: guard on transition %v -> %v that is not allowed", from, to))
	}
	m.edges[e] = append(m.edges[e], g)
	return m
}

// OnEnter adds a hook that runs after an entity enters state s.
func (m *Machine[ID, S]) OnEnter(s S, h Hook[ID, S]) *Machine[ID, S] {
	m.onEnter[s] = append(m.onEnter[s], h)
	return m
}

func (m *Machine[ID, S]) Can(from, to S) bool {
	_, ok := m.edges[edge[S]{from, to}]
	return ok
}

// Transition moves the entity to state `to`, checking the graph and guards, and runs OnEnter hooks on success.
func (m *Machine[ID, S]) Transition(ctx context.Context, store Store[ID, S], id ID, to S) error {
	from, err := store.State(ctx, id)
	if err != nil {
		return err
	}

	guards, ok := m.edges[edge[S]{from, to}]
	if !ok {
		return fmt.Errorf("%w: %v -> %v", ErrInvalidTransition, from, to)
	}

	t := Transition[ID, S]{ID: id, From: from, To: to}
	for _, g := range guards {
		if err := g(ctx, t); err != nil {
			return err
		}
	}

	swapped, err := store.CompareAndSwap(ctx, id, from, to)
	if err != nil {
		return err
	}
	if !swapped {
		return fmt.Errorf("%w: expected %v", ErrConflict, from)
	}

	for _, h := range m.onEnter[to] {
		h(ctx, t)
	}
	return nil
}

// Mermaid renders the transition graph as a Mermaid state diagram.
func (m *Machine[ID, S]) Mermaid() string {
	var b strings.Builder
	b.WriteString("stateDiagram-v2\n")
	for _, e := range m.order {
		fmt.Fprintf(&b, "    %v --> %v\n", e.from, e.to)
	}
	return b.String()
}
