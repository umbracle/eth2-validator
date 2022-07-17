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
	"github.com/umbracle/eth2-validator/internal/version"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.4.0"
	"google.golang.org/grpc"
)

// Server is a validator in the eth2.0 network
type Server struct {
	config       *Config
	state        *state.State
	logger       hclog.Logger
	shutdownCh   chan struct{}
	client       *beacon.HttpAPI
	grpcServer   *grpc.Server
	evalQueue    *EvalQueue
	beaconConfig *beacon.ChainConfig
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

	beaconConfig, err := v.client.ConfigSpec()
	if err != nil {
		return nil, err
	}
	v.beaconConfig = beaconConfig

	for _, privKey := range config.PrivKey {
		if err := v.addValidator(privKey); err != nil {
			return nil, fmt.Errorf("failed to start validator %v", err)
		}
	}

	logger.Info("validator started")
	go v.run()

	if config.TelemetryOLTPExporter != "" {
		if err := v.setupTelemetry(); err != nil {
			return nil, err
		}
	}

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

	pubKey := key.PubKey()
	pubKeyStr := hex.EncodeToString(pubKey[:])

	val, err := v.client.GetValidatorByPubKey(context.Background(), "0x"+pubKeyStr)
	if err != nil {
		panic(err)
	}
	if val == nil {
		return fmt.Errorf("only ready available is allowed")
	}

	validator := &proto.Validator{
		PrivKey:         privKey,
		PubKey:          hex.EncodeToString(pubKey[:]),
		Index:           val.Index,
		ActivationEpoch: val.Validator.ActivationEpoch,
	}
	if err := v.state.UpsertValidator(validator); err != nil {
		return err
	}
	return nil
}

func (v *Server) runWorker() {
	for {
		duty, ctx, err := v.evalQueue.Dequeue()
		if err != nil {
			panic(err)
		}

		v.logger.Info("handle duty", "id", duty.Id, "slot", duty.Slot, "validator", duty.ValidatorIndex, "typ", duty.Type())

		go func(ctx context.Context, duty *proto.Duty) {
			ctx, span := otel.Tracer("Validator").Start(ctx, duty.Type())
			defer span.End()

			var job proto.DutyJob
			switch duty.Job.(type) {
			case *proto.Duty_BlockProposal:
				job, err = v.runBlockProposal(ctx, duty)
			case *proto.Duty_Attestation:
				job, err = v.runSingleAttestation(ctx, duty)
			case *proto.Duty_AttestationAggregate:
				job, err = v.runAttestationAggregate(ctx, duty)
			case *proto.Duty_SyncCommittee:
				job, err = v.runSyncCommittee(ctx, duty)
			}
			if err != nil {
				panic(fmt.Errorf("failed to handle %s: %v", duty.Type(), err))
			}

			// upsert the job on state
			duty.Job = job
			if err := v.state.InsertDuty(duty); err != nil {
				panic(err)
			}

			v.evalQueue.Ack(duty.Id)
		}(ctx, duty)
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

func (v *Server) runSyncCommittee(ctx context.Context, duty *proto.Duty) (proto.DutyJob, error) {
	// get root
	latestRoot, err := v.client.GetHeadBlockRoot(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get head block root: %v", err)
	}

	signature, err := v.Sign(ctx, proto.DomainSyncCommitteeType, duty.Epoch, duty.ValidatorIndex, proto.RootSSZ(latestRoot))
	if err != nil {
		return nil, err
	}

	committeeDuty := []*beacon.SyncCommitteeMessage{
		{
			Slot:           duty.Slot,
			BlockRoot:      latestRoot,
			ValidatorIndex: duty.ValidatorIndex,
			Signature:      signature,
		},
	}

	if err := v.client.SubmitCommitteeDuties(ctx, committeeDuty); err != nil {
		return nil, fmt.Errorf("failed to submit committee duties: %v", err)
	}

	// store the attestation in the state
	job := &proto.Duty_SyncCommittee{
		SyncCommittee: &proto.SyncCommittee{},
	}
	return job, nil
}

func (v *Server) runSingleAttestation(ctx context.Context, duty *proto.Duty) (proto.DutyJob, error) {
	var attestationInput *beacon.AttesterDuty
	if err := json.Unmarshal(duty.Input.Value, &attestationInput); err != nil {
		return nil, err
	}
	attestationData, err := v.client.RequestAttestationData(ctx, duty.Slot, attestationInput.CommitteeIndex)
	if err != nil {
		panic(err)
	}
	attestationRoot, err := attestationData.HashTreeRoot()
	if err != nil {
		return nil, err
	}

	attestedSignature, err := v.Sign(ctx, proto.DomainBeaconAttesterType, duty.Epoch, duty.ValidatorIndex, attestationRoot[:])
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
	if err := v.client.PublishAttestations(ctx, []*structs.Attestation{attestation}); err != nil {
		panic(err)
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

func (v *Server) runAttestationAggregate(ctx context.Context, duty *proto.Duty) (proto.DutyJob, error) {
	attestation, err := v.state.DutyByID(duty.BlockedBy[0])
	if err != nil {
		panic(err)
	}

	attestationRootC, err := hex.DecodeString(attestation.Job.(*proto.Duty_Attestation).Attestation.Root)
	if err != nil {
		panic(err)
	}
	attestationRoot := [32]byte{}
	copy(attestationRoot[:], attestationRootC)

	selectionProof, err := hex.DecodeString(duty.GetAttestationAggregate().SelectionProof)
	if err != nil {
		panic(err)
	}

	aggregateAttestation, err := v.client.AggregateAttestation(ctx, duty.Slot, attestationRoot)
	if err != nil {
		return nil, fmt.Errorf("failed to aggregate attestation: %v", err)
	}

	// Sign the aggregate attestation.
	aggregateAndProof := &structs.AggregateAndProof{
		Index:          duty.ValidatorIndex,
		Aggregate:      aggregateAttestation,
		SelectionProof: selectionProof,
	}
	aggregateAndProofRoot, err := aggregateAndProof.HashTreeRoot()
	if err != nil {
		panic(err)
	}

	aggregateAndProofRootSignature, err := v.Sign(ctx, proto.DomainAggregateAndProofType, duty.Epoch, duty.ValidatorIndex, proto.RootSSZ(aggregateAndProofRoot[:]))
	if err != nil {
		return nil, err
	}

	req := []*beacon.SignedAggregateAndProof{
		{
			Message:   aggregateAndProof,
			Signature: aggregateAndProofRootSignature,
		},
	}
	if err := v.client.PublishAggregateAndProof(ctx, req); err != nil {
		return nil, fmt.Errorf("failed to publish aggregate and proof: %v", err)
	}
	return &proto.Duty_AttestationAggregate{}, nil
}

func (v *Server) runBlockProposal(ctx context.Context, duty *proto.Duty) (proto.DutyJob, error) {
	// create the randao

	randaoReveal, err := v.Sign(ctx, proto.DomainRandaomType, duty.Epoch, duty.ValidatorIndex, proto.Uint64SSZ(duty.Epoch))
	if err != nil {
		return nil, err
	}

	block, err := v.client.GetBlock(ctx, duty.Slot, randaoReveal)
	if err != nil {
		return nil, err
	}

	blockRoot, err := block.HashTreeRoot()
	if err != nil {
		return nil, err
	}

	blockSignature, err := v.Sign(ctx, proto.DomainBeaconProposerType, duty.Epoch, duty.ValidatorIndex, blockRoot[:])
	if err != nil {
		return nil, err
	}
	signedBlock := &structs.SignedBeaconBlock{
		Block:     block,
		Signature: blockSignature,
	}

	if err := v.client.PublishSignedBlock(ctx, signedBlock); err != nil {
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

func (v *Server) Sign(ctx context.Context, domain proto.DomainType, epoch uint64, accountIndex uint64, root []byte) ([]byte, error) {
	_, span := otel.Tracer("Validator").Start(ctx, "Sign")
	defer span.End()

	genesis, err := v.client.Genesis(ctx)
	if err != nil {
		return nil, err
	}

	var forkVersion []byte
	if v.beaconConfig.BellatrixForkEpoch <= epoch {
		forkVersion = v.beaconConfig.BellatrixForkVersion[:]
	} else if v.beaconConfig.AltairForkEpoch <= epoch {
		forkVersion = v.beaconConfig.AltairForkVersion[:]
	} else {
		forkVersion = v.beaconConfig.GenesisForkVersion[:]
	}

	domainVal := domainTypToDomain(domain, v.beaconConfig)

	ddd, err := v.beaconConfig.ComputeDomain(domainVal, forkVersion, genesis.Root)
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
	validator, err := v.state.GetValidatorByIndex(accountIndex)
	if err != nil {
		return nil, err
	}
	key, err := validator.Key()
	if err != nil {
		return nil, err
	}

	signature, err := key.Sign(rootToSign)
	if err != nil {
		return nil, err
	}
	return signature, nil
}

func (v *Server) handleNewEpoch(genesisTime time.Time, epoch uint64) error {
	v.logger.Info("Schedule duties", "epoch", epoch)

	ctx, span := otel.Tracer("Validator").Start(context.Background(), "Epoch")
	defer span.End()

	validators, err := v.state.GetValidatorsActiveAt(epoch)
	if err != nil {
		return err
	}

	validatorsByIndex := map[uint]struct{}{}
	validatorsArray := []string{}
	for _, val := range validators {
		validatorsByIndex[uint(val.Index)] = struct{}{}
		validatorsArray = append(validatorsArray, fmt.Sprintf("%d", val.Index))
	}

	// query duties for this epoch
	attesterDuties, err := v.client.GetAttesterDuties(ctx, epoch, validatorsArray)
	if err != nil {
		return err
	}

	fullProposerDuties, err := v.client.GetProposerDuties(ctx, epoch)
	if err != nil {
		return err
	}
	proposerDuties := []*beacon.ProposerDuty{}
	for _, duty := range fullProposerDuties {
		if _, ok := validatorsByIndex[duty.ValidatorIndex]; ok {
			proposerDuties = append(proposerDuties, duty)
		}
	}

	committeeDuties, err := v.client.GetCommitteeSyncDuties(ctx, epoch, validatorsArray)
	if err != nil {
		return err
	}

	eval := &proto.Evaluation{
		Attestation: attesterDuties,
		Proposer:    proposerDuties,
		Committee:   committeeDuties,
		Epoch:       epoch,
		GenesisTime: genesisTime,
	}

	schedCtx, span := otel.Tracer("Validator").Start(ctx, "Scheduler")
	defer span.End()

	sched := scheduler.NewScheduler(v.logger.Named("scheduler"), schedCtx, v, v.beaconConfig)
	plan, err := sched.Process(eval)
	if err != nil {
		return err
	}

	v.evalQueue.Enqueue(ctx, plan.Duties)
	return nil
}

func (v *Server) watchDuties() {
	genesis, err := v.client.Genesis(context.Background())
	if err != nil {
		panic(err)
	}

	genesisTime := time.Unix(int64(genesis.Time), 0)

	bb := newBeaconTracker(v.logger, genesisTime, v.beaconConfig.SecondsPerSlot, v.beaconConfig.SlotsPerEpoch)
	go bb.run()

	// wait for the ready channel to be closed
	<-bb.readyCh

	v.logger.Info("Start slot", "epoch", bb.startEpoch)

	for {
		select {
		case res := <-bb.resCh:
			// check the state of the validators

			go func() {
				if err := v.handleNewEpoch(genesisTime, res.Epoch); err != nil {
					v.logger.Error("failed to schedule epoch", "epoch", res.Epoch, "err", err)
				}
			}()

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

func (s *Server) setupTelemetry() error {
	s.logger.Info("Telemetry enabled", "otel-exporter", s.config.TelemetryOLTPExporter)

	ctx := context.Background()

	res, err := resource.New(ctx,
		resource.WithAttributes(
			semconv.ServiceNameKey.String("eth2-validator"),
			semconv.ServiceVersionKey.String(version.GetVersion()),
		),
	)
	if err != nil {
		return fmt.Errorf("failed to create open telemetry resource for service: %v", err)
	}

	var exporters []sdktrace.SpanExporter

	// exporter for otel-collector
	if s.config.TelemetryOLTPExporter != "" {
		oltpExporter, err := otlptracegrpc.New(
			ctx,
			otlptracegrpc.WithInsecure(),
			otlptracegrpc.WithEndpoint(s.config.TelemetryOLTPExporter),
		)
		if err != nil {
			return fmt.Errorf("failed to create open telemetry tracer exporter for service: %v", err)
		}
		exporters = append(exporters, oltpExporter)
	}

	opts := []sdktrace.TracerProviderOption{
		sdktrace.WithSampler(sdktrace.AlwaysSample()),
		sdktrace.WithResource(res),
	}
	for _, exporter := range exporters {
		opts = append(opts, sdktrace.WithSyncer(exporter))
	}
	tracerProvider := sdktrace.NewTracerProvider(opts...)

	otel.SetTracerProvider(tracerProvider)
	otel.SetTextMapPropagator(propagation.TraceContext{})

	return nil
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
		// FIX: lighthouse does not return this constant value
		return structs.Domain{7, 0, 0, 0}
	default:
		panic(fmt.Errorf("domain typ not found: %s", typ))
	}
}
