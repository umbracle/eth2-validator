package server

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/umbracle/eth2-validator/internal/beacon"
	"github.com/umbracle/eth2-validator/internal/server/proto"
	"github.com/umbracle/eth2-validator/internal/server/state"
	"github.com/umbracle/go-eth-consensus/http"
)

type syncCommitteeSubscriptionMockAPI struct {
	*beacon.NullAPI

	subsCh chan []*http.SyncCommitteeSubscription
}

func newSyncCommitteeSubscriptionMockAPI() *syncCommitteeSubscriptionMockAPI {
	return &syncCommitteeSubscriptionMockAPI{
		subsCh: make(chan []*http.SyncCommitteeSubscription),
	}
}

func (s *syncCommitteeSubscriptionMockAPI) SyncCommitteeSubscriptions(ctx context.Context, subs []*http.SyncCommitteeSubscription) error {
	s.subsCh <- subs
	return nil
}

func TestSyncCommitteeSubscription_MultipleCommittees(t *testing.T) {
	state := state.NewInmemState(t)
	api := newSyncCommitteeSubscriptionMockAPI()

	duty0 := proto.NewDuty().
		WithID("a").
		WithValIndex(1).
		WithSyncCommitteeAggregate(&proto.Duty_SyncCommitteeAggregate{
			SubCommitteeIndex: 10,
		})

	duty1 := proto.NewDuty().
		WithID("b").
		WithValIndex(1).
		WithSyncCommitteeAggregate(&proto.Duty_SyncCommitteeAggregate{
			SubCommitteeIndex: 20,
		})

	require.NoError(t, state.UpsertDuty(duty0))
	require.NoError(t, state.UpsertDuty(duty1))

	s := NewSyncCommitteeSubscription(state, api)
	go s.Run()

	subs := <-api.subsCh
	require.Len(t, subs, 1)
	require.Equal(t, subs[0], &http.SyncCommitteeSubscription{
		ValidatorIndex:       1,
		UntilEpoch:           1,
		SyncCommitteeIndices: []uint64{10, 20},
	})
}

func TestSyncCommitteeSubscription_DifferentValidators(t *testing.T) {
	state := state.NewInmemState(t)
	api := newSyncCommitteeSubscriptionMockAPI()

	duty0 := proto.NewDuty().
		WithID("a").
		WithValIndex(1).
		WithSyncCommitteeAggregate(&proto.Duty_SyncCommitteeAggregate{
			SubCommitteeIndex: 10,
		})

	duty1 := proto.NewDuty().
		WithID("b").
		WithValIndex(2).
		WithSyncCommitteeAggregate(&proto.Duty_SyncCommitteeAggregate{
			SubCommitteeIndex: 20,
		})

	require.NoError(t, state.UpsertDuty(duty0))
	require.NoError(t, state.UpsertDuty(duty1))

	s := NewSyncCommitteeSubscription(state, api)
	go s.Run()

	subs := <-api.subsCh
	require.Len(t, subs, 2)

	require.Equal(t, subs[0], &http.SyncCommitteeSubscription{
		ValidatorIndex:       1,
		UntilEpoch:           1,
		SyncCommitteeIndices: []uint64{10},
	})

	require.Equal(t, subs[1], &http.SyncCommitteeSubscription{
		ValidatorIndex:       2,
		UntilEpoch:           1,
		SyncCommitteeIndices: []uint64{20},
	})
}
