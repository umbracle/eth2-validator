package testutil

import (
	"bytes"
	"encoding"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/ory/dockertest/v3"
	"github.com/ory/dockertest/v3/docker"
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
	Files      map[string]string
}

type node struct {
	pool     *dockertest.Pool
	resource *dockertest.Resource
	ip       string
}

type nodeOption func(*nodeOpts)

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
		Mount: []string{},
		Files: map[string]string{},
		Cmd:   []string{},
	}
	for _, opt := range opts {
		opt(nOpts)
	}

	pool, err := dockertest.NewPool("")
	if err != nil {
		return nil, fmt.Errorf("could not connect to docker: %s", err)
	}

	dockerOpts := &dockertest.RunOptions{
		Repository: nOpts.Repository,
		Tag:        nOpts.Tag,
		Cmd:        nOpts.Cmd,
		Mounts:     []string{},
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
		dockerOpts.Mounts = append(dockerOpts.Mounts, tmpDir+":"+mount)
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

	resource, err := pool.RunWithOptions(dockerOpts)
	if err != nil {
		return nil, fmt.Errorf("could not start node: %s", err)
	}

	n := &node{
		pool:     pool,
		resource: resource,
		ip:       resource.Container.NetworkSettings.IPAddress,
	}
	if nOpts.Retry != nil {
		if err := pool.Retry(func() error {
			return nOpts.Retry(n)
		}); err != nil {
			return nil, err
		}
	}

	return n, nil
}

func (n *node) Stop() {
	n.pool.Purge(n.resource)
}

func (n *node) GetLogs() (string, error) {
	wr := bytes.NewBuffer(nil)

	logsOptions := docker.LogsOptions{
		Container:    n.resource.Container.ID,
		OutputStream: wr,
		ErrorStream:  wr,
		Follow:       false,
		Stdout:       true,
		Stderr:       true,
	}
	if err := n.pool.Client.Logs(logsOptions); err != nil {
		return "", err
	}

	logs := wr.String()
	return logs, nil
}

func (n *node) IP() string {
	return n.ip
}
