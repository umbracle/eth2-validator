package server

import (
	"context"
	"sort"

	"github.com/hashicorp/go-hclog"
	"github.com/hashicorp/go-memdb"
	"github.com/umbracle/eth2-validator/internal/beacon"
	"github.com/umbracle/eth2-validator/internal/server/proto"
	"github.com/umbracle/eth2-validator/internal/server/state"
	"github.com/umbracle/go-eth-consensus/http"
)

// SyncCommitteeSubscription is the service that does the subscription
// for the sync committees
type SyncCommitteeSubscription struct {
	logger  hclog.Logger
	state   *state.State
	api     beacon.Api
	closeCh chan struct{}
}

func NewSyncCommitteeSubscription(state *state.State, api beacon.Api) *SyncCommitteeSubscription {
	s := &SyncCommitteeSubscription{
		logger:  hclog.NewNullLogger(),
		state:   state,
		api:     api,
		closeCh: make(chan struct{}),
	}

	return s
}

func (s *SyncCommitteeSubscription) Close() {
	close(s.closeCh)
}

func (s *SyncCommitteeSubscription) SetLogger(logger hclog.Logger) {
	s.logger = logger.Named("sync-committee-subscription")
}

func (s *SyncCommitteeSubscription) Run() {
	for {
		ws := memdb.NewWatchSet()
		iter, err := s.state.DutiesList(ws)
		if err != nil {
			panic(err)
		}

		subsByValidator := map[uint64]*http.SyncCommitteeSubscription{}

		for obj := iter.Next(); obj != nil; obj = iter.Next() {
			duty := obj.(*proto.Duty)
			if duty.Type() != proto.DutySyncCommitteeAggregate {
				continue
			}

			syncCommitteeDuty := duty.GetSyncCommitteeAggregate()

			sub, ok := subsByValidator[duty.ValidatorIndex]
			if !ok {
				sub = &http.SyncCommitteeSubscription{
					ValidatorIndex: duty.ValidatorIndex,
					UntilEpoch:     duty.Epoch + 1,
				}
				subsByValidator[duty.ValidatorIndex] = sub
			}

			sub.SyncCommitteeIndices = append(sub.SyncCommitteeIndices, syncCommitteeDuty.GetSubCommitteeIndex())
		}

		subs := []*http.SyncCommitteeSubscription{}
		for _, sub := range subsByValidator {
			subs = append(subs, sub)
		}

		// sort to get a determinisitc order in the array of subscriptions
		sort.Slice(subs, func(i, j int) bool {
			return subs[i].ValidatorIndex < subs[j].ValidatorIndex
		})

		if err := s.api.SyncCommitteeSubscriptions(context.Background(), subs); err != nil {
			s.logger.Error("failed to send sync committees subscriptions", "err", err)
		}

		// wait for the duties to change
		select {
		case <-ws.WatchCh(context.Background()):
		case <-s.closeCh:
			return
		}
	}
}
