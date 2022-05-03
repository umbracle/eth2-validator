package deposit

import (
	ssz "github.com/ferranbt/fastssz"
	"github.com/umbracle/eth2-validator/internal/beacon"
	"github.com/umbracle/eth2-validator/internal/bls"
	"github.com/umbracle/eth2-validator/internal/server/structs"
	"github.com/umbracle/go-web3/abi"
)

const MinGweiAmount = uint64(320)

// DepositEvent is the eth2 deposit event
var DepositEvent = abi.MustNewEvent(`event DepositEvent(
	bytes pubkey,
	bytes whitdrawalcred,
	bytes amount,
	bytes signature,
	bytes index
)`)

func Input(depositKey *bls.Key, withdrawalKey *bls.Key, amountInGwei uint64, config *beacon.ChainConfig) (*structs.DepositData, error) {
	// withdrawalCredentialsHash forms a 32 byte hash of the withdrawal public address.
	//   withdrawal_credentials[:1] == BLS_WITHDRAWAL_PREFIX_BYTE
	//   withdrawal_credentials[1:] == hash(withdrawal_pubkey)[1:]
	// TODO

	unsignedMsgRoot, err := ssz.HashWithDefaultHasher(&structs.DepositMessage{
		Pubkey:                depositKey.Pub.Serialize(),
		Amount:                amountInGwei,
		WithdrawalCredentials: make([]byte, 32),
	})
	if err != nil {
		return nil, err
	}

	domain, err := config.ComputeDepositDomain(nil, nil)
	if err != nil {
		return nil, err
	}
	rootToSign, err := ssz.HashWithDefaultHasher(&structs.SigningData{
		ObjectRoot: unsignedMsgRoot[:],
		Domain:     domain,
	})
	if err != nil {
		return nil, err
	}

	signature, err := depositKey.Sign(rootToSign)
	if err != nil {
		return nil, err
	}

	msg := &structs.DepositData{
		Pubkey:                depositKey.Pub.Serialize(),
		Amount:                amountInGwei,
		WithdrawalCredentials: make([]byte, 32),
		Signature:             signature,
	}
	root, err := msg.HashTreeRoot()
	if err != nil {
		return nil, err
	}
	msg.Root = root
	return msg, nil
}

func SigningData(obj ssz.HashRoot, config *beacon.ChainConfig) ([32]byte, error) {
	unsignedMsgRoot, err := ssz.HashWithDefaultHasher(obj)
	if err != nil {
		return [32]byte{}, err
	}

	domain, err := config.ComputeDepositDomain(nil, nil)
	if err != nil {
		return [32]byte{}, err
	}
	root, err := ssz.HashWithDefaultHasher(&structs.SigningData{
		ObjectRoot: unsignedMsgRoot[:],
		Domain:     domain,
	})
	if err != nil {
		return [32]byte{}, err
	}
	return root, nil
}
