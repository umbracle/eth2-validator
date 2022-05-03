package server

import (
	"context"
	"encoding/hex"
	"fmt"
	"net"
	"time"

	ssz "github.com/ferranbt/fastssz"
	"github.com/hashicorp/go-hclog"
	"github.com/umbracle/eth2-validator/internal/beacon"
	"github.com/umbracle/eth2-validator/internal/bls"
	"github.com/umbracle/eth2-validator/internal/server/proto"
	"github.com/umbracle/eth2-validator/internal/server/state"
	"github.com/umbracle/eth2-validator/internal/server/structs"
	"google.golang.org/grpc"
)

// Server is a validator in the eth2.0 network
type Server struct {
	proto.UnimplementedValidatorServiceServer

	config     *Config
	logger     hclog.Logger
	shutdownCh chan struct{}
	client     *beacon.HttpAPI
	grpcServer *grpc.Server
	state      *state.State
	validator  *validator
}

type validator struct {
	index    uint64
	key      *bls.Key
	graffity []byte
}

// NewServer starts a new validator
func NewServer(logger hclog.Logger, config *Config) (*Server, error) {

	v := &Server{
		config:     config,
		logger:     logger,
		shutdownCh: make(chan struct{}),
	}

	vv := structs.AttestationData{}
	fmt.Println(vv)

	if err := v.setupGRPCServer(config.GrpcAddr); err != nil {
		return nil, err
	}

	//fmt.Println(config.BeaconConfig.MinGenesisTime)
	//fmt.Println(time.Now().Unix())

	v.client = beacon.NewHttpAPI("http://172.17.0.3:5050")

	// h.Events(context.Background())

	logger.Info("validator started")

	// fmt.Println(h.GetAttesterDuties(8, []string{"0"}))

	/*
		duties, err := h.GetProposerDuties(context.Background(), 5)
		if err != nil {
			panic(err)
		}
		for _, duty := range duties {
			fmt.Println("-- duty --")
			fmt.Println(duty.Slot)
		}
	*/

	v.addValidator()
	go v.run()

	return v, nil
}

func (v *Server) addValidator() {
	buf, err := hex.DecodeString("3dbab17387f48bdd0784d3c844c25bd06be26d8284fa6999827780b099e75a40")
	if err != nil {
		panic(err)
	}
	key, err := bls.NewKeyFromPriv(buf)
	if err != nil {
		panic(err)
	}

	graffity, err := hex.DecodeString("cf8e0d4e9587369b2301d0790347320302cc0943d5a1884560367e8208d920f2")
	if err != nil {
		panic(err)
	}

	v.validator = &validator{
		index:    0,
		key:      key,
		graffity: graffity,
	}
}

type dutyUpdates struct {
	epoch uint64
	slot  uint64
	//attester []*beacon.AttesterDuty
	proposer []*beacon.ProposerDuty // start here
}

func (v *Server) run() {

	for i := 14; i < 16; i++ {
		v.runDuty(&dutyUpdates{slot: uint64(i)})
	}

	return

	dutyUpdates := make(chan *dutyUpdates, 8)
	go v.watchDuties(dutyUpdates)

	for {
		select {
		case update := <-dutyUpdates:
			v.runDuty(update)

		case <-v.shutdownCh:
			return
		}
	}
}

func uint64SSZ(epoch uint64) *ssz.Hasher {
	hh := ssz.NewHasher()
	indx := hh.Index()
	hh.PutUint64(epoch)
	hh.Merkleize(indx)
	return hh
}

func (v *Server) runDuty(update *dutyUpdates) {
	fmt.Println("- run duty -")
	fmt.Println(update.proposer)

	// create the randao

	epochSSZ := uint64SSZ(update.epoch)
	randaoReveal := v.signHash(v.config.BeaconConfig.DomainRandao, epochSSZ)

	fmt.Println("-- randao reveal --")
	fmt.Println(randaoReveal)

	block, err := v.client.GetBlock(update.slot, randaoReveal)
	if err != nil {
		panic(err)
	}

	hh := ssz.NewHasher()
	if err := block.HashTreeRootWith(hh); err != nil {
		panic(err)
	}

	blockSignature := v.signHash(v.config.BeaconConfig.DomainBeaconProposer, hh)

	fmt.Println("-- block --")
	fmt.Println(blockSignature)

	signedBlock := &structs.SignedBeaconBlock{
		Block:     block,
		Signature: blockSignature,
	}

	if err := v.client.PublishSignedBlock(signedBlock); err != nil {
		panic(err)
	}
}

func (v *Server) signHash(domain structs.Domain, obj *ssz.Hasher) []byte {
	ddd, err := v.config.BeaconConfig.ComputeDomain(domain, nil, nil)
	if err != nil {
		panic(err)
	}

	unsignedMsgRoot, err := obj.HashRoot()
	if err != nil {
		panic(err)
	}
	rootToSign, err := ssz.HashWithDefaultHasher(&structs.SigningData{
		ObjectRoot: unsignedMsgRoot[:],
		Domain:     ddd,
	})
	if err != nil {
		panic(err)
	}

	signature, err := v.validator.key.Sign(rootToSign)
	if err != nil {
		panic(err)
	}
	return signature
}

func (v *Server) getEpoch(slot uint64) uint64 {
	return slot / v.config.BeaconConfig.SlotsPerEpoch
}

func (v *Server) watchDuties(updates chan *dutyUpdates) {
	secondsPerSlot := time.Duration(v.config.BeaconConfig.SecondsPerSlot) * time.Second
	fmt.Println(secondsPerSlot)

	timer := time.NewTimer(0)

	syncing, err := v.client.Syncing()
	if err != nil {
		panic(err)
	}
	headSlot := syncing.HeadSlot
	epoch := v.getEpoch(headSlot)

	fmt.Println("headSlot", headSlot, "epoch", epoch)

	// query duties for this epoch
	attesterDuties, err := v.client.GetAttesterDuties(epoch, []string{"0"})
	if err != nil {
		panic(err)
	}
	proposerDuties, err := v.client.GetProposerDuties(epoch)
	if err != nil {
		panic(err)
	}

	fmt.Println(attesterDuties)
	fmt.Println(proposerDuties)

	for {
		select {
		case <-timer.C:
			var slotAttester []*beacon.AttesterDuty
			for _, x := range attesterDuties {
				if x.Slot == headSlot {
					slotAttester = append(slotAttester, x)
				}
			}
			var slotProposer []*beacon.ProposerDuty
			for _, x := range proposerDuties {
				if x.Slot == headSlot {
					slotProposer = append(slotProposer, x)
				}
			}

			job := &dutyUpdates{
				epoch: epoch,
				slot:  headSlot,
				//attester: slotAttester,
				proposer: slotProposer,
			}

			updates <- job
			headSlot++

			timer.Reset(secondsPerSlot)

		case <-v.shutdownCh:
			return
		}
	}
}

// Stop stops the validator
func (v *Server) Stop() {
	close(v.shutdownCh)
}

func (s *Server) setupGRPCServer(addr string) error {
	if addr == "" {
		return nil
	}
	s.grpcServer = grpc.NewServer(s.withLoggingUnaryInterceptor())
	proto.RegisterValidatorServiceServer(s.grpcServer, s)

	lis, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}

	go func() {
		if err := s.grpcServer.Serve(lis); err != nil {
			s.logger.Error("failed to serve grpc server", "err", err)
		}
	}()

	s.logger.Info("GRPC Server started", "addr", addr)
	return nil
}

func (s *Server) withLoggingUnaryInterceptor() grpc.ServerOption {
	return grpc.UnaryInterceptor(s.loggingServerInterceptor)
}

func (s *Server) loggingServerInterceptor(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
	start := time.Now()
	h, err := handler(ctx, req)
	s.logger.Trace("Request", "method", info.FullMethod, "duration", time.Since(start), "error", err)
	return h, err
}
