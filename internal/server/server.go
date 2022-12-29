package server

import (
	"context"
	"encoding/hex"
	"fmt"
	"net"
	"time"

	ssz "github.com/ferranbt/fastssz"
	"github.com/hashicorp/go-hclog"
	"github.com/hashicorp/go-memdb"
	"github.com/umbracle/eth2-validator/internal/beacon"
	"github.com/umbracle/eth2-validator/internal/routine"
	"github.com/umbracle/eth2-validator/internal/scheduler"
	"github.com/umbracle/eth2-validator/internal/server/proto"
	"github.com/umbracle/eth2-validator/internal/server/state"
	"github.com/umbracle/eth2-validator/internal/version"
	consensus "github.com/umbracle/go-eth-consensus"
	"github.com/umbracle/go-eth-consensus/bls"
	"github.com/umbracle/go-eth-consensus/chaintime"
	"github.com/umbracle/go-eth-consensus/http"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
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
	client       beacon.Api
	grpcServer   *grpc.Server
	evalQueue    *DutyQueue
	beaconConfig *consensus.Spec
	genesis      *http.GenesisInfo

	routineManager *routine.Manager

	syncCommitteeSubscription *SyncCommitteeSubscription

	syncBeaconCommitteeSubscription *SyncBeaconCommitteeSubscription
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

	client := beacon.NewHttpAPI(config.Endpoint)
	client.SetLogger(logger)
	v.client = client

	beaconConfig, err := v.client.ConfigSpec()
	if err != nil {
		return nil, err
	}
	v.beaconConfig = beaconConfig

	v.syncCommitteeSubscription = NewSyncCommitteeSubscription(state, client)
	v.syncCommitteeSubscription.SetLogger(logger)

	v.syncBeaconCommitteeSubscription = NewSyncBeaconCommitteeSubscription(state, client)
	v.syncBeaconCommitteeSubscription.SetLogger(logger)

	v.routineManager = routine.NewManager(logger)

	genesis, err := v.client.Genesis(context.Background())
	if err != nil {
		return nil, err
	}
	v.genesis = genesis

	initialVals := []*proto.Validator{}
	for _, privKey := range config.PrivKey {
		val, err := validatorFromPrivKey(privKey)
		if err != nil {
			return nil, err
		}
		initialVals = append(initialVals, val)
	}
	if err := v.state.UpsertValidator(initialVals...); err != nil {
		return nil, err
	}

	v.evalQueue.Start()

	v.setupRoutines()

	if config.TelemetryOLTPExporter != "" {
		if err := v.setupTelemetry(); err != nil {
			return nil, err
		}
	}

	// emit metrics for the eval queue
	go v.evalQueue.EmitStats(time.Second, v.shutdownCh)

	return v, nil
}

func validatorFromPrivKey(privKey string) (*proto.Validator, error) {
	buf, err := hex.DecodeString(privKey)
	if err != nil {
		return nil, err
	}
	key, err := bls.NewKeyFromPriv(buf)
	if err != nil {
		return nil, err
	}

	pubKey := key.PubKey()

	validator := &proto.Validator{
		PrivKey: privKey,
		PubKey:  hex.EncodeToString(pubKey[:]),
	}
	return validator, nil
}

func (v *Server) runWorker(ctx context.Context) error {
	for {
		duty, ctx, err := v.evalQueue.Dequeue()
		if err != nil {
			panic(err)
		}

		v.logger.Info("handle duty", "id", duty.Id, "slot", duty.Slot, "validator", duty.ValidatorIndex, "typ", duty.Type())

		go func(ctx context.Context, duty *proto.Duty) {
			ctx, span := otel.Tracer("Validator").Start(ctx, duty.Type().String())
			defer span.End()

			dutyLogic := scheduler.NewDuty(ctx, v.client, v)

			res, err := dutyLogic.Handle(duty)
			if err != nil {
				panic(fmt.Errorf("failed to handle %s: %v", duty.Type(), err))
			}

			// upsert the job on state
			duty.Result = res
			duty.State = proto.Duty_COMPLETE

			if err := v.state.UpsertDuty(duty); err != nil {
				panic(err)
			}

			v.evalQueue.Ack(duty.Id)
		}(ctx, duty)
	}
}

func (v *Server) DutyByID(dutyID string) (*proto.Duty, error) {
	return v.state.DutyByID(dutyID)
}

func (v *Server) Run() {
	v.routineManager.Start(context.Background())
	v.logger.Info("validator started")
}

func (v *Server) setupRoutines() {
	// watch for new duties for the validators
	v.routineManager.Add("watch-duties", v.watchDuties)

	// start the sync committee subscription service
	//go v.syncCommitteeSubscription.Run()

	// start the sync beacon committee subscription service
	//go v.syncBeaconCommitteeSubscription.Run()

	// start the validator lifecycle service to find new validators
	v.routineManager.Add("validator-lifecycle", v.validatorLifecycle)

	// start the worker to process duties
	v.routineManager.Add("duty-worker", v.runWorker)
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
	if validator == nil {
		return [96]byte{}, fmt.Errorf("validator with index %d not found", accountIndex)
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

func (v *Server) handleNewEpoch(epoch uint64) error {
	v.logger.Info("Schedule duties", "epoch", epoch)

	ctx, span := otel.Tracer("Validator").Start(context.Background(), "Process epoch")
	span.SetAttributes(attribute.Int64("epoch", int64(epoch)))
	defer span.End()

	sched := scheduler.NewScheduler(ctx, v.client, v, v.beaconConfig)
	plan, err := sched.HandleEpoch(epoch)
	if err != nil {
		return err
	}

	v.logger.Debug(plan.GoPrint())

	v.evalQueue.Enqueue(ctx, plan.Duties)
	if err := v.state.UpsertDuties(plan.Duties); err != nil {
		return err
	}
	return nil
}

func (v *Server) GetValidatorsActiveAt(epoch uint64) ([]*proto.Validator, error) {
	return v.state.GetValidatorsActiveAt(epoch)
}

func (v *Server) Genesis() time.Time {
	return time.Unix(int64(v.genesis.Time), 0)
}

func (v *Server) watchDuties(ctx context.Context) error {
	genesisTime := time.Unix(int64(v.genesis.Time), 0)

	cTime := chaintime.New(genesisTime, v.beaconConfig.SecondsPerSlot, v.beaconConfig.SlotsPerEpoch)
	if !cTime.IsActive() {
		v.logger.Info("genesis not active yet", "time left", time.Until(genesisTime))

		// genesis is not activated yet
		select {
		case <-time.After(time.Until(genesisTime)):
		case <-v.shutdownCh:
			return nil
		}
	}

	epoch := cTime.CurrentEpoch()
	v.logger.Info("genesis is active", "current epoch", epoch.Number)

	for {
		// handle the epoch
		go func(epoch uint64) {
			if err := v.handleNewEpoch(epoch); err != nil {
				v.logger.Error("failed to schedule epoch", "epoch", epoch, "err", err)
			}
		}(epoch.Number)

		// increase the epoch and wait for it
		epoch = cTime.Epoch(epoch.Number + 1)

		nextEpoch := epoch.Until()
		v.logger.Debug("waiting for next epoch", "num", epoch.Number, "time left", nextEpoch)

		select {
		case <-time.After(nextEpoch):
		case <-v.shutdownCh:
			return nil
		}
	}
}

// Stop stops the validator
func (v *Server) Stop() {
	if err := v.state.Close(); err != nil {
		v.logger.Error("failed to stop state", "err", err)
	}
	// TODO: wait for routine manager
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

func (s *Server) validatorLifecycle(ctx context.Context) error {
	genesisTime := time.Unix(int64(s.genesis.Time), 0)

	cTime := chaintime.New(genesisTime, s.beaconConfig.SecondsPerSlot, s.beaconConfig.SlotsPerEpoch)
	epoch := cTime.CurrentEpoch()

	for {
		pending, err := s.state.GetValidatorsPending(memdb.NewWatchSet())
		if err != nil {
			panic(err)
		}

		updatedVal := []*proto.Validator{}
		for _, val := range pending {
			httpVal, err := s.client.GetValidatorByPubKey(context.Background(), "0x"+val.PubKey)
			if err != nil {
				s.logger.Error("failed to query validator", "pubKey", val.PubKey, "err", err)
			} else {
				// the validator is active, update it
				val = val.Copy()
				val.Metadata = &proto.Validator_Metadata{
					Index:           httpVal.Index,
					ActivationEpoch: httpVal.Validator.ActivationEpoch,
				}

				updatedVal = append(updatedVal, val)

				s.logger.Debug("validator registered", "index", val.Metadata.Index, "epoch", val.Metadata.ActivationEpoch)
			}
		}

		if s.state.UpsertValidator(updatedVal...); err != nil {
			s.logger.Error("failed to upsert validator", "err", err)
		}

		// increase the epoch and wait for it
		epoch = cTime.Epoch(epoch.Number + 1)

		nextEpoch := epoch.Until()

		select {
		case <-time.After(nextEpoch):
		case <-s.shutdownCh:
			return nil
		}
	}
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
