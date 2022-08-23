package slashing

import (
	"bytes"
	"embed"
	"encoding/hex"
	"encoding/json"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/umbracle/eth2-validator/internal/server/proto"
)

//go:embed testcases/*
var testcases embed.FS

func mustEncodeRoot(s string) []byte {
	buf, err := hex.DecodeString(strings.TrimPrefix(s, "0x"))
	if err != nil {
		panic(err)
	}
	return buf
}

func TestFormat(t *testing.T) {
	input := &Input{
		GenesisValidatorRoot: mustEncodeRoot("0x04700007fabc8282644aed6d1c7c9e21d38a03a0c4ba193f3afe428824b3a673"),
		Duties: []*proto.Duty{
			{
				PubKey: "0xb845089a1457f811bfc000588fbb4e713669be8ce060ea6be3c6ece09afc3794106c91ca73acda5e5457122d58723bed",
				Slot:   81952,
				Job:    &proto.Duty_BlockProposal_{},
				Result: &proto.Duty_Result{
					BlockProposal: &proto.Duty_BlockProposalResult{
						Root: mustEncodeRoot("0x4ff6f743a43f3b4f95350831aeaf0a122a1a392922c45d804280284a69eb850b"),
					},
				},
			},
			{
				PubKey: "0xb845089a1457f811bfc000588fbb4e713669be8ce060ea6be3c6ece09afc3794106c91ca73acda5e5457122d58723bed",
				Job:    &proto.Duty_Attestation_{},
				Result: &proto.Duty_Result{
					Attestation: &proto.Duty_AttestationResult{
						Root: mustEncodeRoot("0x587d6a4f59a58fe24f406e0502413e77fe1babddee641fda30034ed37ecc884d"),
						Source: &proto.Duty_AttestationResult_Checkpoint{
							Epoch: 2290,
						},
						Target: &proto.Duty_AttestationResult_Checkpoint{
							Epoch: 3007,
						},
					},
				},
			},
		},
	}

	res, err := Format(input)
	require.NoError(t, err)

	cc, err := testcases.ReadFile("testcases/eip3076.json")
	require.NoError(t, err)

	require.True(t, bytes.Equal(res, cleanJsonFile(cc)))
}

func cleanJsonFile(in []byte) []byte {
	buffer := new(bytes.Buffer)
	if err := json.Compact(buffer, in); err != nil {
		panic(err)
	}
	return buffer.Bytes()
}
