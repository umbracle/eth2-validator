package proto

import (
	"strings"
)

func StringToNodeClient(str string) (NodeClient, bool) {
	found, ok := NodeClient_value[strings.Title(str)]
	if !ok {
		return 0, false
	}
	return NodeClient(found), true
}
