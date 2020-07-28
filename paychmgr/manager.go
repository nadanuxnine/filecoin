package paychmgr

import (
	"context"
	"sync"

	"golang.org/x/sync/errgroup"

	"github.com/filecoin-project/go-statemachine/fsm"
	xerrors "golang.org/x/xerrors"

	"github.com/filecoin-project/lotus/api"

	"github.com/filecoin-project/specs-actors/actors/builtin/paych"

	"github.com/ipfs/go-cid"
	logging "github.com/ipfs/go-log/v2"
	"go.uber.org/fx"

	"github.com/filecoin-project/go-address"

	"github.com/filecoin-project/lotus/chain/stmgr"
	"github.com/filecoin-project/lotus/chain/types"
	"github.com/filecoin-project/lotus/node/impl/full"
)

var log = logging.Logger("paych")

type ManagerApi struct {
	fx.In

	full.MpoolAPI
	full.WalletAPI
	full.StateAPI
}

type StateManagerApi interface {
	LoadActorState(ctx context.Context, a address.Address, out interface{}, ts *types.TipSet) (*types.Actor, error)
	Call(ctx context.Context, msg *types.Message, ts *types.TipSet) (*api.InvocResult, error)
}

type Manager struct {
	// The Manager context is used to terminate wait operations on shutdown
	ctx      context.Context
	shutdown context.CancelFunc

	store         *Store
	sm            StateManagerApi
	sa            *stateAccessor
	statemachines fsm.Group
	pchapi        paychApi

	lk       sync.RWMutex
	channels map[string]*channelAccessor

	mpool  full.MpoolAPI
	wallet full.WalletAPI
	state  full.StateAPI
}

type paychAPIImpl struct {
	full.MpoolAPI
	full.StateAPI
}

func NewManager(sm *stmgr.StateManager, pchstore *Store, api ManagerApi) *Manager {
	return &Manager{
		store:    pchstore,
		sm:       sm,
		sa:       &stateAccessor{sm: sm},
		channels: make(map[string]*channelAccessor),
		// TODO: Is this the correct way to do this or can I do something different
		// with dependency injection?
		pchapi: &paychAPIImpl{api.MpoolAPI, api.StateAPI},

		mpool:  api.MpoolAPI,
		wallet: api.WalletAPI,
		state:  api.StateAPI,
	}
}

// newManager is used by the tests to supply mocks
func newManager(sm StateManagerApi, pchstore *Store, pchapi paychApi) (*Manager, error) {
	pm := &Manager{
		store:    pchstore,
		sm:       sm,
		sa:       &stateAccessor{sm: sm},
		channels: make(map[string]*channelAccessor),
		pchapi:   pchapi,
	}
	return pm, pm.Start(context.Background())
}

// HandleManager is called by dependency injection to set up hooks
func HandleManager(lc fx.Lifecycle, pm *Manager) {
	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			return pm.Start(ctx)
		},
		OnStop: func(context.Context) error {
			return pm.Stop()
		},
	})
}

// Start checks the datastore to see if there are any channels that have
// outstanding add funds messages, and if so, waits on the messages.
// Outstanding messages can occur if an add funds message was sent
// and then lotus was shut down or crashed before the result was
// received.
func (pm *Manager) Start(ctx context.Context) error {
	pm.ctx, pm.shutdown = context.WithCancel(ctx)

	cis, err := pm.store.WithPendingAddFunds()
	if err != nil {
		return err
	}

	group := errgroup.Group{}
	for _, ci := range cis {
		if ci.CreateMsg != nil {
			group.Go(func() error {
				ca, err := pm.accessorByFromTo(ci.Control, ci.Target)
				if err != nil {
					return xerrors.Errorf("error initializing payment channel manager %s -> %s: %s", ci.Control, ci.Target, err)
				}
				go func() {
					err = ca.waitForPaychCreateMsg(ci.Control, ci.Target, *ci.CreateMsg)
					ca.msgWaitComplete(err)
				}()
				return nil
			})
		} else if ci.AddFundsMsg != nil {
			group.Go(func() error {
				ca, err := pm.accessorByAddress(*ci.Channel)
				if err != nil {
					return xerrors.Errorf("error initializing payment channel manager %s: %s", ci.Channel, err)
				}
				go func() {
					err = ca.waitForAddFundsMsg(ci.Control, ci.Target, *ci.AddFundsMsg)
					ca.msgWaitComplete(err)
				}()
				return nil
			})
		}
	}

	return group.Wait()
}

// Stop shuts down any processes used by the manager
func (pm *Manager) Stop() error {
	pm.shutdown()
	return nil
}

func (pm *Manager) TrackOutboundChannel(ctx context.Context, ch address.Address) error {
	return pm.trackChannel(ctx, ch, DirOutbound)
}

func (pm *Manager) TrackInboundChannel(ctx context.Context, ch address.Address) error {
	return pm.trackChannel(ctx, ch, DirInbound)
}

func (pm *Manager) trackChannel(ctx context.Context, ch address.Address, dir uint64) error {
	pm.lk.Lock()
	defer pm.lk.Unlock()

	ci, err := pm.sa.loadStateChannelInfo(ctx, ch, dir)
	if err != nil {
		return err
	}

	return pm.store.TrackChannel(ci)
}

func (pm *Manager) GetPaych(ctx context.Context, from, to address.Address, ensureFree types.BigInt) (address.Address, cid.Cid, error) {
	chanAccessor, err := pm.accessorByFromTo(from, to)
	if err != nil {
		return address.Undef, cid.Undef, err
	}

	return chanAccessor.getPaych(ctx, from, to, ensureFree)
}

func (pm *Manager) ListChannels() ([]address.Address, error) {
	// Need to take an exclusive lock here so that channel operations can't run
	// in parallel (see channelLock)
	pm.lk.Lock()
	defer pm.lk.Unlock()

	return pm.store.ListChannels()
}

func (pm *Manager) GetChannelInfo(addr address.Address) (*ChannelInfo, error) {
	ca, err := pm.accessorByAddress(addr)
	if err != nil {
		return nil, err
	}
	return ca.getChannelInfo(addr)
}

// CheckVoucherValid checks if the given voucher is valid (is or could become spendable at some point)
func (pm *Manager) CheckVoucherValid(ctx context.Context, ch address.Address, sv *paych.SignedVoucher) error {
	ca, err := pm.accessorByAddress(ch)
	if err != nil {
		return err
	}

	_, err = ca.checkVoucherValid(ctx, ch, sv)
	return err
}

// CheckVoucherSpendable checks if the given voucher is currently spendable
func (pm *Manager) CheckVoucherSpendable(ctx context.Context, ch address.Address, sv *paych.SignedVoucher, secret []byte, proof []byte) (bool, error) {
	ca, err := pm.accessorByAddress(ch)
	if err != nil {
		return false, err
	}

	return ca.checkVoucherSpendable(ctx, ch, sv, secret, proof)
}

func (pm *Manager) AddVoucher(ctx context.Context, ch address.Address, sv *paych.SignedVoucher, proof []byte, minDelta types.BigInt) (types.BigInt, error) {
	ca, err := pm.accessorByAddress(ch)
	if err != nil {
		return types.NewInt(0), err
	}
	return ca.addVoucher(ctx, ch, sv, proof, minDelta)
}

func (pm *Manager) AllocateLane(ch address.Address) (uint64, error) {
	ca, err := pm.accessorByAddress(ch)
	if err != nil {
		return 0, err
	}
	return ca.allocateLane(ch)
}

func (pm *Manager) ListVouchers(ctx context.Context, ch address.Address) ([]*VoucherInfo, error) {
	ca, err := pm.accessorByAddress(ch)
	if err != nil {
		return nil, err
	}
	return ca.listVouchers(ctx, ch)
}

func (pm *Manager) OutboundChanTo(from, to address.Address) (address.Address, error) {
	pm.lk.Lock()
	defer pm.lk.Unlock()

	ci, err := pm.store.OutboundByFromTo(from, to)
	if err != nil {
		return address.Undef, err
	}

	// Channel create message has been sent but channel still hasn't been
	// created on chain yet
	if ci.Channel == nil {
		return address.Undef, ErrChannelNotTracked
	}

	return *ci.Channel, nil
}

func (pm *Manager) NextNonceForLane(ctx context.Context, ch address.Address, lane uint64) (uint64, error) {
	ca, err := pm.accessorByAddress(ch)
	if err != nil {
		return 0, err
	}
	return ca.nextNonceForLane(ctx, ch, lane)
}
