package server

import (
	"time"

	"github.com/hashicorp/go-hclog"
)

type beaconTracker struct {
	logger         hclog.Logger
	genesisTime    time.Time
	secondsPerSlot uint64
	slotsPerEpoch  uint64
	resCh          chan SlotResult
	closeCh        chan struct{}
	readyCh        chan struct{}
	startEpoch     uint64
}

func newBeaconTracker(logger hclog.Logger, genesisTime time.Time, secondsPerSlot, slotsPerEpoch uint64) *beaconTracker {
	return &beaconTracker{
		logger:         logger.Named("tracker"),
		genesisTime:    genesisTime,
		secondsPerSlot: secondsPerSlot,
		slotsPerEpoch:  slotsPerEpoch,
		readyCh:        make(chan struct{}),
		closeCh:        make(chan struct{}),
		resCh:          make(chan SlotResult),
	}
}

type SlotResult struct {
	Epoch          uint64
	StartTime      time.Time
	GenesisTime    time.Time
	SecondsPerSlot uint64

	FirstSlot uint64
	LastSlot  uint64
}

func (s *SlotResult) AtSlot(slot uint64) time.Time {
	return s.GenesisTime.Add(time.Duration(slot*s.SecondsPerSlot) * time.Second)
}

func (s *SlotResult) AtSlotAndStage(slot uint64) time.Time {
	return s.GenesisTime.Add(time.Duration(slot*s.SecondsPerSlot) * time.Second)
}

func (b *beaconTracker) run() {
	secondsPerEpoch := time.Duration(b.secondsPerSlot*b.slotsPerEpoch) * time.Second

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

	nextTick := timeSinceGenesis.Truncate(secondsPerEpoch) + secondsPerEpoch
	epoch := uint64(nextTick / secondsPerEpoch)
	nextTickTime := b.genesisTime.Add(nextTick)

	b.startEpoch = epoch

	// close the ready channel to notify that the
	// chain has started
	close(b.readyCh)

	emitEpoch := func(epoch uint64) {
		startTime := b.genesisTime.Add(time.Duration(epoch*b.slotsPerEpoch*b.secondsPerSlot) * time.Second)

		firstSlot := epoch * b.slotsPerEpoch
		lastSlot := epoch*b.slotsPerEpoch + b.slotsPerEpoch

		b.resCh <- SlotResult{
			Epoch:          epoch,
			StartTime:      startTime,
			SecondsPerSlot: b.secondsPerSlot,
			GenesisTime:    b.genesisTime,
			FirstSlot:      firstSlot,
			LastSlot:       lastSlot,
		}
	}

	emitEpoch(epoch - 1)

	for {
		timeToWait := nextTickTime.Sub(now())
		b.logger.Trace("time to wait", "duration", timeToWait)

		select {
		case <-time.After(timeToWait):
		case <-b.closeCh:
			return
		}

		emitEpoch(epoch)
		epoch++
		nextTickTime = nextTickTime.Add(secondsPerEpoch)
	}
}

var now = time.Now
