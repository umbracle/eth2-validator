package routine

import (
	"context"
	"testing"

	"github.com/hashicorp/go-hclog"
)

func TestManager(t *testing.T) {
	r0 := newMockRoutine()
	r1 := newMockRoutine()

	m := NewManager(hclog.NewNullLogger())
	m.Add("r0", r0.Run)
	m.Add("r1", r1.Run)

	m.Start(context.Background())

	// both channels have started
	<-r0.startCh
	<-r1.startCh

	m.Stop()

	// both channels have stopped
	<-r0.closeCh
	<-r1.closeCh
}

type mockRoutine struct {
	startCh chan struct{}
	closeCh chan struct{}
}

func newMockRoutine() *mockRoutine {
	m := &mockRoutine{
		startCh: make(chan struct{}),
		closeCh: make(chan struct{}),
	}
	return m
}

func (m *mockRoutine) Run(ctx context.Context) error {
	close(m.startCh)
	<-ctx.Done()
	close(m.closeCh)

	return nil
}
