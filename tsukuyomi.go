package tsukuyomi

import (
	"context"
	"sync"
)

type Tsukuyomi[ID, K, T comparable] struct {
	storage     Storage[ID, K, T]
	ruleManager *RuleManager[K]

	states map[K]State[K, T]
	lock   sync.RWMutex
}

func new[ID, K, T comparable](storage Storage[ID, K, T]) *Tsukuyomi[ID, K, T] {
	return &Tsukuyomi[ID, K, T]{
		storage:     storage,
		ruleManager: NewRuleManager[K](),
		states:      make(map[K]State[K, T]),
		lock:        sync.RWMutex{},
	}
}

func (t *Tsukuyomi[ID, K, T]) RegisterState(state State[K, T]) error {
	t.lock.Lock()
	defer t.lock.Unlock()

	t.states[state.Key] = state

	return nil
}

func (t *Tsukuyomi[ID, K, T]) HasState(key K) bool {
	t.lock.RLock()
	defer t.lock.RUnlock()

	_, ok := t.states[key]
	return ok
}

func (t *Tsukuyomi[ID, K, T]) BanState(ctx context.Context, key K) error {
	t.lock.Lock()
	defer t.lock.Unlock()

	_, ok := t.states[key]
	if !ok {
		return ErrStateNotFound
	}

	t.ruleManager.RemoveKey(key)

	delete(t.states, key)

	return nil
}

func (t *Tsukuyomi[ID, K, T]) GetState(ctx context.Context, id ID) (State[K, T], error) {
	return t.storage.GetState(ctx, id)
}

func (t *Tsukuyomi[ID, K, T]) SaveState(ctx context.Context, id ID, state State[K, T]) error {
	return t.storage.SaveState(ctx, id, state)
}

func (t *Tsukuyomi[ID, K, T]) DeleteState(ctx context.Context, id ID) error {
	return t.storage.DeleteState(ctx, id)
}

// EnableTransitionByKeys включает разрешенный переход между ключами состояний
func (t *Tsukuyomi[ID, K, T]) EnableTransition(from, to K) error {
	if !t.isRegisteredKeys(from, to) {
		return ErrStateNotFound
	}

	t.ruleManager.EnableTransition(from, to)

	return nil
}

func (t *Tsukuyomi[ID, K, T]) isRegisteredKeys(keys ...K) bool {
	t.lock.RLock()
	defer t.lock.RUnlock()

	for _, key := range keys {
		if _, ok := t.states[key]; !ok {
			return false
		}
	}
	return true
}

func (t *Tsukuyomi[ID, K, T]) DisableTransition(from, to K) error {
	if !t.isRegisteredKeys(from, to) {
		return ErrStateNotFound
	}

	t.ruleManager.DisableTransition(from, to)

	return nil
}

func (t *Tsukuyomi[ID, K, T]) ProcessTransition(ctx context.Context, id ID, to K) error {
	state, err := t.GetState(ctx, id)
	if err != nil {
		return err
	}

	t.lock.RLock()

	newState, ok := t.states[to]
	if !ok {
		return ErrStateNotFound
	}

	t.lock.RUnlock()

	if !t.ruleManager.CanTransition(state.Key, to) {
		return ErrInvalidTransition
	}

	err = t.storage.SaveState(ctx, id, newState)
	if err != nil {
		return err
	}

	return nil
}
