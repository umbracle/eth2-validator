package beacon

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestApi_Genesis(t *testing.T) {
	h := NewHttpAPI("http://172.17.0.3:5050")

	genesis, err := h.Genesis(context.Background())
	assert.NoError(t, err)

	fmt.Println(genesis)
}

func TestApi_Events(t *testing.T) {
	h := NewHttpAPI("http://172.17.0.3:5050")

	h.Events(context.Background(), []string{"head"}, func(obj interface{}) {
		fmt.Println("--")
	})
}
