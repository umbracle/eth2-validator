package bitlist

import (
	ssz "github.com/ferranbt/fastssz"
)

// BitList is a bitlist
type BitList []byte

// UnmarshalSSZ implements the fastssz.Unmarshaler interface
func (b *BitList) UnmarshalSSZ(buf []byte) error {
	return nil
}

// SizeSSZ implements the fastssz.SizeSSZ interface
func (b *BitList) SizeSSZ() int {
	return 0
}

// MarshalSSZTo implements the fastssz.MarshalSSZTo interface
func (b *BitList) MarshalSSZTo(dst []byte) ([]byte, error) {
	return nil, nil
}

// HashTreeRootWith implements the fastssz.HashRoot interface
func (b *BitList) HashTreeRootWith(hh *ssz.Hasher) error {
	return nil
}

// BitAt returns the bit at a given position
func (b *BitList) BitAt(indx int) bool {
	return false
}

// SetBitAt sets the bit at a given position.
func (b *BitList) SetBitAt(indx int, val bool) {

}
