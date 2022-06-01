package server

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net"
	"time"

	ssz "github.com/ferranbt/fastssz"
	"github.com/hashicorp/go-hclog"
	"github.com/umbracle/eth2-validator/internal/beacon"
	"github.com/umbracle/eth2-validator/internal/bitlist"
	"github.com/umbracle/eth2-validator/internal/bls"
	"github.com/umbracle/eth2-validator/internal/scheduler"
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
	evalQueue  *EvalQueue
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
		evalQueue:  NewEvalQueue(),
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
	v.client.SetLogger(logger)

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

func (v *Server) runWorker() {
	for {
		duty, err := v.evalQueue.Dequeue()
		if err != nil {
			panic(err)
		}

		v.logger.Info("handle duty", "id", duty.Id, "slot", duty.Slot, "typ", duty.Type())

		go func(duty *proto.Duty) {
			var job proto.DutyJob
			switch duty.Job.(type) {
			case *proto.Duty_BlockProposal:
				job, err = v.runBlockProposal(duty)
			case *proto.Duty_Attestation:
				job, err = v.runSingleAttestation(duty)
			case *proto.Duty_AttestationAggregate:
				job, err = v.runAttestationAggregate(duty)
			case *proto.Duty_SyncCommittee:
				job, err = v.runSyncCommittee(duty)
			}
			if err != nil {
				panic(err)
			}

			// upsert the job on state
			duty.Job = job
			if err := v.state.InsertDuty(duty); err != nil {
				panic(err)
			}

			v.evalQueue.Ack(duty.Id)
		}(duty)
	}
}

func (v *Server) run() {
	// dutyUpdates := make(chan *dutyUpdates, 8)
	go v.watchDuties()

	// start the queue system
	v.evalQueue.Start()

	// run the worker
	go v.runWorker()
}

func uint64SSZ(epoch uint64) *ssz.Hasher {
	hh := ssz.NewHasher()
	indx := hh.Index()
	hh.PutUint64(epoch)
	hh.Merkleize(indx)
	return hh
}

func signatureSSZ(sig []byte) *ssz.Hasher {
	hh := ssz.NewHasher()
	indx := hh.Index()
	hh.PutBytes(sig)
	hh.Merkleize(indx)
	return hh
}

func (v *Server) runSyncCommittee(duty *proto.Duty) (proto.DutyJob, error) {
	// TODO: hardcoded
	time.Sleep(1 * time.Second)

	// get root
	latestRoot, err := v.client.GetHeadBlockRoot()
	if err != nil {
		return nil, err
	}

	hh := signatureSSZ(latestRoot)
	signature, err := v.signHash(v.config.BeaconConfig.DomainSyncCommittee, hh)
	if err != nil {
		return nil, err
	}

	committeeDuty := []*beacon.SyncCommitteeMessage{
		{
			Slot:           duty.Slot,
			BlockRoot:      latestRoot,
			ValidatorIndex: 0,
			Signature:      signature,
		},
	}

	if err := v.client.SubmitCommitteeDuties(committeeDuty); err != nil {
		return nil, err
	}

	// store the attestation in the state
	job := &proto.Duty_SyncCommittee{
		SyncCommittee: &proto.SyncCommittee{},
	}
	return job, nil
}

func (v *Server) runSingleAttestation(duty *proto.Duty) (proto.DutyJob, error) {
	// TODO: hardcoded
	time.Sleep(1 * time.Second)

	var attestationInput *beacon.AttesterDuty
	if err := json.Unmarshal(duty.Input.Value, &attestationInput); err != nil {
		return nil, err
	}
	attestationData, err := v.client.RequestAttestationData(duty.Slot, attestationInput.CommitteeIndex)
	if err != nil {
		return nil, err
	}

	indexed := &structs.IndexedAttestation{
		AttestationIndices: []uint64{0},
		Data:               attestationData,
	}

	hh := ssz.NewHasher()
	if err := indexed.Data.HashTreeRootWith(hh); err != nil {
		return nil, err
	}

	attestedSignature, err := v.signHash(v.config.BeaconConfig.DomainBeaconAttester, hh)
	if err != nil {
		return nil, err
	}

	bitlist := bitlist.NewBitlist(attestationInput.CommitteeLength)
	bitlist.SetBitAt(attestationInput.CommitteeIndex, true)

	attestation := &structs.Attestation{
		Data:            attestationData,
		AggregationBits: bitlist,
		Signature:       attestedSignature,
	}
	if err := v.client.PublishAttestations([]*structs.Attestation{attestation}); err != nil {
		return nil, err
	}

	attestationRoot, err := attestationData.HashTreeRoot()
	if err != nil {
		return nil, err
	}

	// store the attestation in the state
	job := &proto.Duty_Attestation{
		Attestation: &proto.Attestation{
			Root: hex.EncodeToString(attestationRoot[:]),
			Source: &proto.Attestation_Checkpoint{
				Root:  hex.EncodeToString(attestationData.Source.Root[:]),
				Epoch: attestationData.Source.Epoch,
			},
			Target: &proto.Attestation_Checkpoint{
				Root:  hex.EncodeToString(attestationData.Target.Root[:]),
				Epoch: attestationData.Target.Epoch,
			},
		},
	}
	return job, nil
}

func (v *Server) runAttestationAggregate(duty *proto.Duty) (proto.DutyJob, error) {
	time.Sleep(1 * time.Second)

	attestation, err := v.state.DutyByID(duty.BlockedBy[0])
	if err != nil {
		panic(err)
	}
	blockProposal, err := v.state.DutyByID(duty.BlockedBy[1])
	if err != nil {
		panic(err)
	}

	blockSlotSig, err := hex.DecodeString(blockProposal.Job.(*proto.Duty_BlockProposal).BlockProposal.Signature)
	if err != nil {
		panic(err)
	}
	attestationRootC, err := hex.DecodeString(attestation.Job.(*proto.Duty_Attestation).Attestation.Root)
	if err != nil {
		panic(err)
	}
	attestationRoot := [32]byte{}
	copy(attestationRoot[:], attestationRootC)

	var attestationDuty *beacon.AttesterDuty
	if err := json.Unmarshal(attestation.Input.Value, &attestationDuty); err != nil {
		return nil, err
	}

	aggregateAttestation, err := v.client.AggregateAttestation(duty.Slot, attestationRoot)
	if err != nil {
		return nil, err
	}

	// Sign the aggregate attestation.
	aggregateAndProof := &structs.AggregateAndProof{
		Index:          uint64(attestationDuty.ValidatorIndex),
		Aggregate:      aggregateAttestation,
		SelectionProof: blockSlotSig,
	}
	aggregateAndProofRoot, err := aggregateAndProof.HashTreeRoot()
	if err != nil {
		panic(err)
	}

	hh := signatureSSZ(aggregateAndProofRoot[:])
	aggregateAndProofRootSignature, err := v.signHash(v.config.BeaconConfig.DomainAggregateAndProof, hh)
	if err != nil {
		return nil, err
	}

	req := []*beacon.SignedAggregateAndProof{
		{
			Message:   aggregateAndProof,
			Signature: aggregateAndProofRootSignature,
		},
	}
	if err := v.client.PublishAggregateAndProof(req); err != nil {
		return nil, err
	}
	return &proto.Duty_AttestationAggregate{}, nil
}

func (v *Server) runBlockProposal(duty *proto.Duty) (proto.DutyJob, error) {
	// create the randao

	epochSSZ := uint64SSZ(duty.Epoch)
	randaoReveal, err := v.signHash(v.config.BeaconConfig.DomainRandao, epochSSZ)
	if err != nil {
		return nil, err
	}

	block, err := v.client.GetBlock(duty.Slot, randaoReveal)
	if err != nil {
		return nil, err
	}

	hh := ssz.NewHasher()
	if err := block.HashTreeRootWith(hh); err != nil {
		return nil, err
	}

	blockSignature, err := v.signHash(v.config.BeaconConfig.DomainBeaconProposer, hh)
	if err != nil {
		return nil, err
	}
	signedBlock := &structs.SignedBeaconBlock{
		Block:     block,
		Signature: blockSignature,
	}

	if err := v.client.PublishSignedBlock(signedBlock); err != nil {
		return nil, err
	}

	job := &proto.Duty_BlockProposal{
		BlockProposal: &proto.BlockProposal{
			Root:      hex.EncodeToString(block.StateRoot),
			Signature: hex.EncodeToString(blockSignature),
		},
	}
	return job, nil
}

func (v *Server) signHash(domain structs.Domain, obj *ssz.Hasher) ([]byte, error) {
	genesis, err := v.client.Genesis(context.Background())
	if err != nil {
		return nil, err
	}

	ddd, err := v.config.BeaconConfig.ComputeDomain(domain, v.config.BeaconConfig.AltairForkVersion[:], genesis.Root)
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

func (v *Server) Sign(domain proto.DomainType, account uint64, root []byte) ([]byte, error) {
	genesis, err := v.client.Genesis(context.Background())
	if err != nil {
		return nil, err
	}

	domainVal := domainTypToDomain(domain, v.config.BeaconConfig)

	ddd, err := v.config.BeaconConfig.ComputeDomain(domainVal, v.config.BeaconConfig.AltairForkVersion[:], genesis.Root)
	if err != nil {
		return nil, err
	}

	rootToSign, err := ssz.HashWithDefaultHasher(&structs.SigningData{
		ObjectRoot: root,
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

func (v *Server) watchDuties() {
	genesis, err := v.client.Genesis(context.Background())
	if err != nil {
		panic(err)
	}

	genesisTime := time.Unix(int64(genesis.Time), 0)

	bb := newBeaconTracker(v.logger, genesisTime, v.config.BeaconConfig.SecondsPerSlot, v.config.BeaconConfig.SlotsPerEpoch)
	go bb.run()

	// wait for the ready channel to be closed
	<-bb.readyCh

	v.logger.Info("Start slot", "epoch", bb.startEpoch)

	for {
		select {
		case res := <-bb.resCh:
			v.logger.Info("Schedule duties", "epoch", res.Epoch)

			// query duties for this epoch
			attesterDuties, err := v.client.GetAttesterDuties(res.Epoch, []string{"0"})
			if err != nil {
				panic(err)
			}
			proposerDuties, err := v.client.GetProposerDuties(res.Epoch)
			if err != nil {
				panic(err)
			}
			committeeDuties, err := v.client.GetCommitteeSyncDuties(res.Epoch, []string{"0"})
			if err != nil {
				panic(err)
			}

			eval := &proto.Evaluation{
				Attestation: attesterDuties,
				Proposer:    proposerDuties,
				Committee:   committeeDuties,
				Epoch:       res.Epoch,
				GenesisTime: genesisTime,
			}

			sched := scheduler.NewScheduler(hclog.L(), v, v.config.BeaconConfig)
			plan, err := sched.Process(eval)
			if err != nil {
				panic(err)
			}

			v.evalQueue.Enqueue(plan.Duties)

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

func domainTypToDomain(typ proto.DomainType, config *beacon.ChainConfig) structs.Domain {
	switch typ {
	case proto.DomainBeaconProposerType:
		return config.DomainBeaconProposer
	case proto.DomainRandaomType:
		return config.DomainRandao
	case proto.DomainBeaconAttesterType:
		return config.DomainBeaconAttester
	case proto.DomainDepositType:
		return config.DomainDeposit
	case proto.DomainVoluntaryExitType:
		return config.DomainVoluntaryExit
	case proto.DomainSelectionProofType:
		return config.DomainSelectionProof
	case proto.DomainAggregateAndProofType:
		return config.DomainAggregateAndProof
	case proto.DomainSyncCommitteeType:
		return config.DomainSyncCommittee
	default:
		panic(fmt.Errorf("domain typ not found: %s", typ))
	}
}
