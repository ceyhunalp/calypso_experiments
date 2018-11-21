package zerolottery

import (
	"errors"
	"github.com/dedis/cothority/byzcoin"
	"github.com/dedis/cothority/darc"
	"github.com/dedis/onet/log"
)

var ContractLotteryStoreID = "zeroLotteryStore"

func ContractLotteryStore(cdb byzcoin.CollectionView, inst byzcoin.Instruction, c []byzcoin.Coin) ([]byzcoin.StateChange, []byzcoin.Coin, error) {

	err := inst.VerifyDarcSignature(cdb)
	if err != nil {
		return nil, nil, err
	}

	var darcID darc.ID
	_, _, darcID, err = cdb.GetValues(inst.InstanceID.Slice())
	if err != nil {
		return nil, nil, err
	}

	switch inst.GetType() {
	case byzcoin.SpawnType:
		var sc byzcoin.StateChanges
		nc := c
		switch inst.Spawn.ContractID {
		case ContractLotteryStoreID:
			str := inst.Spawn.Args.Search("store")
			//w := inst.Spawn.Args.Search("commit")
			if str == nil || len(str) == 0 {
				return nil, nil, errors.New("need a store request in 'store' argument")
			}
			instID := inst.DeriveID("")
			log.Lvlf3("Successfully verified store request and will store in %x", instID)
			sc = append(sc, byzcoin.NewStateChange(byzcoin.Create, instID, ContractLotteryStoreID, str, darcID))
		default:
			return nil, nil, errors.New("can only spawn store")
		}
		return sc, nc, nil
	default:
		return nil, nil, errors.New("not a valid operation")
	}
}
