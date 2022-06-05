package testutil

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/hashicorp/go-hclog"
	"github.com/umbracle/eth2-validator/internal/testutil/proto"
	"google.golang.org/grpc"
)

type Config struct {
	Name     string
	LogLevel string
	Spec     *Eth2Spec
}

func DefaultConfig() *Config {
	return &Config{
		Name:     "random",
		LogLevel: "info",
		Spec:     &Eth2Spec{},
	}
}

type Server struct {
	proto.UnimplementedE2EServiceServer
	config *Config
	logger hclog.Logger
	eth1   *Eth1Server

	lock       sync.Mutex
	fileLogger *fileLogger
	nodes      []*node
}

func NewServer(logger hclog.Logger, config *Config) (*Server, error) {
	eth1, err := NewEth1Server()
	if err != nil {
		return nil, err
	}

	// create a folder to store data
	if err := os.Mkdir(config.Name, 0755); err != nil {
		return nil, err
	}

	logger.Info("eth1 server deployed", "addr", eth1.GetAddr(NodePortEth1Http))
	logger.Info("deposit contract deployed", "addr", eth1.deposit.String())

	config.Spec.DepositContract = eth1.deposit.String()

	srv := &Server{
		config:     config,
		logger:     logger,
		eth1:       eth1,
		nodes:      []*node{},
		fileLogger: &fileLogger{path: config.Name},
	}

	if err := srv.writeFile("spec.yaml", config.Spec.buildConfig()); err != nil {
		return nil, err
	}

	grpcServer := grpc.NewServer()
	proto.RegisterE2EServiceServer(grpcServer, srv)

	grpcAddr := "localhost:5555"
	lis, err := net.Listen("tcp", grpcAddr)
	if err != nil {
		return nil, err
	}

	go func() {
		if err := grpcServer.Serve(lis); err != nil {
			logger.Error("failed to serve grpc server", "err", err)
		}
	}()
	logger.Info("GRPC Server started", "addr", grpcAddr)

	return srv, nil
}

func (s *Server) writeFile(path string, content []byte) error {
	localPath := filepath.Join(s.config.Name, path)

	parentDir := filepath.Dir(localPath)
	if err := os.MkdirAll(parentDir, 0700); err != nil {
		return err
	}
	if err := ioutil.WriteFile(localPath, []byte(content), 0644); err != nil {
		return err
	}
	return nil
}

func (s *Server) Stop() {
	/*
		// stop all servers
		for _, node := range s.nodes {
			if err := node.Stop(); err != nil {
				s.logger.Error("failed to stop node", "id", "x", "err", err)
			}
		}
	*/
	if err := s.fileLogger.Close(); err != nil {
		s.logger.Error("failed to close file logger", "err", err.Error())
	}
}

func (s *Server) filterLocked(cond func(opts *nodeOpts) bool) []*node {
	res := []*node{}
	for _, i := range s.nodes {
		if cond(i.opts) {
			res = append(res, i)
		}
	}
	return res
}

func (s *Server) DeployNode(ctx context.Context, req *proto.DeployNodeRequest) (*proto.DeployNodeResponse, error) {
	s.lock.Lock()
	defer s.lock.Unlock()

	bCfg := &BeaconConfig{
		Spec: s.config.Spec,
		Eth1: s.eth1.node.GetAddr(NodePortEth1Http),
	}
	if len(s.nodes) != 0 {
		client := httpClient{
			addr: s.nodes[0].GetAddr(NodePortHttp),
		}
		identity, err := client.NodeIdentity()
		if err != nil {
			return nil, fmt.Errorf("cannto get a bootnode: %v", err)
		}
		bCfg.Bootnode = identity.ENR
	}
	var beaconFactory CreateBeacon2
	switch req.NodeClient {
	case proto.NodeClient_Teku:
		beaconFactory = NewTekuBeacon
	case proto.NodeClient_Prysm:
		beaconFactory = NewPrysmBeacon
	case proto.NodeClient_Lighthouse:
		beaconFactory = NewLighthouseBeacon
	default:
		return nil, fmt.Errorf("beacon type %s not found", req.NodeClient)
	}

	indx := len(s.nodes)

	// generate a name
	name := fmt.Sprintf("beacon-%s-%d", strings.ToLower(req.NodeClient.String()), indx)

	fLogger, err := s.fileLogger.Create(name)
	if err != nil {
		return nil, err
	}
	opts, err := beaconFactory(bCfg)
	if err != nil {
		return nil, err
	}

	labels := map[string]string{
		"name":     name,
		"type":     "beacon",
		"ensemble": s.config.Name,
	}
	genOpts := []nodeOption{
		WithName(name),
		WithOutput(fLogger),
		WithLabels(labels),
	}
	genOpts = append(genOpts, opts...)

	node, err := newNode(genOpts...)
	if err != nil {
		return nil, err
	}

	s.nodes = append(s.nodes, node)
	return &proto.DeployNodeResponse{}, nil
}

func (s *Server) DeployValidator(ctx context.Context, req *proto.DeployValidatorRequest) (*proto.DeployValidatorResponse, error) {
	s.lock.Lock()
	defer s.lock.Unlock()

	if req.NumValidators == 0 {
		return nil, fmt.Errorf("no validators provided")
	}

	beacons := s.filterLocked(func(opts *nodeOpts) bool {
		return opts.NodeType == proto.NodeType_Beacon && opts.NodeClient == req.NodeClient
	})
	if len(beacons) == 0 {
		return nil, fmt.Errorf("no beacon node found for client %s", req.NodeClient.String())
	}

	beacon := beacons[0]

	fmt.Println("- make deposits -")

	var accounts []*Account
	for i := 0; i < int(req.NumValidators); i++ {
		accounts = append(accounts, NewAccount())
	}
	if err := s.eth1.MakeDeposits(accounts, s.config.Spec.GetChainConfig()); err != nil {
		return nil, err
	}

	fmt.Println("- deposit is done -")

	indx := len(s.nodes)

	// generate a name
	name := fmt.Sprintf("validator-%s-%d", strings.ToLower(req.NodeClient.String()), indx)

	fLogger, err := s.fileLogger.Create(name)
	if err != nil {
		return nil, err
	}

	vCfg := &ValidatorConfig{
		Accounts: accounts,
		Spec:     s.config.Spec,
		Beacon:   beacon,
	}

	var validatorFactory CreateValidator2
	switch req.NodeClient {
	case proto.NodeClient_Teku:
		validatorFactory = NewTekuValidator
	case proto.NodeClient_Prysm:
		validatorFactory = NewPrysmValidator
	case proto.NodeClient_Lighthouse:
		validatorFactory = NewLighthouseValidator
	default:
		return nil, fmt.Errorf("validator client %s not found", req.NodeClient)
	}

	opts, err := validatorFactory(vCfg)
	if err != nil {
		return nil, err
	}
	genOpts := []nodeOption{
		WithName(name),
		WithOutput(fLogger),
	}
	genOpts = append(genOpts, opts...)

	node, err := newNode(genOpts...)
	if err != nil {
		return nil, err
	}
	s.nodes = append(s.nodes, node)
	return &proto.DeployValidatorResponse{}, nil
}

type httpClient struct {
	addr string
}

func (h *httpClient) get(url string, objResp interface{}) error {
	fullURL := h.addr + url

	resp, err := http.Get(fullURL)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	var obj struct {
		Data json.RawMessage
	}
	if err := json.Unmarshal(data, &obj); err != nil {
		return err
	}

	if err := json.Unmarshal(obj.Data, &objResp); err != nil {
		return err
	}
	return nil
}

type Identity struct {
	PeerID string `json:"peer_id"`
	ENR    string `json:"enr"`
}

func (h *httpClient) NodeIdentity() (*Identity, error) {
	var out *Identity
	err := h.get("/eth/v1/node/identity", &out)
	return out, err
}

type fileLogger struct {
	path string

	lock  sync.Mutex
	files []*os.File
}

func (f *fileLogger) Create(name string) (io.Writer, error) {
	f.lock.Lock()
	defer f.lock.Unlock()

	file, err := os.OpenFile(filepath.Join(f.path, name+".log"), os.O_RDWR|os.O_APPEND|os.O_CREATE, 0660)
	if err != nil {
		return nil, err
	}
	if len(f.files) == 0 {
		f.files = []*os.File{}
	}
	f.files = append(f.files, file)
	return file, nil
}

func (f *fileLogger) Close() error {
	for _, file := range f.files {
		// TODO, improve
		if err := file.Close(); err != nil {
			return err
		}
	}
	return nil
}
