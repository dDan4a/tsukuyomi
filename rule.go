package tsukuyomi

import (
	"sync"

	"github.com/samber/lo"
)

type StateRule[K comparable] struct {
	Key  K
	From map[K]struct{}
	To   map[K]struct{}
	lock sync.RWMutex
}

func NewStateRule[K comparable]() *StateRule[K] {
	return &StateRule[K]{
		From: make(map[K]struct{}),
		To:   make(map[K]struct{}),
		lock: sync.RWMutex{},
	}
}

func (sr *StateRule[K]) AddTo(to K) {
	sr.lock.Lock()
	defer sr.lock.Unlock()

	sr.To[to] = struct{}{}
}

func (sr *StateRule[K]) AddFrom(from K) {
	sr.lock.Lock()
	defer sr.lock.Unlock()

	sr.From[from] = struct{}{}
}

func (sr *StateRule[K]) RemoveTo(to K) {
	sr.lock.Lock()
	defer sr.lock.Unlock()

	delete(sr.To, to)
}

func (sr *StateRule[K]) RemoveFrom(from K) {
	sr.lock.Lock()
	defer sr.lock.Unlock()

	delete(sr.From, from)
}

func (sr *StateRule[K]) ListTo(key K) []K {
	sr.lock.RLock()
	defer sr.lock.RUnlock()

	return lo.Keys(sr.To)
}

func (sr *StateRule[K]) ListFrom(key K) []K {
	sr.lock.RLock()
	defer sr.lock.RUnlock()

	return lo.Keys(sr.From)
}

func (sr *StateRule[K]) CanToKey(to K) bool {
	sr.lock.RLock()
	defer sr.lock.RUnlock()

	_, ok := sr.To[to]

	return ok
}

func (sr *StateRule[K]) CanFromKey(from K) bool {
	sr.lock.RLock()
	defer sr.lock.RUnlock()

	_, ok := sr.From[from]
	return ok
}

// RuleManager управляет правилами переходов
type RuleManager[K comparable] struct {
	Rules map[K]*StateRule[K]
	lock  sync.RWMutex
}

func NewRuleManager[K comparable]() *RuleManager[K] {
	return &RuleManager[K]{
		Rules: make(map[K]*StateRule[K]),
		lock:  sync.RWMutex{},
	}
}

func (rm *RuleManager[K]) EnableTransition(from, to K) {
	rm.lock.Lock()
	defer rm.lock.Unlock()

	stateRule, ok := rm.Rules[from]
	if !ok {
		stateRule = NewStateRule[K]()
		stateRule.AddTo(to)
		rm.Rules[from] = stateRule
	} else {
		stateRule.AddTo(to)
	}

	stateRule, ok = rm.Rules[to]
	if !ok {
		stateRule = NewStateRule[K]()
		stateRule.AddFrom(from)
		rm.Rules[to] = stateRule
	} else {
		stateRule.AddFrom(from)
	}
}

func (rm *RuleManager[K]) DisableTransition(from, to K) {
	rm.lock.Lock()
	defer rm.lock.Unlock()

	stateRule, ok := rm.Rules[from]
	if !ok {
		return
	}

	stateRule.RemoveTo(to)

	stateRule, ok = rm.Rules[to]
	if !ok {
		return
	}

	stateRule.RemoveFrom(from)
}

func (rm *RuleManager[K]) CanTransition(from, to K) bool {
	rm.lock.RLock()
	defer rm.lock.RUnlock()

	fromRule, ok := rm.Rules[from]
	if !ok {
		return false
	}

	return fromRule.CanToKey(to)
}

func (rm *RuleManager[K]) RemoveKey(key K) {
	rm.lock.Lock()
	defer rm.lock.Unlock()

	stateRule, ok := rm.Rules[key]
	if !ok {
		return
	}

	for _, from := range stateRule.ListFrom(key) {
		rm.Rules[from].RemoveTo(key)
	}

	for _, to := range stateRule.ListTo(key) {
		rm.Rules[to].RemoveFrom(key)
	}

	delete(rm.Rules, key)
}
