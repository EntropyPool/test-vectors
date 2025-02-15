package builders

import (
	"context"

	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/go-state-types/big"
	"github.com/filecoin-project/go-state-types/cbor"
	"github.com/filecoin-project/specs-actors/actors/builtin"
	"github.com/filecoin-project/specs-actors/actors/builtin/account"
	"github.com/filecoin-project/specs-actors/actors/builtin/cron"
	init_ "github.com/filecoin-project/specs-actors/actors/builtin/init"
	"github.com/filecoin-project/specs-actors/actors/builtin/market"
	"github.com/filecoin-project/specs-actors/actors/builtin/miner"
	"github.com/filecoin-project/specs-actors/actors/builtin/power"
	"github.com/filecoin-project/specs-actors/actors/builtin/reward"
	"github.com/filecoin-project/specs-actors/actors/builtin/system"
	"github.com/filecoin-project/specs-actors/actors/builtin/verifreg"
	"github.com/filecoin-project/specs-actors/actors/util/adt"

	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-bitfield"
	"github.com/ipfs/go-cid"

	"github.com/filecoin-project/lotus/conformance/chaos"

	"github.com/filecoin-project/test-vectors/schema"
)

const (
	totalFilecoin     = 2_000_000_000
	filecoinPrecision = 1_000_000_000_000_000_000
)

var (
	TotalNetworkBalance = big.Mul(big.NewInt(totalFilecoin), big.NewInt(filecoinPrecision))
	EmptyReturnValue    []byte
)

var (
	// initialized by calling insertEmptyStructures
	EmptyArrayCid        cid.Cid
	EmptyDeadlinesCid    cid.Cid
	EmptyMapCid          cid.Cid
	EmptyMultiMapCid     cid.Cid
	EmptyBitfieldCid     cid.Cid
	EmptyVestingFundsCid cid.Cid
)

const (
	TestSealProofType = abi.RegisteredSealProof_StackedDrg2KiBV1
)

func (st *StateTracker) initializeZeroState(selector schema.Selector) {
	if err := insertEmptyStructures(st.Stores.ADTStore); err != nil {
		panic(err)
	}

	type ActorState struct {
		Addr    address.Address
		Balance abi.TokenAmount
		Code    cid.Cid
		State   cbor.Marshaler
	}

	var actors []ActorState

	actors = append(actors, ActorState{
		Addr:    builtin.InitActorAddr,
		Balance: big.Zero(),
		Code:    builtin.InitActorCodeID,
		State:   init_.ConstructState(EmptyMapCid, "chain-validation"),
	})

	zeroRewardState := reward.ConstructState(big.Zero())

	actors = append(actors, ActorState{
		Addr:    builtin.RewardActorAddr,
		Balance: TotalNetworkBalance,
		Code:    builtin.RewardActorCodeID,
		State:   zeroRewardState,
	})

	actors = append(actors, ActorState{
		Addr:    builtin.BurntFundsActorAddr,
		Balance: big.Zero(),
		Code:    builtin.AccountActorCodeID,
		State:   &account.State{Address: builtin.BurntFundsActorAddr},
	})

	actors = append(actors, ActorState{
		Addr:    builtin.StoragePowerActorAddr,
		Balance: big.Zero(),
		Code:    builtin.StoragePowerActorCodeID,
		State:   power.ConstructState(EmptyMapCid, EmptyMultiMapCid),
	})

	actors = append(actors, ActorState{
		Addr:    builtin.StorageMarketActorAddr,
		Balance: big.Zero(),
		Code:    builtin.StorageMarketActorCodeID,
		State: &market.State{
			Proposals:        EmptyArrayCid,
			States:           EmptyArrayCid,
			PendingProposals: EmptyMapCid,
			EscrowTable:      EmptyMapCid,
			LockedTable:      EmptyMapCid,
			NextID:           abi.DealID(0),
			DealOpsByEpoch:   EmptyMultiMapCid,
			LastCron:         0,
		},
	})

	actors = append(actors, ActorState{
		Addr:    builtin.SystemActorAddr,
		Balance: big.Zero(),
		Code:    builtin.SystemActorCodeID,
		State:   &system.State{},
	})

	actors = append(actors, ActorState{
		Addr:    builtin.CronActorAddr,
		Balance: big.Zero(),
		Code:    builtin.CronActorCodeID,
		State: &cron.State{Entries: []cron.Entry{
			{
				Receiver:  builtin.StoragePowerActorAddr,
				MethodNum: builtin.MethodsPower.OnEpochTickEnd,
			},
		}},
	})

	// Add the chaos actor if this test requires it.
	if chaosOn, ok := selector["chaos_actor"]; ok && chaosOn == "true" {
		actors = append(actors, ActorState{
			Addr:    chaos.Address,
			Balance: big.Zero(),
			Code:    chaos.ChaosActorCodeCID,
			State:   &chaos.State{},
		})
	}

	rootVerifierID, err := address.NewFromString("t080")
	if err != nil {
		panic(err)
	}

	actors = append(actors, ActorState{
		Addr:    builtin.VerifiedRegistryActorAddr,
		Balance: big.Zero(),
		Code:    builtin.VerifiedRegistryActorCodeID,
		State:   verifreg.ConstructState(EmptyMapCid, rootVerifierID),
	})

	for _, act := range actors {
		_ = st.bc.Actors.CreateActor(act.Code, act.Addr, act.Balance, act.State)
	}
}

func insertEmptyStructures(store adt.Store) error {
	var err error
	_, err = store.Put(context.TODO(), []struct{}{})
	if err != nil {
		return err
	}

	EmptyArrayCid, err = adt.MakeEmptyArray(store).Root()
	if err != nil {
		return err
	}

	EmptyMapCid, err = adt.MakeEmptyMap(store).Root()
	if err != nil {
		return err
	}

	EmptyMultiMapCid, err = adt.MakeEmptyMultimap(store).Root()
	if err != nil {
		return err
	}

	EmptyDeadlinesCid, err = store.Put(context.TODO(), miner.ConstructDeadline(EmptyArrayCid))
	if err != nil {
		return err
	}

	emptyBitfield := bitfield.NewFromSet(nil)
	EmptyBitfieldCid, err = store.Put(context.TODO(), emptyBitfield)
	if err != nil {
		return err
	}

	EmptyVestingFundsCid, err = store.Put(context.Background(), miner.ConstructVestingFunds())
	if err != nil {
		return err
	}

	return nil
}
