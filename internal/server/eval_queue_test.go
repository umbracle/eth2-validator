package server

import (
	"fmt"
	"testing"
	"time"

	"github.com/umbracle/eth2-validator/internal/server/proto"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func TestEvalQueue_X(t *testing.T) {
	e := NewEvalQueue()
	e.Start()

	e.Enqueue([]*proto.Duty{
		{
			Id:         "a",
			ActiveTime: timestamppb.New(time.Now().Add(1 * time.Second)),
			BlockedBy:  []string{"b", "c"},
		},
	})

	e.Enqueue([]*proto.Duty{
		{
			Id:         "b",
			ActiveTime: timestamppb.New(time.Now().Add(1 * time.Second)),
		},
	})

	e.Enqueue([]*proto.Duty{
		{
			Id:         "c",
			ActiveTime: timestamppb.New(time.Now().Add(1 * time.Second)),
		},
	})

	fmt.Println(e.Dequeue())

	e.Ack("b")

	fmt.Println(e.Dequeue())

	e.Ack("c")

	fmt.Println(e.Dequeue())

}
