package bls

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"fmt"

	uuid "github.com/hashicorp/go-uuid"
	"github.com/umbracle/go-web3/keystore"
)

// Key is a reference to a key in the keymanager
type Key struct {
	Id  string
	Pub *PublicKey
	Prv *SecretKey
}

func (k *Key) Unmarshal(data []byte) error {
	k.Prv = &SecretKey{}
	if err := k.Prv.Unmarshal(data); err != nil {
		return err
	}
	k.Pub = k.Prv.GetPublicKey()
	return nil
}

func (k *Key) Marshal() ([]byte, error) {
	return k.Prv.Marshal()
}

func (k *Key) Equal(kk *Key) bool {
	return bytes.Equal(k.Pub.Serialize(), kk.Pub.Serialize())
}

func (k *Key) PubKey() (out [48]byte) {
	copy(out[:], k.Pub.Serialize())
	return
}

func (k *Key) Sign(root [32]byte) ([]byte, error) {
	signed := k.Prv.Sign(root[:])
	return signed.Serialize(), nil
}

func NewKeyFromPriv(priv []byte) (*Key, error) {
	k := &Key{}
	if err := k.Unmarshal(priv); err != nil {
		return nil, err
	}
	return k, nil
}

func NewRandomKey() *Key {
	sec := RandomKey()
	id, _ := uuid.GenerateUUID()

	k := &Key{
		Id:  id,
		Prv: sec,
		Pub: sec.GetPublicKey(),
	}
	return k
}

func FromKeystore(content []byte, password string) (*Key, error) {
	var dec map[string]interface{}
	if err := json.Unmarshal(content, &dec); err != nil {
		return nil, err
	}

	priv, err := keystore.DecryptV4(content, password)
	if err != nil {
		return nil, err
	}
	key, err := NewKeyFromPriv(priv)
	if err != nil {
		return nil, err
	}

	pub := key.PubKey()
	if hex.EncodeToString(pub[:]) != dec["pubkey"] {
		return nil, fmt.Errorf("pub key does not match")
	}
	key.Id = dec["uuid"].(string)
	return key, nil
}

func ToKeystore(k *Key, password string) ([]byte, error) {
	priv, err := k.Prv.Marshal()
	if err != nil {
		return nil, err
	}

	keystore, err := keystore.EncryptV4(priv, password)
	if err != nil {
		return nil, err
	}

	var dec map[string]interface{}
	if err := json.Unmarshal(keystore, &dec); err != nil {
		return nil, err
	}
	dec["pubkey"] = hex.EncodeToString(k.Pub.Serialize())
	dec["uuid"] = k.Id

	// small error here, params is set to nil in ethgo
	a := dec["crypto"].(map[string]interface{})
	b := a["checksum"].(map[string]interface{})
	b["params"] = map[string]interface{}{}

	return json.Marshal(dec)
}
