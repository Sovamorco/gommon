package memory

import (
	"context"
	"sync"

	"github.com/sovamorco/errorx"
	"github.com/sovamorco/gommon/locker"
)

//nolint:gochecknoinits // driver pattern.
func init() {
	locker.Register("mock", newMock)
}

type Memory struct {
	locks sync.Map `exhaustruct:"optional"`
}

//nolint:ireturn // required by locker.Register.
func newMock(_ context.Context, _ string) (locker.Locker, error) {
	return &Memory{}, nil
}

// required by interface.
//
//nolint:ireturn
func (m *Memory) Lock(_ context.Context, name string) (locker.Lock, error) {
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
	parent *Memory
}

func (l *Lock) Unlock(_ context.Context) error {
	l.parent.locks.Delete(l.name)

	return nil
}
