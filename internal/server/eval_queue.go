package server

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/umbracle/eth2-validator/internal/delayheap"
	"github.com/umbracle/eth2-validator/internal/server/proto"
)

type EvalQueue struct {
	l             sync.RWMutex
	delayHeap     *delayheap.DelayHeap
	delayUpdateCh chan struct{}
	closeCh       chan struct{}

	// unack is the list of not aknowledge duties
	unack map[string]*proto.Duty

	// map of duties to trace contexts
	ctxMap map[string]context.Context

	// blocked tracks the blocked duties
	blocked map[string]*blockedDuty

	reverseBlockedMap map[string][]string

	// ready tracks the duties ready to be processed
	ready []*proto.Duty

	// updateCh notifies whenever there are new ready duties
	updateCh chan struct{}
}

type blockedDuty struct {
	Duty    *proto.Duty
	Blocked map[string]struct{}
}

func NewEvalQueue() *EvalQueue {
	e := &EvalQueue{
		delayHeap:         delayheap.NewDelayHeap(),
		unack:             map[string]*proto.Duty{},
		delayUpdateCh:     make(chan struct{}),
		closeCh:           make(chan struct{}),
		blocked:           map[string]*blockedDuty{},
		ready:             []*proto.Duty{},
		updateCh:          make(chan struct{}),
		reverseBlockedMap: map[string][]string{},
		ctxMap:            map[string]context.Context{},
	}
	return e
}

func (p *EvalQueue) Start() {
	go p.runDelayHeap()
}

type dutyWrapper struct {
	eval *proto.Duty
}

func (d *dutyWrapper) ID() string {
	return d.eval.Id
}

func (p *EvalQueue) Enqueue(ctx context.Context, duties []*proto.Duty) {
	p.l.Lock()
	defer p.l.Unlock()

	if len(duties) == 0 {
		return
	}
	for _, duty := range duties {
		if len(duty.BlockedBy) != 0 {
			// wait for other tasks to unlock
			blockedD := &blockedDuty{
				Duty:    duty,
				Blocked: map[string]struct{}{},
			}
			for _, elem := range duty.BlockedBy {
				blockedD.Blocked[elem] = struct{}{}
			}
			p.blocked[duty.Id] = blockedD

			for _, dep := range duty.BlockedBy {
				if _, ok := p.reverseBlockedMap[dep]; ok {
					p.reverseBlockedMap[dep] = []string{}
				}
				p.reverseBlockedMap[dep] = append(p.reverseBlockedMap[dep], duty.Id)
			}
		} else {
			// not blocked, push right away to the heap
			p.delayHeap.Push(&dutyWrapper{duty}, duty.ActiveTime.AsTime())
		}
		// add entry in the context map
		p.ctxMap[duty.Id] = ctx
	}

	select {
	case p.delayUpdateCh <- struct{}{}:
	default:
	}
}

func (p *EvalQueue) Dequeue() (*proto.Duty, context.Context, error) {
START:
	p.l.Lock()
	if len(p.ready) != 0 {
		// dequeue a duty
		var duty *proto.Duty
		duty, p.ready = p.ready[0], p.ready[1:]
		p.unack[duty.Id] = duty

		ctx, ok := p.ctxMap[duty.Id]
		if !ok {
			p.l.Unlock()
			return nil, nil, fmt.Errorf("context not found for task: %s", duty.Id)
		}

		p.l.Unlock()
		return duty, ctx, nil
	}

	p.l.Unlock()

	select {
	case <-p.updateCh:
		goto START
	case <-p.closeCh:
		return nil, nil, nil
	}
}

func (p *EvalQueue) Ack(dutyID string) error {
	p.l.Lock()
	defer p.l.Unlock()

	_, ok := p.unack[dutyID]
	if !ok {
		return fmt.Errorf("duty '%s' not found", dutyID)
	}
	delete(p.unack, dutyID)
	delete(p.ctxMap, dutyID)

	// unblock pending tasks
	blockedDuties, ok := p.reverseBlockedMap[dutyID]
	if ok {
		for _, duty := range blockedDuties {
			found, ok := p.blocked[duty]
			if !ok {
				return fmt.Errorf("duty to unblock '%s' not found", duty)
			}
			if _, ok := found.Blocked[dutyID]; !ok {
				return fmt.Errorf("duty is not a dependency '%s' of '%s'", dutyID, found.Duty.Id)
			}

			delete(found.Blocked, dutyID)
			if len(found.Blocked) == 0 {
				delete(p.blocked, duty)

				// enqueue the task in the delay heap
				p.delayHeap.Push(&dutyWrapper{found.Duty}, found.Duty.ActiveTime.AsTime())
				select {
				case p.delayUpdateCh <- struct{}{}:
				default:
				}
			}
		}
	}
	return nil
}

func (p *EvalQueue) enqueueLocked(duty *proto.Duty) {
	p.ready = append(p.ready, duty)

	select {
	case p.updateCh <- struct{}{}:
	default:
	}
}

// nextDelayedEval returns the next delayed eval to launch and when it should be enqueued.
func (p *EvalQueue) nextDelayedEval() (*proto.Duty, time.Time) {
	p.l.RLock()
	defer p.l.RUnlock()

	// If there is nothing wait for an update.
	if p.delayHeap.Length() == 0 {
		return nil, time.Time{}
	}
	nextEval := p.delayHeap.Peek()
	if nextEval == nil {
		return nil, time.Time{}
	}
	eval := nextEval.Node
	return eval.(*dutyWrapper).eval, nextEval.WaitUntil
}

func (p *EvalQueue) runDelayHeap() {
	var timerChannel <-chan time.Time
	var delayTimer *time.Timer
	for {
		duty, waitUntil := p.nextDelayedEval()
		if waitUntil.IsZero() {
			timerChannel = nil
		} else {
			launchDur := waitUntil.Sub(time.Now().UTC())
			if delayTimer == nil {
				delayTimer = time.NewTimer(launchDur)
			} else {
				delayTimer.Reset(launchDur)
			}
			timerChannel = delayTimer.C

		}

		select {
		case <-p.closeCh:
			return

		case <-timerChannel:
			// remove from the heap since we can enqueue it now
			p.l.Lock()
			p.delayHeap.Remove(&dutyWrapper{duty})
			p.enqueueLocked(duty)
			p.l.Unlock()

		case <-p.delayUpdateCh:
			continue
		}
	}
}
