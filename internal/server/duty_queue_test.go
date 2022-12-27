package server

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/umbracle/eth2-validator/internal/server/proto"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func TestDutyEvalQueue_Dequeue(t *testing.T) {
	ctx := context.Background()

	e := NewEvalQueue()
	e.Start()

	expectedTimes := map[string]time.Time{}

	// enqueue n duties with 1 second from each other
	num := 5
	for i := 0; i < num; i++ {
		id := fmt.Sprintf("id%d", i)
		tt := time.Now().Add(1 * time.Second).Add(time.Duration(i) * time.Second)

		expectedTimes[id] = tt

		e.Enqueue(ctx, []*proto.Duty{
			{
				Id:         fmt.Sprintf("id%d", i),
				ActiveTime: timestamppb.New(tt),
			},
		})
	}

	for i := 0; i < num; i++ {
		duty, _, err := e.Dequeue()
		require.NoError(t, err)

		// it should not have passed more than one second from the expected time
		timeElapsed := time.Now().Sub(expectedTimes[duty.Id]).Seconds()
		require.Less(t, timeElapsed, float64(1))
	}
}

func TestDutyEvalQueue_Blocked(t *testing.T) {
	ctx := context.Background()

	e := NewEvalQueue()
	e.Start()

	duties := []*proto.Duty{
		{
			Id:         "a",
			ActiveTime: timestamppb.New(time.Now().Add(5 * time.Second)),
			BlockedBy:  []string{"b", "c"},
		},
		{
			Id:         "b",
			ActiveTime: timestamppb.New(time.Now().Add(1 * time.Second)),
		},
		{
			Id:         "c",
			ActiveTime: timestamppb.New(time.Now().Add(1 * time.Second)),
		},
	}
	for _, duty := range duties {
		e.Enqueue(ctx, []*proto.Duty{duty})
	}

	order := []string{
		// dequeue and ack "b" and "c"
		"b", "c",
		// "a" is unblocked
		"a",
	}
	for _, o := range order {
		duty, _, err := e.Dequeue()
		require.NoError(t, err)
		require.Equal(t, duty.Id, o)
		require.NoError(t, e.Ack(o))
	}
}
