package beacon

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestConfig_ReadMainnet(t *testing.T) {
	_, err := ReadChainConfig("")
	assert.NoError(t, err)
}
