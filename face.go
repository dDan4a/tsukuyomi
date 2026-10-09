package tsukuyomi

import "context"

// Storage абстракция над хранилищем состояний
// ID — идентификатор сущности, K — ключ состояния, T — тип состояния
type Storage[ID, K, T comparable] interface {
	GetState(ctx context.Context, id ID) (State[K, T], error)
	SaveState(ctx context.Context, id ID, state State[K, T]) error
	DeleteState(ctx context.Context, id ID) error
}
