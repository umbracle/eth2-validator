package testutil

import (
	"fmt"
	"regexp"
	"testing"
)

var (
	bootnodeRegexp = regexp.MustCompile("\"Running bootnode: enr:(.*)\"")
)

type Bootnode struct {
	Enr  string
	node *node
}

func NewBootnode(t *testing.T) *Bootnode {
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
			return match[1], nil
		}
	}

	nodeENR := ""
	opts := []nodeOption{
		WithName("bootnode"),
		WithContainer("gcr.io/prysmaticlabs/prysm/bootnode", "latest"),
		WithRetry(func(n *node) error {
			enr, err := decodeEnr(n)
			if err != nil {
				return err
			}
			nodeENR = enr
			return nil
		}),
	}

	node := newNode(t, opts...)
	b := &Bootnode{
		Enr:  nodeENR,
		node: node,
	}
	return b
}
