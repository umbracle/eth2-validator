package testutil

import (
	"bytes"
	"embed"
	"encoding/hex"
	"fmt"
	"html/template"
	"reflect"
	"strings"

	ssz "github.com/ferranbt/fastssz"
	"github.com/mitchellh/mapstructure"
	"github.com/umbracle/eth2-validator/internal/beacon"
	"gopkg.in/yaml.v2"
)

var (
	//go:embed fixtures
	res embed.FS
)

// Eth2Spec is the config of the Eth2.0 node
type Eth2Spec struct {
	GenesisValidatorCount     int
	GenesisDelay              int
	MinGenesisTime            int
	EthFollowDistance         int
	SecondsPerEth1Block       int
	EpochsPerEth1VotingPeriod int
	ShardCommitteePeriod      int
	NetworkID                 int
	SlotsPerEpoch             int
	SecondsPerSlot            int
	DepositContract           string
}

func (e *Eth2Spec) GetChainConfig() *beacon.ChainConfig {
	var config *beacon.ChainConfig
	if err := yaml.Unmarshal(e.buildConfig(), &config); err != nil {
		panic(err)
	}
	return config
}

func (e *Eth2Spec) MarshalText() ([]byte, error) {
	return e.buildConfig(), nil
}

func (e *Eth2Spec) buildConfig() []byte {
	// set up default config values
	if e.GenesisValidatorCount == 0 {
		e.GenesisValidatorCount = 1
	}
	if e.GenesisDelay == 0 {
		e.GenesisDelay = 10 // second
	}
	if e.EthFollowDistance == 0 {
		e.EthFollowDistance = 10 // blocks
	}
	if e.SecondsPerEth1Block == 0 {
		e.SecondsPerEth1Block = 1 // second
	}
	if e.EpochsPerEth1VotingPeriod == 0 {
		e.EpochsPerEth1VotingPeriod = 64
	}
	if e.ShardCommitteePeriod == 0 {
		e.ShardCommitteePeriod = 4
	}
	if e.NetworkID == 0 {
		e.NetworkID = 1337
	}
	if e.SlotsPerEpoch == 0 {
		e.SlotsPerEpoch = 12 // default 32 slots
	}
	if e.SecondsPerSlot == 0 {
		e.SecondsPerSlot = 3 // default 12 seconds
	}
	tmpl, err := template.ParseFS(res, "fixtures/config.yaml.tmpl")
	if err != nil {
		panic(fmt.Sprintf("BUG: Failed to load eth2 config template: %v", err))
	}

	var tpl bytes.Buffer
	if err = tmpl.Execute(&tpl, e); err != nil {
		panic(fmt.Sprintf("BUG: Failed to render template: %v", err))
	}
	return tpl.Bytes()
}

func isByteSlice(t reflect.Type) bool {
	return t.Kind() == reflect.Slice && t.Elem().Kind() == reflect.Uint8
}

func isByteArray(t reflect.Type) bool {
	return t.Kind() == reflect.Array && t.Elem().Kind() == reflect.Uint8
}

func customHook(f reflect.Type, t reflect.Type, data interface{}) (interface{}, error) {
	if f.Kind() != reflect.String {
		return data, nil
	}

	raw := data.(string)
	if !strings.HasPrefix(raw, "0x") {
		return nil, fmt.Errorf("0x prefix not found")
	}
	elem, err := hex.DecodeString(raw[2:])
	if err != nil {
		return nil, err
	}
	if isByteSlice(t) {
		// []byte
		return elem, nil
	}
	if isByteArray(t) {
		// [n]byte
		if t.Len() != len(elem) {
			return nil, fmt.Errorf("incorrect array length: %d %d", t.Len(), len(elem))
		}

		v := reflect.New(t)
		reflect.Copy(v.Elem(), reflect.ValueOf(elem))
		return v.Interface(), nil
	}

	var v reflect.Value
	if t.Kind() == reflect.Ptr {
		v = reflect.New(t.Elem())
	} else {
		v = reflect.New(t)
	}
	if vv, ok := v.Interface().(ssz.Unmarshaler); ok {
		if err := vv.UnmarshalSSZ(elem); err != nil {
			return nil, err
		}
		return vv, nil
	}
	return nil, fmt.Errorf("type not found")
}

// UnmarshalSSZTest decodes a spec tests in yaml format
func UnmarshalSSZTest(content []byte, result interface{}) error {
	var source map[string]interface{}
	if err := yaml.Unmarshal(content, &source); err != nil {
		return err
	}

	dc := &mapstructure.DecoderConfig{
		Result:     result,
		DecodeHook: customHook,
		TagName:    "json",
	}
	ms, err := mapstructure.NewDecoder(dc)
	if err != nil {
		return err
	}
	if err = ms.Decode(source); err != nil {
		return err
	}
	return nil
}
