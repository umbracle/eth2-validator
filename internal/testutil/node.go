package testutil

import (
	"bytes"
	"context"
	"encoding"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"text/template"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/api/types/strslice"
	"github.com/docker/docker/client"
	"github.com/hashicorp/go-hclog"
)

type NodeClient string

const (
	Teku       NodeClient = "teku"
	Prysm      NodeClient = "prysm"
	Lighthouse NodeClient = "lighthouse"
)

type nodeOpts struct {
	Repository string
	Tag        string
	Cmd        []string
	Retry      func(n *node) error
	Name       string
	Mount      []string
	InHost     bool
	Files      map[string]string
	Logger     hclog.Logger
	Output     []io.Writer
	Labels     map[string]string
	NodeClient NodeClient
}

type exitResult struct {
	err error
}

type node struct {
	cli        *client.Client
	id         string
	opts       *nodeOpts
	ip         string
	ports      map[NodePort]uint64
	waitCh     chan struct{}
	exitResult *exitResult
}

type nodeOption func(*nodeOpts)

func WithNodeType(nodeClient NodeClient) nodeOption {
	return func(n *nodeOpts) {
		n.NodeClient = nodeClient
	}
}

func WithHostNetwork() nodeOption {
	return func(n *nodeOpts) {
		n.InHost = true
	}
}

func WithContainer(repository, tag string) nodeOption {
	return func(n *nodeOpts) {
		n.Repository = repository
		n.Tag = tag
	}
}

func WithCmd(cmd []string) nodeOption {
	return func(n *nodeOpts) {
		n.Cmd = append(n.Cmd, cmd...)
	}
}

func WithName(name string) nodeOption {
	return func(n *nodeOpts) {
		n.Name = name
	}
}

func WithRetry(retry func(n *node) error) nodeOption {
	return func(n *nodeOpts) {
		n.Retry = retry
	}
}

func WithMount(mount string) nodeOption {
	return func(n *nodeOpts) {
		n.Mount = append(n.Mount, mount)
	}
}

func WithOutput(output io.Writer) nodeOption {
	return func(n *nodeOpts) {
		n.Output = append(n.Output, output)
	}
}

func WithLabels(m map[string]string) nodeOption {
	return func(n *nodeOpts) {
		for k, v := range m {
			n.Labels[k] = v
		}
	}
}

func WithFile(path string, obj interface{}) nodeOption {
	return func(n *nodeOpts) {
		var data []byte
		var err error

		if objS, ok := obj.(string); ok {
			data = []byte(objS)
		} else if objB, ok := obj.([]byte); ok {
			data = objB
		} else if objT, ok := obj.(encoding.TextMarshaler); ok {
			data, err = objT.MarshalText()
		} else {
			data, err = json.Marshal(obj)
		}
		if err != nil {
			panic(err)
		}
		n.Files[path] = string(data)
	}
}

func newNode(opts ...nodeOption) (*node, error) {
	nOpts := &nodeOpts{
		Mount:  []string{},
		Files:  map[string]string{},
		Cmd:    []string{},
		Logger: hclog.L(),
		InHost: false,
		Output: []io.Writer{},
		Labels: map[string]string{},
	}
	for _, opt := range opts {
		opt(nOpts)
	}

	ctx := context.Background()

	cli, err := client.NewClientWithOpts(client.FromEnv)
	if err != nil {
		return nil, fmt.Errorf("could not connect to docker: %s", err)
	}

	// setup configuration
	dirPrefix := "node-"
	if nOpts.Name != "" {
		dirPrefix += nOpts.Name + "-"
	}

	// build any mount path
	mountMap := map[string]string{}
	for _, mount := range nOpts.Mount {
		tmpDir, err := ioutil.TempDir("/tmp", dirPrefix)
		if err != nil {
			return nil, err
		}
		mountMap[mount] = tmpDir
	}

	// build the files
	for path, content := range nOpts.Files {
		// find the mount match
		var mount, local string
		var found bool

	MOUNT:
		for mount, local = range mountMap {
			if strings.HasPrefix(path, mount) {
				found = true
				break MOUNT
			}
		}
		if !found {
			return nil, fmt.Errorf("mount match for '%s' not found", path)
		}

		relPath := strings.TrimPrefix(path, mount)
		localPath := filepath.Join(local, relPath)

		// create all the directory paths required
		parentDir := filepath.Dir(localPath)
		if err := os.MkdirAll(parentDir, 0700); err != nil {
			return nil, err
		}
		if err := ioutil.WriteFile(localPath, []byte(content), 0644); err != nil {
			return nil, err
		}
	}

	imageName := nOpts.Repository + ":" + nOpts.Tag

	// pull image if it does not exists
	_, _, err = cli.ImageInspectWithRaw(ctx, imageName)
	if err != nil {
		reader, err := cli.ImagePull(ctx, imageName, types.ImagePullOptions{})
		if err != nil {
			return nil, err
		}
		_, err = io.Copy(nOpts.Logger.StandardWriter(&hclog.StandardLoggerOptions{}), reader)
		if err != nil {
			return nil, err
		}
	}

	n := &node{
		cli:    cli,
		opts:   nOpts,
		ip:     "127.0.0.1",
		ports:  map[NodePort]uint64{},
		waitCh: make(chan struct{}),
	}

	// build CLI arguments which might include template arguments
	cmdArgs := []string{}
	for _, cmd := range nOpts.Cmd {
		cleanCmd, err := n.execCmd(cmd)
		if err != nil {
			return nil, err
		}
		cmdArgs = append(cmdArgs, cleanCmd)
	}

	config := &container.Config{
		Image:  imageName,
		Cmd:    strslice.StrSlice(cmdArgs),
		Labels: nOpts.Labels,
	}
	hostConfig := &container.HostConfig{
		Binds: []string{},
	}
	if nOpts.InHost {
		hostConfig.NetworkMode = "host"
	}

	for mount, local := range mountMap {
		hostConfig.Binds = append(hostConfig.Binds, local+":"+mount)
	}

	body, err := cli.ContainerCreate(ctx, config, hostConfig, &network.NetworkingConfig{}, nil, "")
	if err != nil {
		return nil, fmt.Errorf("could not create container: %v", err)
	}
	n.id = body.ID

	// start container
	if err := cli.ContainerStart(ctx, n.id, types.ContainerStartOptions{}); err != nil {
		return nil, fmt.Errorf("could not start container: %v", err)
	}

	go n.run()

	// get the ip of the node if not running as a host network
	if !nOpts.InHost {
		containerData, err := cli.ContainerInspect(ctx, n.id)
		if err != nil {
			return nil, err
		}
		n.ip = containerData.NetworkSettings.IPAddress
	}

	if len(nOpts.Output) != 0 {
		// track the logs to output
		go func() {
			if err := n.trackOutput(); err != nil {
				n.opts.Logger.Error("failed to log container", "id", n.id, "err", err)
			}
		}()
	}

	if nOpts.Retry != nil {
		if err := n.retryFn(func() error {
			return nOpts.Retry(n)
		}); err != nil {
			return nil, err
		}
	}

	return n, nil
}

type NodePort string

const (
	// NodePortEth1Http is the http port for the eth1 node.
	NodePortEth1Http = "eth1.http"

	// NodePortP2P is the p2p port for an eth2 node.
	NodePortP2P = "eth2.p2p"

	// NodePortHttp is the http port for an eth2 node.
	NodePortHttp = "eth2.http"

	// NodePortPrysmGrpc is the specific prysm port for its Grpc server
	NodePortPrysmGrpc = "eth2.prysm.grpc"
)

func uintPtr(i uint64) *uint64 {
	return &i
}

// port ranges for each node value.
var ports = map[NodePort]*uint64{
	NodePortEth1Http:  uintPtr(8000),
	NodePortP2P:       uintPtr(5000),
	NodePortHttp:      uintPtr(7000),
	NodePortPrysmGrpc: uintPtr(4000),
}

func (n *node) execCmd(cmd string) (string, error) {
	t := template.New("node_cmd")
	t.Funcs(template.FuncMap{
		"Port": func(name NodePort) string {
			port, ok := ports[name]
			if !ok {
				panic(fmt.Sprintf("Port '%s' not found", name))
			}

			relPort := atomic.AddUint64(port, 1)
			n.ports[name] = relPort
			return fmt.Sprintf("%d", relPort)
		},
	})

	t, err := t.Parse(cmd)
	if err != nil {
		return "", err
	}

	var buf bytes.Buffer
	if err := t.Execute(&buf, nil); err != nil {
		return "", err
	}
	return buf.String(), nil
}

func (n *node) WaitCh() <-chan struct{} {
	return n.waitCh
}

func (n *node) run() {
	resCh, errCh := n.cli.ContainerWait(context.Background(), n.id, container.WaitConditionNotRunning)

	var exitErr error
	select {
	case res := <-resCh:
		if res.Error != nil {
			exitErr = fmt.Errorf(res.Error.Message)
		}
	case err := <-errCh:
		exitErr = err
	}

	n.exitResult = &exitResult{
		err: exitErr,
	}
	close(n.waitCh)
}

func (n *node) GetAddr(port NodePort) string {
	num, ok := n.ports[port]
	if !ok {
		panic(fmt.Sprintf("port '%s' not found", port))
	}
	return fmt.Sprintf("http://%s:%d", n.ip, num)
}

func (n *node) Stop() {
	if err := n.cli.ContainerStop(context.Background(), n.id, nil); err != nil {
		n.opts.Logger.Error("failed to stop container", "id", n.id, "err", err)
	}
}

func (n *node) trackOutput() error {
	writer := io.MultiWriter(n.opts.Output...)

	opts := types.ContainerLogsOptions{
		ShowStdout: true,
		ShowStderr: true,
		Follow:     true,
	}
	out, err := n.cli.ContainerLogs(context.Background(), n.id, opts)
	if err != nil {
		return err
	}
	if _, err := io.Copy(writer, out); err != nil {
		return err
	}
	return nil
}

func (n *node) GetLogs() (string, error) {
	wr := bytes.NewBuffer(nil)

	out, err := n.cli.ContainerLogs(context.Background(), n.id, types.ContainerLogsOptions{ShowStdout: true, ShowStderr: true})
	if err != nil {
		return "", nil
	}
	if _, err := io.Copy(wr, out); err != nil {
		return "", err
	}
	logs := wr.String()
	return logs, nil
}

func (n *node) IP() string {
	return n.ip
}

func (n *node) Type() NodeClient {
	return n.opts.NodeClient
}

func (n *node) retryFn(handler func() error) error {
	timeoutT := time.NewTimer(1 * time.Minute)

	for {
		select {
		case <-time.After(5 * time.Second):
			if err := handler(); err == nil {
				return nil
			}

		case <-n.waitCh:
			return fmt.Errorf("node stopped")

		case <-timeoutT.C:
			return fmt.Errorf("timeout")
		}
	}
}

type Node interface {
	GetAddr(NodePort) string
	Type() NodeClient
	IP() string
	Stop()
}
