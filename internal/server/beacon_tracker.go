package server

import (
	"time"

	"github.com/hashicorp/go-hclog"
)

type beaconTracker struct {
	logger         hclog.Logger
	genesisTime    time.Time
	secondsPerSlot uint64
	resCh          chan SlotResult
	closeCh        chan struct{}
	readyCh        chan struct{}
	startSlot      uint64
}

func newBeaconTracker(logger hclog.Logger, genesisTime time.Time, secondsPerSlot uint64) *beaconTracker {
	return &beaconTracker{
		logger:         logger.Named("tracker"),
		genesisTime:    genesisTime,
		secondsPerSlot: secondsPerSlot,
		readyCh:        make(chan struct{}),
		closeCh:        make(chan struct{}),
		resCh:          make(chan SlotResult),
	}
}

type SlotResult struct {
	Slot uint64
}

func (b *beaconTracker) run() {
	secondsPerSlot := time.Duration(b.secondsPerSlot) * time.Second

	// time since genesis
	currentTime := now()
	timeSinceGenesis := currentTime.Sub(b.genesisTime)

	if timeSinceGenesis < 0 {
		// wait until the chain has started
		timeUntilGenesis := b.genesisTime.Sub(currentTime)
		select {
		case <-time.After(timeUntilGenesis):
			timeSinceGenesis = 0

		case <-b.closeCh:
			return
		}
	}

	nextTick := timeSinceGenesis.Truncate(secondsPerSlot) + secondsPerSlot
	slot := uint64(nextTick / secondsPerSlot)
	nextTickTime := b.genesisTime.Add(nextTick)

	b.startSlot = slot

	// close the ready channel to notify that the
	// chain has started
	close(b.readyCh)

	for {
		timeToWait := nextTickTime.Sub(now())
		b.logger.Trace("time to wait", "duration", timeToWait)

		select {
		case <-time.After(timeToWait):
		case <-b.closeCh:
			return
		}
		b.resCh <- SlotResult{
			Slot: slot,
		}

		slot++
		nextTickTime = nextTickTime.Add(secondsPerSlot)
	}
}

var now = time.Now
