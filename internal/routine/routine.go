package routine

import (
	"context"

	"github.com/hashicorp/go-hclog"
)

type Routine func(ctx context.Context) error

type routineTracker struct {
	stoppedCh chan struct{}
}

func (r *routineTracker) wait() {
	<-r.stoppedCh
}

type Manager struct {
	logger    hclog.Logger
	instances map[string]*routineTracker
	routines  map[string]Routine
	cancelFn  context.CancelFunc
}

func NewManager(logger hclog.Logger) *Manager {
	m := &Manager{
		logger:    logger.Named("routine-manager"),
		routines:  map[string]Routine{},
		instances: map[string]*routineTracker{},
	}
	return m
}

func (m *Manager) Add(name string, routine Routine) {
	m.routines[name] = routine
}

func (m *Manager) Start(ctx context.Context) {
	rtCtx, cancel := context.WithCancel(ctx)
	m.cancelFn = cancel

	for name, routine := range m.routines {
		instance := &routineTracker{
			stoppedCh: make(chan struct{}),
		}

		go m.execute(rtCtx, name, routine, instance.stoppedCh)
		m.instances[name] = instance

		m.logger.Debug("started routine", "routine", name)
	}
}

func (m *Manager) execute(ctx context.Context, name string, routine Routine, done chan struct{}) {
	defer func() {
		close(done)
	}()

	err := routine(ctx)
	if err != nil && err != context.DeadlineExceeded && err != context.Canceled {
		m.logger.Error("routine exited with error",
			"routine", name,
			"error", err,
		)
	} else {
		m.logger.Info("stopped routine", "routine", name)
	}
}

func (m *Manager) Stop() {
	m.cancelFn()

	for _, instance := range m.instances {
		instance.wait()
	}

	m.instances = map[string]*routineTracker{}
}
