package testutil

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBootnode(t *testing.T) {
	b, err := NewBootnode()
	assert.NoError(t, err)
	assert.NotEmpty(t, b.Enr)
}
