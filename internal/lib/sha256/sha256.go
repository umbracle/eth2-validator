package sha256

import (
	"crypto/sha256"
	"hash"
	"sync"
)

// DefaultSHA256Pool is a default pool
var DefaultSHA256Pool Pool

// Pool is a pool of keccaks
type Pool struct {
	pool sync.Pool
}

// Get returns the keccak
func (p *Pool) Get() hash.Hash {
	v := p.pool.Get()
	if v == nil {
		return sha256.New()
	}
	return v.(hash.Hash)
}

// Put releases the keccak
func (p *Pool) Put(k hash.Hash) {
	k.Reset()
	p.pool.Put(k)
}

func Hash(b []byte) []byte {
	h := DefaultSHA256Pool.Get()
	h.Write(b)
	buf := h.Sum(nil)
	DefaultSHA256Pool.Put(h)
	return buf
}
