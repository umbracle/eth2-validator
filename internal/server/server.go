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

	syncCommitteeSubscription *SyncCommitteeSubscription
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

func (v *Server) run() {
	// dutyUpdates := make(chan *dutyUpdates, 8)
	go v.watchDuties()

	// start the queue system
	v.evalQueue.Start()

	// run the worker
	go v.runWorker()
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

func (v *Server) watchDuties() {
	genesisTime := time.Unix(int64(v.genesis.Time), 0)

	cTime := chaintime.New(genesisTime, v.beaconConfig.SecondsPerSlot, v.beaconConfig.SlotsPerEpoch)
	if !cTime.IsActive() {
		v.logger.Info("genesis not active yet", "time left", time.Until(genesisTime))

		// genesis is not activated yet
		select {
		case <-time.After(time.Until(genesisTime)):
		case <-v.shutdownCh:
			return
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
