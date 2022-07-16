package beacon

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestConfig_ReadMainnet(t *testing.T) {
	config, err := ReadChainConfig("")
	assert.NoError(t, err)
	assert.NotZero(t, config.SlotsPerEpoch)
}
