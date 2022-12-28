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

type syncBeaconCommitteeSubscriptionMockAPI struct {
	*beacon.NullAPI

	subsCh chan []*http.BeaconCommitteeSubscription
}

func newSyncBeaconCommitteeSubscriptionMockAPI() *syncBeaconCommitteeSubscriptionMockAPI {
	return &syncBeaconCommitteeSubscriptionMockAPI{
		subsCh: make(chan []*http.BeaconCommitteeSubscription),
	}
}

func (s *syncBeaconCommitteeSubscriptionMockAPI) BeaconCommitteeSubscriptions(ctx context.Context, subs []*http.BeaconCommitteeSubscription) error {
	s.subsCh <- subs
	return nil
}

func TestSyncBeaconCommitteeSubscription(t *testing.T) {
	state := state.NewInmemState(t)
	api := newSyncBeaconCommitteeSubscriptionMockAPI()

	duty0 := proto.NewDuty().
		WithID("a").
		WithValIndex(1).
		WithAttestation(&proto.Duty_Attestation{})

	require.NoError(t, state.UpsertDuty(duty0))

	s := NewSyncBeaconCommitteeSubscription(state, api)
	go s.Run()

	subs := <-api.subsCh
	require.Len(t, subs, 1)
	require.Equal(t, subs[0], &http.BeaconCommitteeSubscription{
		ValidatorIndex: 1,
	})
}
