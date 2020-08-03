package validation

import (
	"context"
	"github.com/filecoin-project/go-address"

	"golang.org/x/xerrors"

	"github.com/filecoin-project/specs-actors/actors/abi"
	"github.com/filecoin-project/specs-actors/actors/abi/big"
	"github.com/filecoin-project/specs-actors/actors/builtin"
	"github.com/filecoin-project/specs-actors/actors/crypto"
	"github.com/filecoin-project/specs-actors/actors/puppet"
	"github.com/ipfs/go-cid"

	vtypes "github.com/filecoin-project/chain-validation/chain/types"
	vstate "github.com/filecoin-project/chain-validation/state"

	"github.com/filecoin-project/lotus/chain/stmgr"
	"github.com/filecoin-project/lotus/chain/store"
	"github.com/filecoin-project/lotus/chain/types"
	"github.com/filecoin-project/lotus/chain/vm"
)

// Applier applies messages to state trees and storage.
type Applier struct {
	stateWrapper *StateWrapper
	syscalls     vm.SyscallBuilder
}

var _ vstate.Applier = &Applier{}

func NewApplier(sw *StateWrapper, syscalls vm.SyscallBuilder) *Applier {
	return &Applier{sw, syscalls}
}

func (a *Applier) ApplyMessage(epoch abi.ChainEpoch, message *vtypes.Message) (vtypes.ApplyMessageResult, error) {
	lm := toLotusMsg(message)
	receipt, penalty, reward, err := a.applyMessage(epoch, lm)
	return vtypes.ApplyMessageResult{
		Receipt: receipt,
		Penalty: penalty,
		Reward:  reward,
		Root:    a.stateWrapper.Root().String(),
	}, err
}

func (a *Applier) ApplySignedMessage(epoch abi.ChainEpoch, msg *vtypes.SignedMessage) (vtypes.ApplyMessageResult, error) {
	var lm types.ChainMsg
	switch msg.Signature.Type {
	case crypto.SigTypeSecp256k1:
		lm = toLotusSignedMsg(msg)
	case crypto.SigTypeBLS:
		lm = toLotusMsg(&msg.Message)
	default:
		return vtypes.ApplyMessageResult{}, xerrors.New("Unknown signature type")
	}
	// TODO: Validate the sig first
	receipt, penalty, reward, err := a.applyMessage(epoch, lm)
	return vtypes.ApplyMessageResult{
		Receipt: receipt,
		Penalty: penalty,
		Reward:  reward,
		Root:    a.stateWrapper.Root().String(),
	}, err

}

func (a *Applier) ApplyTipSetMessages(epoch abi.ChainEpoch, blocks []vtypes.BlockMessagesInfo, rnd vstate.RandomnessSource) (vtypes.ApplyTipSetResult, error) {
	cs := store.NewChainStore(a.stateWrapper.bs, a.stateWrapper.ds, a.syscalls)
	sm := stmgr.NewStateManager(cs)

	var bms []stmgr.BlockMessages
	for _, b := range blocks {
		bm := stmgr.BlockMessages{
			Miner:    b.Miner,
			WinCount: 1,
		}

		for _, m := range b.BLSMessages {
			bm.BlsMessages = append(bm.BlsMessages, toLotusMsg(m))
		}

		for _, m := range b.SECPMessages {
			bm.SecpkMessages = append(bm.SecpkMessages, toLotusSignedMsg(m))
		}

		bms = append(bms, bm)
	}

	var receipts []vtypes.MessageReceipt
	sroot, _, err := sm.ApplyBlocks(context.TODO(), epoch-1, a.stateWrapper.Root(), bms, epoch, &randWrapper{rnd}, func(c cid.Cid, msg *types.Message, ret *vm.ApplyRet) error {
		if msg.From == builtin.SystemActorAddr {
			return nil // ignore reward and cron calls
		}
		rval := ret.Return
		if rval == nil {
			rval = []byte{} // chain validation tests expect empty arrays to not be nil...
		}
		receipts = append(receipts, vtypes.MessageReceipt{
			Msg:         fromLotusMsg(msg),
			ExitCode:    ret.ExitCode,
			ReturnValue: rval,

			GasUsed: vtypes.GasUnits(ret.GasUsed),
		})
		return nil
	})
	if err != nil {
		return vtypes.ApplyTipSetResult{}, err
	}

	a.stateWrapper.stateRoot = sroot

	return vtypes.ApplyTipSetResult{
		Receipts: receipts,
		Root:     a.stateWrapper.Root().String(),
	}, nil
}

type randWrapper struct {
	rnd vstate.RandomnessSource
}

func (w *randWrapper) GetRandomness(ctx context.Context, pers crypto.DomainSeparationTag, round abi.ChainEpoch, entropy []byte) ([]byte, error) {
	return w.rnd.Randomness(ctx, pers, round, entropy)
}

type vmRand struct {
}

func (*vmRand) GetRandomness(ctx context.Context, dst crypto.DomainSeparationTag, h abi.ChainEpoch, input []byte) ([]byte, error) {
	panic("implement me")
}

func (a *Applier) applyMessage(epoch abi.ChainEpoch, lm types.ChainMsg) (vtypes.MessageReceipt, abi.TokenAmount, abi.TokenAmount, error) {
	ctx := context.TODO()
	base := a.stateWrapper.Root()

	lotusVM, err := vm.NewVM(base, epoch, &vmRand{}, a.stateWrapper.bs, a.syscalls, nil)
	// need to modify the VM invoker to add the puppet actor
	chainValInvoker := vm.NewInvoker()
	chainValInvoker.Register(puppet.PuppetActorCodeID, puppet.Actor{}, puppet.State{})
	lotusVM.SetInvoker(chainValInvoker)
	if err != nil {
		return vtypes.MessageReceipt{}, big.Zero(), big.Zero(), err
	}

	ret, err := lotusVM.ApplyMessage(ctx, lm)
	if err != nil {
		return vtypes.MessageReceipt{}, big.Zero(), big.Zero(), err
	}

	rval := ret.Return
	if rval == nil {
		rval = []byte{}
	}

	a.stateWrapper.stateRoot, err = lotusVM.Flush(ctx)
	if err != nil {
		return vtypes.MessageReceipt{}, big.Zero(), big.Zero(), err
	}

	mr := vtypes.MessageReceipt{
		Msg:         fromLotusMsg(lm.VMMessage()),
		ExitCode:    ret.ExitCode,
		ReturnValue: rval,
		GasUsed:     vtypes.GasUnits(ret.GasUsed),
	}

	return mr, ret.Penalty, abi.NewTokenAmount(ret.GasUsed), nil
}

func fromLotusMsg(msg *types.Message) vtypes.Message {
	return vtypes.Message{
		To:   msg.To.String(),
		From: msg.From.String(),

		CallSeqNum: msg.Nonce,
		Method:     msg.Method,

		Value:    msg.Value.Int.Int64(),
		GasPrice: msg.GasPrice.Int.Int64(),
		GasLimit: msg.GasLimit,

		Params: msg.Params,
	}
}

func toLotusMsg(msg *vtypes.Message) *types.Message {
	to, err := address.NewFromString(msg.To)
	if err != nil {
		panic("couldn't decode to address")
	}

	from, err := address.NewFromString(msg.From)
	if err != nil {
		panic("couldn't decode from address")
	}
	return &types.Message{
		To:   to,
		From: from,

		Nonce:  msg.CallSeqNum,
		Method: msg.Method,

		Value:    types.NewInt(uint64(msg.Value)),
		GasPrice: types.NewInt(uint64(msg.GasPrice)),
		GasLimit: msg.GasLimit,

		Params: msg.Params,
	}
}

func toLotusSignedMsg(msg *vtypes.SignedMessage) *types.SignedMessage {
	return &types.SignedMessage{
		Message:   *toLotusMsg(&msg.Message),
		Signature: msg.Signature,
	}
}
