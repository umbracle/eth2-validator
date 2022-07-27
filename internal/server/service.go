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
	iter, err := s.srv.state.DutiesList(ws)
	if err != nil {
		return nil, err
	}

	duties := []*proto.Duty{}
	for obj := iter.Next(); obj != nil; obj = iter.Next() {
		elem := obj.(*proto.Duty)
		if req.ValidatorId >= 0 && elem.ValidatorIndex != uint64(req.ValidatorId) {
			continue
		}
		duties = append(duties, elem)
	}

	resp := &proto.ListDutiesResponse{
		Duties: duties,
	}
	return resp, nil
}

func (s *service) ValidatorList(ctx context.Context, req *proto.ValidatorListRequest) (*proto.ValidatorListResponse, error) {
	ws := memdb.NewWatchSet()
	iter, err := s.srv.state.ValidatorsList(ws)
	if err != nil {
		return nil, err
	}

	validators := []*proto.Validator{}
	for obj := iter.Next(); obj != nil; obj = iter.Next() {
		elem := obj.(*proto.Validator)
		validators = append(validators, elem)
	}

	resp := &proto.ValidatorListResponse{
		Validators: validators,
	}
	return resp, nil
}
