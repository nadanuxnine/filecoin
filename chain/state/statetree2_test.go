package state_test

/*

import (
	"testing"

	"github.com/filecoin-project/lotus/chain/gen"
	"github.com/filecoin-project/lotus/chain/state"
	"github.com/filecoin-project/specs-actors/actors/abi"
	"github.com/filecoin-project/specs-actors/actors/abi/big"
	"github.com/filecoin-project/specs-actors/actors/builtin/miner"
	"github.com/filecoin-project/specs-actors/actors/builtin/power"
	"github.com/filecoin-project/specs-actors/actors/builtin/verifreg"
	cbor "github.com/ipfs/go-ipld-cbor"
	"github.com/stretchr/testify/assert"
)

func init() {
	miner.SupportedProofTypes = map[abi.RegisteredProof]struct{}{
		abi.RegisteredProof_StackedDRG2KiBSeal: {},
	}
	power.ConsensusMinerMinPower = big.NewInt(2048)
	verifreg.MinVerifiedDealSize = big.NewInt(256)
}

func TestConsistency(t *testing.T) {
	cg, err := gen.NewGenerator()
	if err != nil {
		t.Fatal(err)
	}
	t1Addr := cg.Banker()
	cst := cbor.NewCborStore(cg.Blockstore())
	st, err := state.LoadStateTree(cst, cg.CurTipset.TipSet().Blocks()[0].ParentStateRoot)
	assert.NoError(t, err)
	idAddr, err := st.LookupID(t1Addr)
	assert.NoError(t, err)
	t.Logf("addr: t1: %s, id: %s", t1Addr, idAddr)

	idAct, err := st.GetActor(idAddr)
	assert.NoError(t, err)
	t1Act, err := st.GetActor(t1Addr)
	assert.NoError(t, err)
	t.Logf("poiner: t1: %v, id: %v", t1Act, idAct)

}
*/
