package deposit

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/umbracle/eth2-validator/internal/beacon"
	"github.com/umbracle/eth2-validator/internal/bls"
	"github.com/umbracle/eth2-validator/internal/server/structs"
	"github.com/umbracle/go-web3"
	"github.com/umbracle/go-web3/jsonrpc"
	"github.com/umbracle/go-web3/testutil"
)

func TestDeposit_Signing(t *testing.T) {
	config, err := beacon.ReadChainConfig("mainnet")
	if err != nil {
		t.Fatal(err)
	}

	kk := bls.NewRandomKey()
	data, err := Input(kk, nil, web3.Gwei(MinGweiAmount).Uint64(), config)
	if err != nil {
		t.Fatal(err)
	}

	pub := &bls.PublicKey{}
	if err := pub.UnmarshalSSZ(data.Pubkey); err != nil {
		t.Fatal(err)
	}

	sig := &bls.Signature{}
	if err := sig.UnmarshalSSZ(data.Signature); err != nil {
		t.Fatal(err)
	}

	deposit := structs.DepositMessage{
		Pubkey:                data.Pubkey,
		Amount:                data.Amount,
		WithdrawalCredentials: data.WithdrawalCredentials,
	}
	root, err := SigningData(&deposit, config)
	if err != nil {
		t.Fatal(err)
	}

	if !sig.VerifyByte(pub, root[:]) {
		t.Fatal("bad signature")
	}
}

func TestDeposit_EndToEnd(t *testing.T) {
	server := testutil.NewTestServer(t, nil)
	defer server.Close()

	// deploy the contract
	receipt, err := server.SendTxn(&web3.Transaction{
		Input: DepositBin(),
	})
	assert.NoError(t, err)

	client, _ := jsonrpc.NewClient(server.HTTPAddr())
	code, err := client.Eth().GetCode(receipt.ContractAddress, web3.Latest)
	assert.NoError(t, err)
	assert.NotEqual(t, code, "0x")

	// sign the deposit
	config, err := beacon.ReadChainConfig("mainnet")
	assert.NoError(t, err)

	key := bls.NewRandomKey()

	input, err := Input(key, nil, web3.Gwei(MinGweiAmount).Uint64(), config)
	assert.NoError(t, err)

	// deploy transaction
	contract := NewDeposit(receipt.ContractAddress, client)
	contract.Contract().SetFrom(server.Account(0))

	txn := contract.Deposit(input.Pubkey, input.WithdrawalCredentials, input.Signature, input.Root)
	txn.SetValue(web3.Ether(MinGweiAmount))

	assert.NoError(t, txn.Do())
	assert.NoError(t, txn.Wait())

	// query the contract
	count, err := contract.GetDepositCount()
	assert.NoError(t, err)
	assert.Equal(t, int(count[0]), 1)
}
