package tsukuyomi

type State[K comparable, T comparable] struct {
	Key         K
	Type        T
	Name        string
	Description string
}

func NewState[K comparable, T comparable](key K, stateType T, name string, description string) State[K, T] {
	return State[K, T]{
		Key:         key,
		Type:        stateType,
		Name:        name,
		Description: description,
	}
}

