package state

import (
	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/lotus/chain/types"
)

type stateSnaps struct {
	layers []*stateSnapLayer
}

type stateSnapLayer struct {
	actors       map[address.Address]streeOp
	resolveCache map[address.Address]address.Address
}

func newStateSnapLayer() *stateSnapLayer {
	return &stateSnapLayer{
		actors:       make(map[address.Address]streeOp),
		resolveCache: make(map[address.Address]address.Address),
	}
}

type streeOp struct {
	Act    types.Actor
	Delete bool
}

func newStateSnaps() *stateSnaps {
	ss := &stateSnaps{}
	ss.addLayer()
	return ss
}

func (ss *stateSnaps) addLayer() {
	ss.layers = append(ss.layers, newStateSnapLayer())
}

func (ss *stateSnaps) dropLayer() {
	ss.layers[len(ss.layers)-1] = nil // allow it to be GCed
	ss.layers = ss.layers[:len(ss.layers)-1]
}

func (ss *stateSnaps) mergeLastLayer() {
	last := ss.layers[len(ss.layers)-1]
	nextLast := ss.layers[len(ss.layers)-2]

	for k, v := range last.actors {
		nextLast.actors[k] = v
	}

	for k, v := range last.resolveCache {
		nextLast.resolveCache[k] = v
	}

	ss.dropLayer()
}

func (ss *stateSnaps) resolveAddress(addr address.Address) (address.Address, bool) {
	for i := len(ss.layers) - 1; i >= 0; i-- {
		resa, ok := ss.layers[i].resolveCache[addr]
		if ok {
			return resa, true
		}
	}
	return address.Undef, false
}

func (ss *stateSnaps) cacheResolveAddress(addr, resa address.Address) {
	ss.layers[len(ss.layers)-1].resolveCache[addr] = resa
}

func (ss *stateSnaps) getActor(addr address.Address) (*types.Actor, error) {
	for i := len(ss.layers) - 1; i >= 0; i-- {
		act, ok := ss.layers[i].actors[addr]
		if ok {
			if act.Delete {
				return nil, types.ErrActorNotFound
			}

			return &act.Act, nil
		}
	}
	return nil, nil
}

func (ss *stateSnaps) setActor(addr address.Address, act *types.Actor) {
	ss.layers[len(ss.layers)-1].actors[addr] = streeOp{Act: *act}
}

func (ss *stateSnaps) deleteActor(addr address.Address) {
	ss.layers[len(ss.layers)-1].actors[addr] = streeOp{Delete: true}
}
