package state

import (
	"testing"

	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/lotus/chain/types"
	"github.com/stretchr/testify/assert"
)

func TestSnapSetActor(t *testing.T) {
	a := &types.Actor{Nonce: 1}
	addr, err := address.NewIDAddress(1)
	assert.NoError(t, err)

	ss := newStateSnaps()
	ss.setActor(addr, a)

	a2, err := ss.getActor(addr)
	assert.NoError(t, err)
	if a != a2 {
		t.Fatalf("set/get actor inconsistent: a: %p, a2: %p", a, a2)
	}
}

func TestGetActorLayer(t *testing.T) {
	a := &types.Actor{Nonce: 1}
	addr, err := address.NewIDAddress(1)
	assert.NoError(t, err)

	ss := newStateSnaps()
	ss.setActor(addr, a)
	ss.addLayer()

	a1, err := ss.getActor(addr)
	assert.NoError(t, err)
	a2, err := ss.getActor(addr)
	assert.NoError(t, err)
	if a1 != a2 {
		t.Fatalf("two actor gets inconsistent: a1: %p, a2: %p", a1, a2)
	}
}
