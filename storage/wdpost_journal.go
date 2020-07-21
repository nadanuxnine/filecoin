package storage

import (
	"github.com/filecoin-project/specs-actors/actors/abi"
	"github.com/filecoin-project/specs-actors/actors/builtin/miner"

	"github.com/ipfs/go-cid"
)

const (
	entryTypeWdPoStRun = iota
	entryTypeWdPoStProofs
	entryTypeWdPoStFaults
	entryTypeWdPoStRecoveries
)

type wdPoStEntryCommon struct {
	Deadline *miner.DeadlineInfo
	Height   abi.ChainEpoch
	TipSet   []cid.Cid
}

type WdPoStRunEntry struct {
	wdPoStEntryCommon

	State string
	Error error `json:",omitempty"`
}

type WdPoStProofsEntry struct {
	wdPoStEntryCommon

	Partitions []miner.PoStPartition
	MessageCID cid.Cid `json:",omitempty"`
}

type WdPoStRecoveriesEntry struct {
	wdPoStEntryCommon

	Declarations []miner.RecoveryDeclaration
	MessageCID   cid.Cid `json:",omitempty"`
}

type WdPoStFaultsEntry struct {
	wdPoStEntryCommon

	Declarations []miner.FaultDeclaration
	MessageCID   cid.Cid `json:",omitempty"`
}
