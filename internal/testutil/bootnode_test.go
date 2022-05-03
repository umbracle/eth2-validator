package testutil

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBootnode(t *testing.T) {
	b := NewBootnode(t)
	assert.NotEmpty(t, b.Enr)
}
