package structs

import (
	"encoding/hex"
	"fmt"
	"strings"
)

type Domain [4]byte

func (d *Domain) UnmarshalText(data []byte) error {
	domainStr := string(data)
	if !strings.HasPrefix(domainStr, "0x") {
		return fmt.Errorf("not prefixed")
	}
	buf, err := hex.DecodeString(domainStr[2:])
	if err != nil {
		return err
	}
	if len(buf) != 4 {
		return fmt.Errorf("bad size")
	}
	copy(d[:], buf)
	return nil
}

// UnmarshalYAML implements the Unmarshaler interface in yaml package
func (d *Domain) UnmarshalYAML(unmarshal func(interface{}) error) error {
	panic("xx")

	var domainStr string
	if err := unmarshal(&domainStr); err != nil {
		return fmt.Errorf("failed to unmarshal Domain: %v", err)
	}
	if !strings.HasPrefix(domainStr, "0x") {
		return fmt.Errorf("not prefixed")
	}
	buf, err := hex.DecodeString(domainStr[2:])
	if err != nil {
		return err
	}
	if len(buf) != 4 {
		return fmt.Errorf("bad size")
	}
	copy(d[:], buf)
	return nil
}

func (d Domain) MarshalText() ([]byte, error) {
	return []byte("0x" + hex.EncodeToString(d[:])), nil
}
