package memory

import (
	"context"
	"sync"

	"github.com/sovamorco/errorx"
	"github.com/sovamorco/gommon/locker"
)

type Mock struct {
	locks sync.Map `exhaustruct:"optional"`
}

func New() *Mock {
	return &Mock{}
}

// required by interface.
//
//nolint:ireturn
func (m *Mock) Lock(_ context.Context, name string) (locker.Lock, error) {
	l := &Lock{
		name:   name,
		parent: m,
	}

	_, exists := m.locks.LoadOrStore(name, l)
	if exists {
		return nil, errorx.IllegalState.New("lock %s already exists", name)
	}

	return l, nil
}

type Lock struct {
	name   string
	parent *Mock
}

func (l *Lock) Unlock(_ context.Context) error {
	l.parent.locks.Delete(l.name)

	return nil
}
