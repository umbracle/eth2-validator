package slashing

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"

	"github.com/umbracle/eth2-validator/internal/server/proto"
)

type accountDuties struct {
	PubKey            string               `json:"pubkey"`
	SignedBlocks      []*signedBlock       `json:"signed_blocks"`
	SignedAttestation []*signedAttestation `json:"signed_attestations"`
}

type signedBlock proto.Duty

func (s *signedBlock) MarshalJSON() ([]byte, error) {
	data := map[string]interface{}{
		"slot":         s.Slot,
		"signing_root": "0x" + hex.EncodeToString(s.Result.BlockProposal.Root),
	}
	return json.Marshal(data)
}

type signedAttestation proto.Duty

func (s *signedAttestation) MarshalJSON() ([]byte, error) {
	data := map[string]interface{}{
		"source_epoch": s.Result.Attestation.Source.Epoch,
		"target_epoch": s.Result.Attestation.Target.Epoch,
		"signing_root": "0x" + hex.EncodeToString(s.Result.Attestation.Root),
	}
	return json.Marshal(data)
}

type metadata struct {
	FormatVersion         string `json:"interchange_format_version"`
	GenesisValidatorsRoot string `json:"genesis_validators_root"`
}

type eip3076Format struct {
	Metadata *metadata        `json:"metadata"`
	Data     []*accountDuties `json:"data"`
}

type Input struct {
	GenesisValidatorRoot []byte
	Duties               []*proto.Duty
}

// Format formats a list of block and attestation duties to the EIP-3076
// slashing format
func Format(input *Input) ([]byte, error) {

	dutiesByAccount := map[string]*accountDuties{}
	for indx, duty := range input.Duties {
		typ := duty.Type()

		// ensure there are no duties different than block and attestation
		if typ != proto.DutyBlockProposal && typ != proto.DutyAttestation {
			return nil, fmt.Errorf("duty '%d' is not of correct type: %s", indx, typ.String())
		}

		var w *accountDuties
		var ok bool

		if w, ok = dutiesByAccount[duty.PubKey]; !ok {
			w = &accountDuties{
				PubKey:            duty.PubKey,
				SignedBlocks:      []*signedBlock{},
				SignedAttestation: []*signedAttestation{},
			}
			dutiesByAccount[duty.PubKey] = w
		}

		if typ == proto.DutyBlockProposal {
			block := signedBlock(*duty)
			w.SignedBlocks = append(w.SignedBlocks, &block)
		} else {
			attestation := signedAttestation(*duty)
			w.SignedAttestation = append(w.SignedAttestation, &attestation)
		}
	}

	// sort the sortedAccounts to return a deterministic output
	sortedAccounts := []string{}
	for acct := range dutiesByAccount {
		sortedAccounts = append(sortedAccounts, acct)
	}
	sort.Strings(sortedAccounts)

	sortedDuties := []*accountDuties{}
	for _, acct := range sortedAccounts {
		sortedDuties = append(sortedDuties, dutiesByAccount[acct])
	}

	f := &eip3076Format{
		Metadata: &metadata{
			FormatVersion:         "5",
			GenesisValidatorsRoot: "0x" + hex.EncodeToString(input.GenesisValidatorRoot[:]),
		},
		Data: sortedDuties,
	}
	res, err := json.Marshal(f)
	if err != nil {
		return nil, err
	}
	return res, nil
}
