package server

import (
	"context"

	"github.com/hashicorp/go-memdb"
	"github.com/umbracle/eth2-validator/internal/server/proto"
)

type service struct {
	proto.UnimplementedValidatorServiceServer

	srv *Server
}

func (s *service) ListDuties(ctx context.Context, req *proto.ListDutiesRequest) (*proto.ListDutiesResponse, error) {
	ws := memdb.NewWatchSet()
	iter, err := s.srv.state.JobsList(ws)
	if err != nil {
		return nil, err
	}

	duties := []*proto.Duty{}
	for obj := iter.Next(); obj != nil; obj = iter.Next() {
		elem := obj.(*proto.Duty)
		duties = append(duties, elem)
	}

	resp := &proto.ListDutiesResponse{
		Duties: duties,
	}
	return resp, nil
}
