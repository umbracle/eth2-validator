package testutil

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"

	"github.com/hashicorp/go-hclog"
	"github.com/umbracle/eth2-validator/internal/testutil/proto"
	"google.golang.org/grpc"
)

type Config struct {
}

type Server struct {
	proto.UnimplementedE2EServiceServer

	logger hclog.Logger
	Spec   *Eth2Spec
	eth1   *Eth1Server

	nodes []Node
}

func NewServer(logger hclog.Logger) (*Server, error) {
	eth1, err := NewEth1Server()
	if err != nil {
		return nil, err
	}

	logger.Info("eth1 server deployed", "addr", eth1.GetAddr(NodePortEth1Http))
	logger.Info("deposit contract deployed", "addr", eth1.deposit.String())

	spec := &Eth2Spec{
		DepositContract:       eth1.deposit.String(),
		GenesisValidatorCount: 10,
	}

	srv := &Server{
		logger: logger,
		eth1:   eth1,
		Spec:   spec,
		nodes:  []Node{},
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

func (s *Server) Stop() {
	// stop all servers
}

func (s *Server) DeployNode(ctx context.Context, req *proto.DeployNodeRequest) (*proto.DeployNodeResponse, error) {
	bCfg := &BeaconConfig{
		Spec: s.Spec,
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
	node, err := NewTekuBeacon(bCfg)
	if err != nil {
		return nil, err
	}
	s.nodes = append(s.nodes, node)
	return &proto.DeployNodeResponse{}, nil
}

func (s *Server) DeployValidator(ctx context.Context, req *proto.DeployValidatorRequest) (*proto.DeployValidatorResponse, error) {

	var accounts []*Account
	for i := 0; i < 10; i++ {
		accounts = append(accounts, NewAccount())
	}
	if err := s.eth1.MakeDeposits(accounts, s.Spec.GetChainConfig()); err != nil {
		return nil, err
	}

	vCfg := &ValidatorConfig{
		Accounts: accounts,
		Spec:     s.Spec,
		Beacon:   s.nodes[0],
	}
	v, err := NewTekuValidator(vCfg)
	if err != nil {
		return nil, err
	}
	fmt.Println(v)

	return nil, nil
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
