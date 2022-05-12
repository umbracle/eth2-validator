package bitlist

import (
	"math/bits"
)

// BitList is a bitlist
type BitList []byte

func NewBitlist(n uint64) BitList {
	ret := make(BitList, n/8+1)

	i := uint8(1 << (n % 8))
	ret[n/8] |= i

	return ret
}

// Len returns the length of the bitlist
func (b BitList) Len() uint64 {
	if len(b) == 0 {
		return 0
	}
	msb := bits.Len8(b[len(b)-1])
	if msb == 0 {
		return 0
	}
	return uint64(8*(len(b)-1) + msb - 1)
}

// SetBitAt sets the bit at a given position.
func (b BitList) SetBitAt(indx uint64, val bool) {
	if len := b.Len(); indx >= len {
		return
	}

	bit := uint8(1 << (indx % 8))
	if val {
		b[indx/8] |= bit
	} else {
		b[indx/8] &^= bit
	}
}

// BitAt returns the bit at a given position
func (b *BitList) BitAt(indx int) bool {
	return false
}
