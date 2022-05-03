package bls

import (
	"crypto/rand"

	ssz "github.com/ferranbt/fastssz"
	blst "github.com/supranational/blst/bindings/go"
)

type blstPublicKey = blst.P1Affine
type blstSignature = blst.P2Affine

var dst = []byte("BLS_SIG_BLS12381G2_XMD:SHA-256_SSWU_RO_POP_")

// --- Signature ---

// Signature is a Bls signature
type Signature struct {
	sig *blstSignature
}

func (s *Signature) Deserialize(buf []byte) error {
	s.sig = new(blstSignature).Uncompress(buf)
	return nil
}

func (s *Signature) Serialize() []byte {
	return s.sig.Compress()
}

func (s *Signature) VerifyByte(pub *PublicKey, msg []byte) bool {
	return s.sig.Verify(false, pub.pub, false, msg, dst)
}

// UnmarshalSSZ implements the fastssz.Unmarshaler interface
func (s *Signature) UnmarshalSSZ(buf []byte) error {
	return s.Deserialize(buf)
}

// SizeSSZ implements the fastssz.SizeSSZ interface
func (s *Signature) SizeSSZ() int {
	return 96
}

// MarshalSSZTo implements the fastssz.MarshalSSZTo interface
func (s *Signature) MarshalSSZTo(dst []byte) ([]byte, error) {
	dst = append(dst, s.Serialize()...)
	return dst, nil
}

// HashTreeRootWith implements the fastssz.HashRoot interface
func (s *Signature) HashTreeRootWith(hh *ssz.Hasher) error {
	hh.PutBytes(s.Serialize())
	return nil
}

/// --- Public Key ---

// PublicKey is a Bls public key
type PublicKey struct {
	pub *blstPublicKey
}

func (p *PublicKey) Deserialize(buf []byte) error {
	p.pub = new(blstPublicKey).Uncompress(buf)
	return nil
}

func (p *PublicKey) Serialize() []byte {
	return p.pub.Compress()
}

// UnmarshalSSZ implements the fastssz.Unmarshaler interface
func (p *PublicKey) UnmarshalSSZ(buf []byte) error {
	return p.Deserialize(buf)
}

// SizeSSZ implements the fastssz.SizeSSZ interface
func (p *PublicKey) SizeSSZ() int {
	return 48
}

// MarshalSSZTo implements the fastssz.MarshalSSZTo interface
func (p *PublicKey) MarshalSSZTo(dst []byte) ([]byte, error) {
	dst = append(dst, p.Serialize()...)
	return dst, nil
}

// HashTreeRootWith implements the fastssz.HashRoot interface
func (p *PublicKey) HashTreeRootWith(hh *ssz.Hasher) error {
	hh.PutBytes(p.Serialize())
	return nil
}

// --- Secret Key ---

// SecretKey is a Bls secret key
type SecretKey struct {
	key *blst.SecretKey
}

func (s *SecretKey) Unmarshal(data []byte) error {
	s.key = new(blst.SecretKey).Deserialize(data)
	return nil
}

func (s *SecretKey) Marshal() ([]byte, error) {
	return s.key.Serialize(), nil
}

func (s *SecretKey) GetPublicKey() *PublicKey {
	pub := new(blstPublicKey).From(s.key)
	return &PublicKey{pub: pub}
}

func (s *SecretKey) Sign(msg []byte) *Signature {
	//res := new(blst.P2Affine).Sign(s.New, msg, dst)
	//sig := &Signature{new: *res}
	sig := new(blstSignature).Sign(s.key, msg, dst)
	return &Signature{sig: sig}
}

func RandomKey() *SecretKey {
	var ikm [32]byte
	_, _ = rand.Read(ikm[:])
	sk := blst.KeyGen(ikm[:])

	sec := &SecretKey{
		key: sk,
	}
	return sec
}
