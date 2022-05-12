package server

import (
	"testing"
	"time"
)

func restoreHooks() func() {
	n := now
	return func() {
		now = n
	}
}

func TestBeaconTracker_Genesis(t *testing.T) {
	defer restoreHooks()()

	now = func() time.Time { return time.Unix(33, 0) }

	b := &beaconTracker{
		genesisTime:    time.Unix(10, 0),
		secondsPerSlot: 5,
	}

	b.run()
}
