package server

import (
	"context"

	"github.com/hashicorp/go-hclog"
	"github.com/hashicorp/go-memdb"
	"github.com/umbracle/eth2-validator/internal/beacon"
	"github.com/umbracle/eth2-validator/internal/server/proto"
	"github.com/umbracle/eth2-validator/internal/server/state"
	"github.com/umbracle/go-eth-consensus/http"
)

type SyncBeaconCommitteeSubscription struct {
	logger  hclog.Logger
	state   *state.State
	api     beacon.Api
	closeCh chan struct{}
}

func NewSyncBeaconCommitteeSubscription(state *state.State, api beacon.Api) *SyncBeaconCommitteeSubscription {
	s := &SyncBeaconCommitteeSubscription{
		logger:  hclog.NewNullLogger(),
		state:   state,
		api:     api,
		closeCh: make(chan struct{}),
	}

	return s
}

func (s *SyncBeaconCommitteeSubscription) Close() {
	close(s.closeCh)
}

func (s *SyncBeaconCommitteeSubscription) SetLogger(logger hclog.Logger) {
	s.logger = logger.Named("sync-beacon-committee-subscription")
}

func (s *SyncBeaconCommitteeSubscription) Run() {
	for {
		ws := memdb.NewWatchSet()
		iter, err := s.state.DutiesList(ws)
		if err != nil {
			panic(err)
		}

		subs := []*http.BeaconCommitteeSubscription{}

		for obj := iter.Next(); obj != nil; obj = iter.Next() {
			duty := obj.(*proto.Duty)

			// only attestation
			if duty.Type() != proto.DutyAttestation {
				continue
			}

			attestation := duty.GetAttestation()

			sub := &http.BeaconCommitteeSubscription{
				ValidatorIndex:   duty.ValidatorIndex,
				Slot:             duty.Slot,
				CommitteeIndex:   attestation.CommitteeIndex,
				CommitteesAtSlot: attestation.CommitteesAtSlot,
			}
			subs = append(subs, sub)
		}

		if len(subs) != 0 {
			if err := s.api.BeaconCommitteeSubscriptions(context.Background(), subs); err != nil {
				s.logger.Error("failed to send beacon committe subscriptions", "err", err)
			}
		}

		// wait for the duties to change
		select {
		case <-ws.WatchCh(context.Background()):
		case <-s.closeCh:
			return
		}
	}
}
