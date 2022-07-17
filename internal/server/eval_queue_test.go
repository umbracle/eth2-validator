package server

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/umbracle/eth2-validator/internal/server/proto"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func TestEvalQueue_X(t *testing.T) {
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

	fmt.Println(e.Dequeue())

	e.Ack("b")

	fmt.Println(e.Dequeue())

	e.Ack("c")

	fmt.Println(e.Dequeue())

}
