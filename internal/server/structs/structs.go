package structs

type AggregateAndProof struct {
	Index          uint64       `json:"aggregator_index"`
	Aggregate      *Attestation `json:"aggregate"`
	SelectionProof []byte       `json:"selection_proof" ssz-size:"96"`
}

type Checkpoint struct {
	Epoch uint64   `json:"epoch"`
	Root  [32]byte `json:"root" ssz-size:"32"`
}

type Hash [32]byte

type AttestationData struct {
	Slot            uint64      `json:"slot"`
	Index           uint64      `json:"index"`
	BeaconBlockHash Hash        `json:"beacon_block_root" ssz-size:"32"`
	Source          *Checkpoint `json:"source"`
	Target          *Checkpoint `json:"target"`
}

type Attestation struct {
	AggregationBits []byte           `json:"aggregation_bits" ssz:"bitlist" ssz-max:"2048"`
	Data            *AttestationData `json:"data"`
	Signature       []byte           `json:"signature" ssz-size:"96"`
}

type DepositData struct {
	Pubkey                []byte   `json:"pubkey" ssz-size:"48"`
	WithdrawalCredentials []byte   `json:"withdrawal_credentials" ssz-size:"32"`
	Amount                uint64   `json:"amount"`
	Signature             []byte   `json:"signature" ssz-size:"96"`
	Root                  [32]byte `ssz:"-"`
}

type Deposit struct {
	Proof [][]byte `ssz-size:"33,32"`
	Data  *DepositData
}

type DepositMessage struct {
	Pubkey                []byte `json:"pubkey" ssz-size:"48"`
	WithdrawalCredentials []byte `json:"withdrawal_credentials" ssz-size:"32"`
	Amount                uint64 `json:"amount"`
}

type IndexedAttestation struct {
	AttestationIndices []uint64         `json:"attesting_indices" ssz-max:"2048"`
	Data               *AttestationData `json:"data"`
	Signature          []byte           `json:"signature" ssz-size:"96"`
}

type PendingAttestation struct {
	AggregationBits []byte           `json:"aggregation_bits" ssz:"bitlist" ssz-max:"2048"`
	Data            *AttestationData `json:"data"`
	InclusionDelay  uint64           `json:"inclusion_delay"`
	ProposerIndex   uint64           `json:"proposer_index"`
}

type Fork struct {
	PreviousVersion []byte `json:"previous_version" ssz-size:"4"`
	CurrentVersion  []byte `json:"current_version" ssz-size:"4"`
	Epoch           uint64 `json:"epoch"`
}

type Validator struct {
	Pubkey                     []byte `json:"pubkey" ssz-size:"48"`
	WithdrawalCredentials      []byte `json:"withdrawal_credentials" ssz-size:"32"`
	EffectiveBalance           uint64 `json:"effective_balance"`
	Slashed                    bool   `json:"slashed"`
	ActivationEligibilityEpoch uint64 `json:"activation_eligibility_epoch"`
	ActivationEpoch            uint64 `json:"activation_epoch"`
	ExitEpoch                  uint64 `json:"exit_epoch"`
	WithdrawableEpoch          uint64 `json:"withdrawable_epoch"`
}

type VoluntaryExit struct {
	Epoch          uint64 `json:"epoch"`
	ValidatorIndex uint64 `json:"validator_index"`
}

type SignedVoluntaryExit struct {
	Exit      *VoluntaryExit `json:"message"`
	Signature []byte         `json:"signature" ssz-size:"96"`
}

type Eth1Data struct {
	DepositRoot  []byte `json:"deposit_root" ssz-size:"32"`
	DepositCount uint64 `json:"deposit_count"`
	BlockHash    []byte `json:"block_hash" ssz-size:"32"`
}

type SigningRoot struct {
	ObjectRoot []byte `json:"object_root" ssz-size:"32"`
	Domain     []byte `json:"domain" ssz-size:"8"`
}

type ProposerSlashing struct {
	Header1 *SignedBeaconBlockHeader `json:"signed_header_1"`
	Header2 *SignedBeaconBlockHeader `json:"signed_header_2"`
}

type AttesterSlashing struct {
	Attestation1 *IndexedAttestation `json:"attestation_1"`
	Attestation2 *IndexedAttestation `json:"attestation_2"`
}

type BeaconBlock struct {
	Slot          uint64           `json:"slot"`
	ProposerIndex uint64           `json:"proposer_index"`
	ParentRoot    []byte           `json:"parent_root" ssz-size:"32"`
	StateRoot     []byte           `json:"state_root" ssz-size:"32"`
	Body          *BeaconBlockBody `json:"body"`
}

type SignedBeaconBlock struct {
	Block     *BeaconBlock `json:"message"`
	Signature []byte       `json:"signature" ssz-size:"96"`
}

type Transfer struct {
	Sender    uint64 `json:"sender"`
	Recipient uint64 `json:"recipient"`
	Amount    uint64 `json:"amount"`
	Fee       uint64 `json:"fee"`
	Slot      uint64 `json:"slot"`
	Pubkey    []byte `json:"pubkey" ssz-size:"48"`
	Signature []byte `json:"signature" ssz-size:"96"`
}

type BeaconBlockBody struct {
	RandaoReveal      []byte                 `json:"randao_reveal" ssz-size:"96"`
	Eth1Data          *Eth1Data              `json:"eth1_data"`
	Graffiti          [32]byte               `json:"graffiti" ssz-size:"32"`
	ProposerSlashings []*ProposerSlashing    `json:"proposer_slashings" ssz-max:"16"`
	AttesterSlashings []*AttesterSlashing    `json:"attester_slashings" ssz-max:"2"`
	Attestations      []*Attestation         `json:"attestations" ssz-max:"128"`
	Deposits          []*Deposit             `json:"deposits" ssz-max:"16"`
	VoluntaryExits    []*SignedVoluntaryExit `json:"voluntary_exits" ssz-max:"16"`
}

type SignedBeaconBlockHeader struct {
	Header    *BeaconBlockHeader `json:"message"`
	Signature []byte             `json:"signature" ssz-size:"96"`
}

type BeaconBlockHeader struct {
	Slot          uint64 `json:"slot"`
	ProposerIndex uint64 `json:"proposer_index"`
	ParentRoot    []byte `json:"parent_root" ssz-size:"32"`
	StateRoot     []byte `json:"state_root" ssz-size:"32"`
	BodyRoot      []byte `json:"body_root" ssz-size:"32"`
}

type ForkData struct {
	CurrentVersion        []byte `ssz-size:"4"`
	GenesisValidatorsRoot []byte `ssz-size:"32"`
}

type SigningData struct {
	ObjectRoot []byte `ssz-size:"32"`
	Domain     []byte `ssz-size:"32"`
}

// Altair fork

type SignedBeaconBlockAltair struct {
	Block     *BeaconBlockAltair `json:"message"`
	Signature []byte             `json:"signature" ssz-size:"96"`
}

type BeaconBlockAltair struct {
	Slot          uint64                 `json:"slot"`
	ProposerIndex uint64                 `json:"proposer_index"`
	ParentRoot    []byte                 `json:"parent_root" ssz-size:"32"`
	StateRoot     []byte                 `json:"state_root" ssz-size:"32"`
	Body          *BeaconBlockBodyAltair `json:"body"`
}

type BeaconBlockBodyAltair struct {
	RandaoReveal      []byte                 `json:"randao_reveal" ssz-size:"96"`
	Eth1Data          *Eth1Data              `json:"eth1_data"`
	Graffiti          [32]byte               `json:"graffiti" ssz-size:"32"`
	ProposerSlashings []*ProposerSlashing    `json:"proposer_slashings" ssz-max:"16"`
	AttesterSlashings []*AttesterSlashing    `json:"attester_slashings" ssz-max:"2"`
	Attestations      []*Attestation         `json:"attestations" ssz-max:"128"`
	Deposits          []*Deposit             `json:"deposits" ssz-max:"16"`
	VoluntaryExits    []*SignedVoluntaryExit `json:"voluntary_exits" ssz-max:"16"`
	SyncAggregate     *SyncAggregate         `json:"sync_aggregate"`
}

type SyncAggregate struct {
	SyncCommiteeBits      []byte   `json:"sync_committee_bits" ssz-size:"64"`
	SyncCommiteeSignature [96]byte `json:"sync_committee_signature" ssz-size:"96"`
}

type SyncCommittee struct {
	PubKeys         [][]byte `json:"pubkeys" ssz-size:"512,48"`
	AggregatePubKey [48]byte `json:"aggregate_pubkey" ssz-size:"48"`
}

// bellatrix

type BeaconBlockBodyBellatrix struct {
	RandaoReveal      []byte                 `json:"randao_reveal" ssz-size:"96"`
	Eth1Data          *Eth1Data              `json:"eth1_data"`
	Graffiti          [32]byte               `json:"graffiti" ssz-size:"32"`
	ProposerSlashings []*ProposerSlashing    `json:"proposer_slashings" ssz-max:"16"`
	AttesterSlashings []*AttesterSlashing    `json:"attester_slashings" ssz-max:"2"`
	Attestations      []*Attestation         `json:"attestations" ssz-max:"128"`
	Deposits          []*Deposit             `json:"deposits" ssz-max:"16"`
	VoluntaryExits    []*SignedVoluntaryExit `json:"voluntary_exits" ssz-max:"16"`
	SyncAggregate     *SyncAggregate         `json:"sync_aggregate"`
	ExecutionPayload  *ExecutionPayload      `json:"execution_payload"`
}

type ExecutionPayload struct {
	ParentHash    [32]byte  `ssz-size:"32" json:"parent_hash"`
	FeeRecipient  [20]byte  `ssz-size:"20" json:"fee_recipient"`
	StateRoot     [32]byte  `ssz-size:"32" json:"state_root"`
	ReceiptsRoot  [32]byte  `ssz-size:"32" json:"receipts_root"`
	LogsBloom     [256]byte `ssz-size:"256" json:"logs_bloom"`
	PrevRandao    [32]byte  `ssz-size:"32" json:"prev_randao"`
	BlockNumber   uint64    `json:"block_number"`
	GasLimit      uint64    `json:"gas_limit"`
	GasUsed       uint64    `json:"gas_used"`
	Timestamp     uint64    `json:"timestamp"`
	ExtraData     []byte    `ssz-max:"32" json:"extra_data"`
	BaseFeePerGas [32]byte  `ssz-size:"32" json:"base_fee_per_gas"`
	BlockHash     [32]byte  `ssz-size:"32" json:"block_hash"`
	Transactions  [][]byte  `ssz-max:"1048576,1073741824" ssz-size:"?,?" json:"transactions"`
}
