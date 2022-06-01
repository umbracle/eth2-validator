package beacon

import (
	"io/ioutil"

	"github.com/umbracle/eth2-validator/internal/server/structs"
	"gopkg.in/yaml.v2"
)

// ReadChainConfig reads a beacon chain configuration
func ReadChainConfig(ref string) (*ChainConfig, error) {
	if ref == "" {
		// it defaults to mainnet
		ref = "mainnet"
	}

	var yamlContent []byte
	switch ref {
	case "mainnet":
		yamlContent = []byte(mainnetConfig)
	default:
		// try to read it as a file
		var err error
		if yamlContent, err = ioutil.ReadFile(ref); err != nil {
			return nil, err
		}
	}

	var config *ChainConfig
	if err := yaml.Unmarshal(yamlContent, &config); err != nil {
		return nil, err
	}
	return config, nil
}

// ChainConfig is the configuration of the chain
type ChainConfig struct {
	// Constants (non-configurable)
	GenesisSlot              uint64 `yaml:"GENESIS_SLOT"`                // GenesisSlot represents the first canonical slot number of the beacon chain.
	GenesisEpoch             uint64 `yaml:"GENESIS_EPOCH"`               // GenesisEpoch represents the first canonical epoch number of the beacon chain.
	FarFutureEpoch           uint64 `yaml:"FAR_FUTURE_EPOCH"`            // FarFutureEpoch represents a epoch extremely far away in the future used as the default penalization slot for validators.
	BaseRewardsPerEpoch      uint64 `yaml:"BASE_REWARDS_PER_EPOCH"`      // BaseRewardsPerEpoch is used to calculate the per epoch rewards.
	DepositContractTreeDepth uint64 `yaml:"DEPOSIT_CONTRACT_TREE_DEPTH"` // DepositContractTreeDepth depth of the Merkle trie of deposits in the validator deposit contract on the PoW chain.
	JustificationBitsLength  uint64 `yaml:"JUSTIFICATION_BITS_LENGTH"`   // JustificationBitsLength used

	// Misc constants.
	TargetCommitteeSize            uint64 `yaml:"TARGET_COMMITTEE_SIZE"`              // TargetCommitteeSize is the number of validators in a committee when the chain is healthy.
	MaxValidatorsPerCommittee      uint64 `yaml:"MAX_VALIDATORS_PER_COMMITTEE"`       // MaxValidatorsPerCommittee defines the upper bound of the size of a committee.
	MaxCommitteesPerSlot           uint64 `yaml:"MAX_COMMITTEES_PER_SLOT"`            // MaxCommitteesPerSlot defines the max amount of committee in a single slot.
	MinPerEpochChurnLimit          uint64 `yaml:"MIN_PER_EPOCH_CHURN_LIMIT"`          // MinPerEpochChurnLimit is the minimum amount of churn allotted for validator rotations.
	ChurnLimitQuotient             uint64 `yaml:"CHURN_LIMIT_QUOTIENT"`               // ChurnLimitQuotient is used to determine the limit of how many validators can rotate per epoch.
	ShuffleRoundCount              uint64 `yaml:"SHUFFLE_ROUND_COUNT"`                // ShuffleRoundCount is used for retrieving the permuted index.
	MinGenesisActiveValidatorCount uint64 `yaml:"MIN_GENESIS_ACTIVE_VALIDATOR_COUNT"` // MinGenesisActiveValidatorCount defines how many validator deposits needed to kick off beacon chain.
	MinGenesisTime                 uint64 `yaml:"MIN_GENESIS_TIME"`                   // MinGenesisTime is the time that needed to pass before kicking off beacon chain.
	TargetAggregatorsPerCommittee  uint64 `yaml:"TARGET_AGGREGATORS_PER_COMMITTEE"`   // TargetAggregatorsPerCommittee defines the number of aggregators inside one committee.
	HysteresisQuotient             uint64 `yaml:"HYSTERESIS_QUOTIENT"`                // HysteresisQuotient defines the hysteresis quotient for effective balance calculations.
	HysteresisDownwardMultiplier   uint64 `yaml:"HYSTERESIS_DOWNWARD_MULTIPLIER"`     // HysteresisDownwardMultiplier defines the hysteresis downward multiplier for effective balance calculations.
	HysteresisUpwardMultiplier     uint64 `yaml:"HYSTERESIS_UPWARD_MULTIPLIER"`       // HysteresisUpwardMultiplier defines the hysteresis upward multiplier for effective balance calculations.

	// Gwei value constants.
	MinDepositAmount          uint64 `yaml:"MIN_DEPOSIT_AMOUNT"`          // MinDepositAmount is the maximal amount of Gwei a validator can send to the deposit contract at once.
	MaxEffectiveBalance       uint64 `yaml:"MAX_EFFECTIVE_BALANCE"`       // MaxEffectiveBalance is the maximal amount of Gwei that is effective for staking.
	EjectionBalance           uint64 `yaml:"EJECTION_BALANCE"`            // EjectionBalance is the minimal GWei a validator needs to have before ejected.
	EffectiveBalanceIncrement uint64 `yaml:"EFFECTIVE_BALANCE_INCREMENT"` // EffectiveBalanceIncrement is used for converting the high balance into the low balance for validators.

	// Initial value constants.
	BLSWithdrawalPrefixByte byte     `yaml:"BLS_WITHDRAWAL_PREFIX"` // BLSWithdrawalPrefixByte is used for BLS withdrawal and it's the first byte.
	ZeroHash                [32]byte // ZeroHash is used to represent a zeroed out 32 byte array.

	// Time parameters constants.
	GenesisDelay                     uint64 `yaml:"GENESIS_DELAY"`                       // Minimum number of seconds to delay starting the ETH2 genesis. Must be at least 1 second.
	MinAttestationInclusionDelay     uint64 `yaml:"MIN_ATTESTATION_INCLUSION_DELAY"`     // MinAttestationInclusionDelay defines how many slots validator has to wait to include attestation for beacon block.
	SecondsPerSlot                   uint64 `yaml:"SECONDS_PER_SLOT"`                    // SecondsPerSlot is how many seconds are in a single slot.
	SlotsPerEpoch                    uint64 `yaml:"SLOTS_PER_EPOCH"`                     // SlotsPerEpoch is the number of slots in an epoch.
	MinSeedLookahead                 uint64 `yaml:"MIN_SEED_LOOKAHEAD"`                  // SeedLookahead is the duration of randao look ahead seed.
	MaxSeedLookahead                 uint64 `yaml:"MAX_SEED_LOOKAHEAD"`                  // MaxSeedLookahead is the duration a validator has to wait for entry and exit in epoch.
	EpochsPerEth1VotingPeriod        uint64 `yaml:"EPOCHS_PER_ETH1_VOTING_PERIOD"`       // EpochsPerEth1VotingPeriod defines how often the merkle root of deposit receipts get updated in beacon node on per epoch basis.
	SlotsPerHistoricalRoot           uint64 `yaml:"SLOTS_PER_HISTORICAL_ROOT"`           // SlotsPerHistoricalRoot defines how often the historical root is saved.
	MinValidatorWithdrawabilityDelay uint64 `yaml:"MIN_VALIDATOR_WITHDRAWABILITY_DELAY"` // MinValidatorWithdrawabilityDelay is the shortest amount of time a validator has to wait to withdraw.
	ShardCommitteePeriod             uint64 `yaml:"SHARD_COMMITTEE_PERIOD"`              // ShardCommitteePeriod is the minimum amount of epochs a validator must participate before exiting.
	MinEpochsToInactivityPenalty     uint64 `yaml:"MIN_EPOCHS_TO_INACTIVITY_PENALTY"`    // MinEpochsToInactivityPenalty defines the minimum amount of epochs since finality to begin penalizing inactivity.
	Eth1FollowDistance               uint64 // Eth1FollowDistance is the number of eth1.0 blocks to wait before considering a new deposit for voting. This only applies after the chain as been started.
	SafeSlotsToUpdateJustified       uint64 // SafeSlotsToUpdateJustified is the minimal slots needed to update justified check point.
	SecondsPerETH1Block              uint64 `yaml:"SECONDS_PER_ETH1_BLOCK"` // SecondsPerETH1Block is the approximate time for a single eth1 block to be produced.
	// State list lengths
	EpochsPerHistoricalVector uint64 `yaml:"EPOCHS_PER_HISTORICAL_VECTOR"` // EpochsPerHistoricalVector defines max length in epoch to store old historical stats in beacon state.
	EpochsPerSlashingsVector  uint64 `yaml:"EPOCHS_PER_SLASHINGS_VECTOR"`  // EpochsPerSlashingsVector defines max length in epoch to store old stats to recompute slashing witness.
	HistoricalRootsLimit      uint64 `yaml:"HISTORICAL_ROOTS_LIMIT"`       // HistoricalRootsLimit the define max historical roots can be saved in state before roll over.
	ValidatorRegistryLimit    uint64 `yaml:"VALIDATOR_REGISTRY_LIMIT"`     // ValidatorRegistryLimit defines the upper bound of validators can participate in eth2.

	// Reward and penalty quotients constants.
	BaseRewardFactor            uint64 `yaml:"BASE_REWARD_FACTOR"`            // BaseRewardFactor is used to calculate validator per-slot interest rate.
	WhistleBlowerRewardQuotient uint64 `yaml:"WHISTLEBLOWER_REWARD_QUOTIENT"` // WhistleBlowerRewardQuotient is used to calculate whistler blower reward.
	ProposerRewardQuotient      uint64 `yaml:"PROPOSER_REWARD_QUOTIENT"`      // ProposerRewardQuotient is used to calculate the reward for proposers.
	InactivityPenaltyQuotient   uint64 `yaml:"INACTIVITY_PENALTY_QUOTIENT"`   // InactivityPenaltyQuotient is used to calculate the penalty for a validator that is offline.
	MinSlashingPenaltyQuotient  uint64 `yaml:"MIN_SLASHING_PENALTY_QUOTIENT"` // MinSlashingPenaltyQuotient is used to calculate the minimum penalty to prevent DoS attacks.

	// Max operations per block constants.
	MaxProposerSlashings uint64 `yaml:"MAX_PROPOSER_SLASHINGS"` // MaxProposerSlashings defines the maximum number of slashings of proposers possible in a block.
	MaxAttesterSlashings uint64 `yaml:"MAX_ATTESTER_SLASHINGS"` // MaxAttesterSlashings defines the maximum number of casper FFG slashings possible in a block.
	MaxAttestations      uint64 `yaml:"MAX_ATTESTATIONS"`       // MaxAttestations defines the maximum allowed attestations in a beacon block.
	MaxDeposits          uint64 `yaml:"MAX_DEPOSITS"`           // MaxVoluntaryExits defines the maximum number of validator deposits in a block.
	MaxVoluntaryExits    uint64 `yaml:"MAX_VOLUNTARY_EXITS"`    // MaxVoluntaryExits defines the maximum number of validator exits in a block.

	// BLS domain values.
	DomainBeaconProposer    structs.Domain `yaml:"DOMAIN_BEACON_PROPOSER"`     // DomainBeaconProposer defines the BLS signature domain for beacon proposal verification.
	DomainRandao            structs.Domain `yaml:"DOMAIN_RANDAO"`              // DomainRandao defines the BLS signature domain for randao verification.
	DomainBeaconAttester    structs.Domain `yaml:"DOMAIN_BEACON_ATTESTER"`     // DomainBeaconAttester defines the BLS signature domain for attestation verification.
	DomainDeposit           structs.Domain `yaml:"DOMAIN_DEPOSIT"`             // DomainDeposit defines the BLS signature domain for deposit verification.
	DomainVoluntaryExit     structs.Domain `yaml:"DOMAIN_VOLUNTARY_EXIT"`      // DomainVoluntaryExit defines the BLS signature domain for exit verification.
	DomainSelectionProof    structs.Domain `yaml:"DOMAIN_SELECTION_PROOF"`     // DomainSelectionProof defines the BLS signature domain for selection proof.
	DomainAggregateAndProof structs.Domain `yaml:"DOMAIN_AGGREGATE_AND_PROOF"` // DomainAggregateAndProof defines the BLS signature domain for aggregate and proof.
	DomainSyncCommittee     structs.Domain `yaml:"DOMAIN_SYNC_COMMITTEE"`

	// Fork values
	GenesisForkVersion structs.Domain `yaml:"GENESIS_FORK_VERSION"` // GenesisForkVersion is used to track fork version between state transitions.

	AltairForkVersion structs.Domain `yaml:"ALTAIR_FORK_VERSION"`
	AltairForkEpoch   uint64         `yaml:"ALTAIR_FORK_EPOCH"`

	BellatrixForkVersion structs.Domain `yaml:"BELLATRIX_FORK_VERSION"`
	BellatrixForkEpoch   uint64         `yaml:"BELLATRIX_FORK_EPOCH"`
}

var zeroHash = make([]byte, 32)

// ComputeDomain returns the domain version for BLS private key to sign and verify with a zeroed 4-byte
// array as the fork version
func (c *ChainConfig) ComputeDomain(domain structs.Domain, forkVersion []byte, genesisValidatorsRoot []byte) ([]byte, error) {
	if forkVersion == nil {
		forkVersion = c.GenesisForkVersion[:]
	}
	if genesisValidatorsRoot == nil {
		genesisValidatorsRoot = zeroHash
	}

	// compute_fork_data_root
	// this returns the 32byte fork data root for the ``current_version`` and ``genesis_validators_root``.
	// This is used primarily in signature domains to avoid collisions across forks/chains.
	forkData := structs.ForkData{
		CurrentVersion:        forkVersion[:4],
		GenesisValidatorsRoot: genesisValidatorsRoot,
	}
	forkRoot, err := forkData.HashTreeRoot()
	if err != nil {
		return nil, err
	}

	return append(domain[:], forkRoot[:28]...), nil
}

// ComputeDepositDomain computes the domain for the deposit type
func (c *ChainConfig) ComputeDepositDomain(forkVersion []byte, genesisValidatorsRoot []byte) ([]byte, error) {
	return c.ComputeDomain(c.DomainDeposit, forkVersion, genesisValidatorsRoot)
}
