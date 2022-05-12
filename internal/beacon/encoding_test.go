package beacon

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/umbracle/eth2-validator/internal/server/structs"
)

func TestHttpEncoding(t *testing.T) {

	mustDecode := func(str string) []byte {
		buf, err := hex.DecodeString(strings.TrimPrefix(str, "0x"))
		if err != nil {
			t.Fatal(err)
		}
		return buf
	}

	mustDecode32 := func(str string) (res [32]byte) {
		buf, err := hex.DecodeString(strings.TrimPrefix(str, "0x"))
		if err != nil {
			t.Fatal(err)
		}
		copy(res[:], buf)
		return
	}

	example := []byte(`{
		"slot": "20",
		"proposer_index": "0",
		"parent_root": "0x1d96d93260c726fc63cf8986c5cf8fde9096f032818dbacfd2186b531423401f",
		"state_root": "0xfd9a880fd6a30f1ef9aba66f86c5160829702a44332281e0028b760b05476847",
		"body": {
		  "randao_reveal": "0xb523efff78ebbf1a1d412f9829c674339db26e08d24b9b9622e21ba993720bf1724deab6017bf4acb6c83b5244d2833d12968b1e4e7b905eea93695fdf6e2358702497f41625bb67657a265ea9e397eb26fce4f5995d1e1d3aa28f0f27bed11b",
		  "eth1_data": {
			"deposit_root": "0xe2dc4e09f9e4e0d5d40495f2a313154454077291df75434460e74f630581cb49",
			"deposit_count": "1",
			"block_hash": "0xff3f75762e004ac6d8925d2f67ae686c778e26652c16b18978764e4f053716a5"
		  },
		  "graffiti": "0x74656b752f7632312e322e300000000000000000000000000000000000000000",
		  "proposer_slashings": [
			
		  ],
		  "attester_slashings": [
			
		  ],
		  "attestations": [
			
		  ],
		  "deposits": [
			
		  ],
		  "voluntary_exits": [
			
		  ]
		}
	  }`)

	var out *structs.BeaconBlock
	err := Unmarshal(example, &out)

	result := &structs.BeaconBlock{
		Slot:       20,
		ParentRoot: mustDecode("0x1d96d93260c726fc63cf8986c5cf8fde9096f032818dbacfd2186b531423401f"),
		StateRoot:  mustDecode("0xfd9a880fd6a30f1ef9aba66f86c5160829702a44332281e0028b760b05476847"),
		Body: &structs.BeaconBlockBody{
			RandaoReveal: mustDecode("0xb523efff78ebbf1a1d412f9829c674339db26e08d24b9b9622e21ba993720bf1724deab6017bf4acb6c83b5244d2833d12968b1e4e7b905eea93695fdf6e2358702497f41625bb67657a265ea9e397eb26fce4f5995d1e1d3aa28f0f27bed11b"),
			Eth1Data: &structs.Eth1Data{
				DepositRoot:  mustDecode("0xe2dc4e09f9e4e0d5d40495f2a313154454077291df75434460e74f630581cb49"),
				DepositCount: 1,
				BlockHash:    mustDecode("0xff3f75762e004ac6d8925d2f67ae686c778e26652c16b18978764e4f053716a5"),
			},
			Graffiti:          mustDecode32("0x74656b752f7632312e322e300000000000000000000000000000000000000000"),
			ProposerSlashings: []*structs.ProposerSlashing{},
			AttesterSlashings: []*structs.AttesterSlashing{},
			Attestations:      []*structs.Attestation{},
			Deposits:          []*structs.Deposit{},
			VoluntaryExits:    []*structs.SignedVoluntaryExit{},
		},
	}

	fmt.Println(out.Slot)
	fmt.Println(out.ParentRoot)
	fmt.Println(mustDecode("0x1d96d93260c726fc63cf8986c5cf8fde9096f032818dbacfd2186b531423401f"))
	fmt.Println(reflect.DeepEqual(result, out))
	assert.Equal(t, result, out)
	fmt.Println(err)

	data2, err := Marshal(result)
	assert.NoError(t, err)

	fmt.Println(string(data2))

	assert.NoError(t, compareJSON(data2, example))
}

func compareJSON(a, b []byte) error {
	decode := func(a []byte) (map[string]interface{}, error) {
		out := map[string]interface{}{}
		if err := json.Unmarshal(a, &out); err != nil {
			return nil, err
		}
		return out, nil
	}

	aMap, err := decode(a)
	if err != nil {
		return err
	}
	bMap, err := decode(b)
	if err != nil {
		return err
	}
	if !reflect.DeepEqual(aMap, bMap) {
		return fmt.Errorf("not equal")
	}
	return nil
}

func TestMarshalSimp(t *testing.T) {
	res, err := Marshal([]string{"1"})
	assert.NoError(t, err)
	assert.Equal(t, string(res), `["1"]`)
}
