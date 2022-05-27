package testutil

import (
	"fmt"
	"regexp"
)

var (
	bootnodeRegexp = regexp.MustCompile("\"Running bootnode: enr:(.*)\"")
)

type Bootnode struct {
	*node

	Enr string
}

func NewBootnode() (*Bootnode, error) {
	decodeEnr := func(node *node) (string, error) {
		logs, err := node.GetLogs()
		if err != nil {
			return "", err
		}
		match := bootnodeRegexp.FindStringSubmatch(logs)
		if len(match) == 0 {
			// not found
			return "", fmt.Errorf("not found")
		} else {
			return "enr:" + match[1], nil
		}
	}

	cmd := []string{
		"--debug",
		"--external-ip", "127.0.0.1",
		"--discv5-port", "3000",
	}

	nodeENR := ""
	opts := []nodeOption{
		WithName("bootnode"),
		WithCmd(cmd),
		WithContainer("gcr.io/prysmaticlabs/prysm/bootnode", "latest"),
		WithHostNetwork(),
		WithRetry(func(n *node) error {
			enr, err := decodeEnr(n)
			if err != nil {
				return err
			}
			nodeENR = enr
			return nil
		}),
	}

	node, err := newNode(opts...)
	if err != nil {
		return nil, err
	}
	b := &Bootnode{
		Enr:  nodeENR,
		node: node,
	}
	return b, nil
}
