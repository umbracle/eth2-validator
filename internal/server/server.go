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
	"github.com/umbracle/eth2-validator/internal/scheduler"
	"github.com/umbracle/eth2-validator/internal/server/proto"
	"github.com/umbracle/eth2-validator/internal/server/state"
	"github.com/umbracle/eth2-validator/internal/version"
	consensus "github.com/umbracle/go-eth-consensus"
	"github.com/umbracle/go-eth-consensus/bls"
	"github.com/umbracle/go-eth-consensus/http"
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
	beaconConfig *consensus.Spec
	genesis      *http.Genesis
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

	genesis, err := v.client.Genesis(context.Background())
	if err != nil {
		return nil, err
	}
	v.genesis = genesis

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

	// emit metrics for the eval queue
	go v.evalQueue.EmitStats(time.Second, v.shutdownCh)

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
			ctx, span := otel.Tracer("Validator").Start(ctx, duty.Type().String())
			defer span.End()

			var res *proto.Duty_Result
			switch duty.Job.(type) {
			case *proto.Duty_BlockProposal_:
				res, err = v.runBlockProposal(ctx, duty)
			case *proto.Duty_Attestation_:
				res, err = v.runSingleAttestation(ctx, duty)
			case *proto.Duty_AttestationAggregate_:
				res, err = v.runAttestationAggregate(ctx, duty)
			case *proto.Duty_SyncCommittee_:
				res, err = v.runSyncCommittee(ctx, duty)
			case *proto.Duty_SyncCommitteeAggregate_:
				res, err = v.runSyncCommitteeAggregate(ctx, duty)
			}
			if err != nil {
				panic(fmt.Errorf("failed to handle %s: %v", duty.Type(), err))
			}

			// upsert the job on state
			duty.Result = res
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

func (v *Server) runSyncCommitteeAggregate(ctx context.Context, duty *proto.Duty) (*proto.Duty_Result, error) {
	job := duty.GetSyncCommitteeAggregate()

	latestRoot, err := v.client.GetHeadBlockRoot(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get head block root: %v", err)
	}

	contribution, err := v.client.SyncCommitteeContribution(ctx, duty.Slot, job.SubCommitteeIndex, latestRoot[:])
	if err != nil {
		return nil, fmt.Errorf("failed to get sync committee contribution: %v", err)
	}

	contributionAggregate := &consensus.ContributionAndProof{
		AggregatorIndex: duty.ValidatorIndex,
		Contribution:    contribution,
		SelectionProof:  consensus.ToBytes96(job.SelectionProof),
	}
	root, err := contributionAggregate.HashTreeRoot()
	if err != nil {
		return nil, err
	}

	signature, err := v.Sign(ctx, proto.DomainContributionAndProof, duty.Epoch, duty.ValidatorIndex, root)
	if err != nil {
		return nil, err
	}

	msg := &consensus.SignedContributionAndProof{
		Message:   contributionAggregate,
		Signature: signature,
	}
	if err := v.client.SubmitSignedContributionAndProof(ctx, []*consensus.SignedContributionAndProof{msg}); err != nil {
		return nil, fmt.Errorf("failed to submit signed committee aggregate proof: %v", err)
	}

	result := &proto.Duty_Result{
		SyncCommitteeAggregate: &proto.Duty_SyncCommitteeAggregateResult{},
	}
	return result, nil
}

func (v *Server) runSyncCommittee(ctx context.Context, duty *proto.Duty) (*proto.Duty_Result, error) {
	// get root
	latestRoot, err := v.client.GetHeadBlockRoot(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get head block root: %v", err)
	}

	signature, err := v.Sign(ctx, proto.DomainSyncCommitteeType, duty.Epoch, duty.ValidatorIndex, proto.RootSSZ(latestRoot))
	if err != nil {
		return nil, err
	}

	committeeDuty := []*consensus.SyncCommitteeMessage{
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
	result := &proto.Duty_Result{
		SyncCommittee: &proto.Duty_SyncCommitteeResult{},
	}
	return result, nil
}

func (v *Server) runSingleAttestation(ctx context.Context, duty *proto.Duty) (*proto.Duty_Result, error) {
	job := duty.GetAttestation()

	attestationData, err := v.client.RequestAttestationData(ctx, duty.Slot, job.CommitteeIndex)
	if err != nil {
		return nil, fmt.Errorf("failed to reuest attestation data: %v", err)
	}
	attestationRoot, err := attestationData.HashTreeRoot()
	if err != nil {
		return nil, err
	}

	attestedSignature, err := v.Sign(ctx, proto.DomainBeaconAttesterType, duty.Epoch, duty.ValidatorIndex, attestationRoot)
	if err != nil {
		return nil, err
	}

	bitlist := bitlist.NewBitlist(job.CommitteeLength)
	bitlist.SetBitAt(job.CommitteeIndex, true)

	attestation := &consensus.Attestation{
		Data:            attestationData,
		AggregationBits: bitlist,
		Signature:       attestedSignature,
	}
	if err := v.client.PublishAttestations(ctx, []*consensus.Attestation{attestation}); err != nil {
		return nil, fmt.Errorf("failed to publish attestations: %v", err)
	}

	// store the attestation in the state
	result := &proto.Duty_Result{
		Attestation: &proto.Duty_AttestationResult{
			Root: attestationRoot[:],
			Source: &proto.Duty_AttestationResult_Checkpoint{
				Root:  attestationData.Source.Root[:],
				Epoch: attestationData.Source.Epoch,
			},
			Target: &proto.Duty_AttestationResult_Checkpoint{
				Root:  attestationData.Target.Root[:],
				Epoch: attestationData.Target.Epoch,
			},
		},
	}
	return result, nil
}

func (v *Server) runAttestationAggregate(ctx context.Context, duty *proto.Duty) (*proto.Duty_Result, error) {
	job := duty.GetAttestationAggregate()

	attestation, err := v.state.DutyByID(duty.BlockedBy[0])
	if err != nil {
		return nil, err
	}

	aggregateAttestation, err := v.client.AggregateAttestation(ctx, duty.Slot, attestation.Result.Attestation.Root)
	if err != nil {
		return nil, fmt.Errorf("failed to aggregate attestation: %v", err)
	}

	// Sign the aggregate attestation.
	aggregateAndProof := &consensus.AggregateAndProof{
		Index:          duty.ValidatorIndex,
		Aggregate:      aggregateAttestation,
		SelectionProof: consensus.ToBytes96(job.SelectionProof),
	}
	aggregateAndProofRoot, err := aggregateAndProof.HashTreeRoot()
	if err != nil {
		return nil, err
	}

	aggregateAndProofRootSignature, err := v.Sign(ctx, proto.DomainAggregateAndProofType, duty.Epoch, duty.ValidatorIndex, proto.RootSSZ(aggregateAndProofRoot))
	if err != nil {
		return nil, err
	}

	req := []*consensus.SignedAggregateAndProof{
		{
			Message:   aggregateAndProof,
			Signature: aggregateAndProofRootSignature,
		},
	}
	if err := v.client.PublishAggregateAndProof(ctx, req); err != nil {
		return nil, fmt.Errorf("failed to publish aggregate and proof: %v", err)
	}

	result := &proto.Duty_Result{
		AttestationAggregate: &proto.Duty_AttestationAggregateResult{},
	}
	return result, nil
}

func (v *Server) runBlockProposal(ctx context.Context, duty *proto.Duty) (*proto.Duty_Result, error) {
	// create the randao
	randaoReveal, err := v.Sign(ctx, proto.DomainRandaomType, duty.Epoch, duty.ValidatorIndex, proto.Uint64SSZ(duty.Epoch))
	if err != nil {
		return nil, err
	}

	block, err := v.client.GetBlock(ctx, duty.Slot, randaoReveal[:])
	if err != nil {
		return nil, fmt.Errorf("failed to get block: %v", err)
	}

	blockRoot, err := block.HashTreeRoot()
	if err != nil {
		return nil, err
	}

	blockSignature, err := v.Sign(ctx, proto.DomainBeaconProposerType, duty.Epoch, duty.ValidatorIndex, blockRoot)
	if err != nil {
		return nil, err
	}
	signedBlock := &consensus.SignedBeaconBlock{
		Block:     block,
		Signature: blockSignature,
	}

	if err := v.client.PublishSignedBlock(ctx, signedBlock); err != nil {
		return nil, fmt.Errorf("failed to publish block: %v", err)
	}

	result := &proto.Duty_Result{
		BlockProposal: &proto.Duty_BlockProposalResult{
			Root:      block.StateRoot[:],
			Signature: blockSignature[:],
		},
	}
	return result, nil
}

func (v *Server) Sign(ctx context.Context, domain proto.DomainType, epoch uint64, accountIndex uint64, root [32]byte) ([96]byte, error) {
	_, span := otel.Tracer("Validator").Start(ctx, "Sign")
	defer span.End()

	var forkVersion [4]byte
	if v.beaconConfig.BellatrixForkEpoch <= epoch {
		forkVersion = v.beaconConfig.BellatrixForkVersion
	} else if v.beaconConfig.AltairForkEpoch <= epoch {
		forkVersion = v.beaconConfig.AltairForkVersion
	} else {
		forkVersion = v.beaconConfig.GenesisForkVersion
	}

	domainVal := domainTypToDomain(domain)

	ddd, err := consensus.ComputeDomain(domainVal, forkVersion, v.genesis.Root)
	if err != nil {
		return [96]byte{}, err
	}

	rootToSign, err := ssz.HashWithDefaultHasher(&consensus.SigningData{
		ObjectRoot: root,
		Domain:     ddd,
	})
	if err != nil {
		return [96]byte{}, err
	}
	validator, err := v.state.GetValidatorByIndex(accountIndex)
	if err != nil {
		return [96]byte{}, err
	}
	key, err := validator.Key()
	if err != nil {
		return [96]byte{}, err
	}

	signature, err := key.Sign(rootToSign)
	if err != nil {
		return [96]byte{}, err
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
	proposerDuties := []*http.ProposerDuty{}
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

func domainTypToDomain(typ proto.DomainType) consensus.Domain {
	switch typ {
	case proto.DomainBeaconProposerType:
		return consensus.Domain{0, 0, 0, 0}
	case proto.DomainBeaconAttesterType:
		return consensus.Domain{1, 0, 0, 0}
	case proto.DomainRandaomType:
		return consensus.Domain{2, 0, 0, 0}
	case proto.DomainDepositType:
		return consensus.Domain{3, 0, 0, 0}
	case proto.DomainVoluntaryExitType:
		return consensus.Domain{4, 0, 0, 0}
	case proto.DomainSelectionProofType:
		return consensus.Domain{5, 0, 0, 0}
	case proto.DomainAggregateAndProofType:
		return consensus.Domain{6, 0, 0, 0}
	case proto.DomainSyncCommitteeType:
		return consensus.Domain{7, 0, 0, 0}
	case proto.DomainSyncCommitteeSelectionProof:
		return consensus.Domain{8, 0, 0, 0}
	case proto.DomainContributionAndProof:
		return consensus.Domain{9, 0, 0, 0}
	default:
		panic(fmt.Errorf("domain typ not found: %s", typ))
	}
}
