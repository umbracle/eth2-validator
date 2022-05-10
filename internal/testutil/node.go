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
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/api/types/strslice"
	"github.com/docker/docker/client"
	"github.com/hashicorp/go-hclog"
)

const (
	eth2ApiPort = "5050"
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
}

type node struct {
	cli  *client.Client
	id   string
	opts *nodeOpts
	ip   string
}

type nodeOption func(*nodeOpts)

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

	config := &container.Config{
		Image: imageName,
		Cmd:   strslice.StrSlice(nOpts.Cmd),
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
	containerID := body.ID

	// start container
	if err := cli.ContainerStart(ctx, containerID, types.ContainerStartOptions{}); err != nil {
		return nil, fmt.Errorf("could not start container: %v", err)
	}

	n := &node{
		cli:  cli,
		id:   containerID,
		opts: nOpts,
		ip:   "127.0.0.1",
	}

	// get the ip of the node if not running as a host network
	if !nOpts.InHost {
		containerData, err := cli.ContainerInspect(ctx, containerID)
		if err != nil {
			return nil, err
		}
		n.ip = containerData.NetworkSettings.IPAddress
	}

	if nOpts.Retry != nil {
		if err := retryFn(func() error {
			return nOpts.Retry(n)
		}); err != nil {
			return nil, err
		}
	}

	return n, nil
}

func (n *node) Stop() {
	if err := n.cli.ContainerStop(context.Background(), n.id, nil); err != nil {
		n.opts.Logger.Error("failed to stop container", "id", n.id, "err", err)
	}
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

func retryFn(handler func() error) error {
	timeoutT := time.NewTimer(1 * time.Minute)

	for {
		select {
		case <-time.After(5 * time.Second):
			if err := handler(); err == nil {
				return nil
			}

		case <-timeoutT.C:
			return fmt.Errorf("timeout")
		}
	}
}
