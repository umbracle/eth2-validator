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
	"github.com/umbracle/eth2-validator/internal/bitlist"
	"github.com/umbracle/eth2-validator/internal/bls"
	"github.com/umbracle/eth2-validator/internal/server/proto"
	"github.com/umbracle/eth2-validator/internal/server/state"
	"github.com/umbracle/eth2-validator/internal/server/structs"
	"google.golang.org/grpc"
)

// Server is a validator in the eth2.0 network
type Server struct {
	config     *Config
	state      *state.State
	logger     hclog.Logger
	shutdownCh chan struct{}
	client     *beacon.HttpAPI
	grpcServer *grpc.Server
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

	state, err := state.NewState("TODO")
	if err != nil {
		return nil, fmt.Errorf("failed to start state: %v", err)
	}
	v.state = state

	if err := v.setupGRPCServer(config.GrpcAddr); err != nil {
		return nil, err
	}

	v.client = beacon.NewHttpAPI(config.Endpoint)

	if err := v.addValidator(config.PrivKey); err != nil {
		return nil, fmt.Errorf("failed to start validator %v", err)
	}

	logger.Info("validator started")
	go v.run()
	return v, nil
}

func (v *Server) addValidator(privKey string) error {
	buf, err := hex.DecodeString(privKey)
	if err != nil {
		return err
	}
	key, err := bls.NewKeyFromPriv(buf)
	if err != nil {
		return err
	}

	graffity, err := hex.DecodeString("cf8e0d4e9587369b2301d0790347320302cc0943d5a1884560367e8208d920f2")
	if err != nil {
		return err
	}

	v.validator = &validator{
		index:    0,
		key:      key,
		graffity: graffity,
	}
	return nil
}

type dutyUpdates struct {
	epoch    uint64
	slot     uint64
	attester []*beacon.AttesterDuty
	proposer []*beacon.ProposerDuty // start here
}

func (v *Server) run() {

	dutyUpdates := make(chan *dutyUpdates, 8)
	go v.watchDuties(dutyUpdates)

	for {
		select {
		case update := <-dutyUpdates:

			if update.attester != nil {
				if err := v.runAttestation(update.attester[0]); err != nil {
					v.logger.Error("failed to send attestation", "err", err)
				} else {
					v.logger.Info("attestation proposed successfully", "slot", update.slot)
				}
			}

			if err := v.runDuty(update); err != nil {
				v.logger.Error("failed to propose block: %v", err)
			} else {
				v.logger.Info("block proposed successfully", "slot", update.slot)
			}

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

func (v *Server) runAttestation(duty *beacon.AttesterDuty) error {
	// TODO: hardcoded
	time.Sleep(1 * time.Second)

	attestationData, err := v.client.RequestAttestationData(duty.Slot, duty.CommitteeIndex)
	if err != nil {
		return err
	}

	indexed := &structs.IndexedAttestation{
		AttestationIndices: []uint64{0},
		Data:               attestationData,
	}

	hh := ssz.NewHasher()
	if err := indexed.Data.HashTreeRootWith(hh); err != nil {
		return err
	}

	attestedSignature, err := v.signHash(v.config.BeaconConfig.DomainBeaconAttester, hh)
	if err != nil {
		return err
	}

	bitlist := bitlist.NewBitlist(duty.CommitteeLength)
	bitlist.SetBitAt(duty.CommitteeIndex, true)

	attestation := &structs.Attestation{
		Data:            attestationData,
		AggregationBits: bitlist,
		Signature:       attestedSignature,
	}
	if err := v.client.PublishAttestations([]*structs.Attestation{attestation}); err != nil {
		return err
	}

	// store the attestation in the state
	job := &proto.Duty{
		PubKey: "pubkey",
		Slot:   attestationData.Slot,
		Job: &proto.Duty_Attestation{
			Attestation: &proto.Attestation{
				Source: &proto.Attestation_Checkpoint{
					Root:  hex.EncodeToString(attestationData.Source.Root[:]),
					Epoch: attestationData.Source.Epoch,
				},
				Target: &proto.Attestation_Checkpoint{
					Root:  hex.EncodeToString(attestationData.Target.Root[:]),
					Epoch: attestationData.Target.Epoch,
				},
			},
		},
	}
	if v.state.InsertDuty(job); err != nil {
		return err
	}
	return nil
}

func (v *Server) runDuty(update *dutyUpdates) error {
	// create the randao

	epochSSZ := uint64SSZ(update.epoch)
	randaoReveal, err := v.signHash(v.config.BeaconConfig.DomainRandao, epochSSZ)
	if err != nil {
		return err
	}

	block, err := v.client.GetBlock(update.slot, randaoReveal)
	if err != nil {
		return err
	}

	hh := ssz.NewHasher()
	if err := block.HashTreeRootWith(hh); err != nil {
		return err
	}

	blockSignature, err := v.signHash(v.config.BeaconConfig.DomainBeaconProposer, hh)
	if err != nil {
		return err
	}
	signedBlock := &structs.SignedBeaconBlock{
		Block:     block,
		Signature: blockSignature,
	}

	if err := v.client.PublishSignedBlock(signedBlock); err != nil {
		return err
	}

	// save the proposed block in the state
	job := &proto.Duty{
		PubKey: "pubkey",
		Slot:   update.slot,
		Job: &proto.Duty_BlockProposal{
			BlockProposal: &proto.BlockProposal{
				Root: hex.EncodeToString(block.StateRoot),
			},
		},
	}
	if err := v.state.InsertDuty(job); err != nil {
		return err
	}
	return nil
}

func (v *Server) signHash(domain structs.Domain, obj *ssz.Hasher) ([]byte, error) {
	genesis, err := v.client.Genesis(context.Background())
	if err != nil {
		return nil, err
	}

	ddd, err := v.config.BeaconConfig.ComputeDomain(domain, nil, genesis.Root)
	if err != nil {
		return nil, err
	}

	unsignedMsgRoot, err := obj.HashRoot()
	if err != nil {
		return nil, err
	}

	rootToSign, err := ssz.HashWithDefaultHasher(&structs.SigningData{
		ObjectRoot: unsignedMsgRoot[:],
		Domain:     ddd,
	})
	if err != nil {
		return nil, err
	}

	signature, err := v.validator.key.Sign(rootToSign)
	if err != nil {
		return nil, err
	}
	return signature, nil
}

func (v *Server) getEpoch(slot uint64) uint64 {
	return slot / v.config.BeaconConfig.SlotsPerEpoch
}

func (v *Server) isEpochStart(slot uint64) bool {
	return slot%v.config.BeaconConfig.SlotsPerEpoch == 0
}

func (v *Server) isEpochEnd(slot uint64) bool {
	return v.isEpochStart(slot + 1)
}

func (v *Server) watchDuties(updates chan *dutyUpdates) {
	genesis, err := v.client.Genesis(context.Background())
	if err != nil {
		panic(err)
	}

	genesisTime := time.Unix(int64(genesis.Time), 0)

	bb := newBeaconTracker(v.logger, genesisTime, v.config.BeaconConfig.SecondsPerSlot)
	go bb.run()

	// wait for the ready channel to be closed
	<-bb.readyCh

	epoch := v.getEpoch(bb.startSlot)
	v.logger.Info("Start slot", "slot", bb.startSlot, "epoch", epoch)

EPOCH:

	// query duties for this epoch
	attesterDuties, err := v.client.GetAttesterDuties(epoch, []string{"0"})
	if err != nil {
		panic(err)
	}
	proposerDuties, err := v.client.GetProposerDuties(epoch)
	if err != nil {
		panic(err)
	}

	for {
		select {
		case res := <-bb.resCh:
			slot := res.Slot

			var slotAttester []*beacon.AttesterDuty
			for _, x := range attesterDuties {
				if x.Slot == slot {
					slotAttester = append(slotAttester, x)
				}
			}

			var slotProposer []*beacon.ProposerDuty
			for _, x := range proposerDuties {
				if x.Slot == slot {
					slotProposer = append(slotProposer, x)
				}
			}

			job := &dutyUpdates{
				epoch:    epoch,
				slot:     slot,
				attester: slotAttester,
				proposer: slotProposer,
			}
			updates <- job

			if v.isEpochEnd(res.Slot) {
				epoch++
				goto EPOCH
			}

		case <-v.shutdownCh:
			return
		}
	}
}

// Stop stops the validator
func (v *Server) Stop() {
	if err := v.state.Close(); err != nil {
		v.logger.Error("failed to stop state", "err", err)
	}
	v.grpcServer.Stop()
	close(v.shutdownCh)
}

func (s *Server) setupGRPCServer(addr string) error {
	if addr == "" {
		return nil
	}
	s.grpcServer = grpc.NewServer(s.withLoggingUnaryInterceptor())
	proto.RegisterValidatorServiceServer(s.grpcServer, &service{srv: s})

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
