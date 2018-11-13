package zerolottery

import (
	"errors"
	"github.com/dedis/cothority/byzcoin"
	"github.com/dedis/cothority/darc"
	"github.com/dedis/onet/log"
)

var ContractCommitID = "zeroLotteryCommit"

func ContractCommit(cdb byzcoin.CollectionView, inst byzcoin.Instruction, c []byzcoin.Coin) ([]byzcoin.StateChange, []byzcoin.Coin, error) {

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
		case ContractCommitID:
			cmt := inst.Spawn.Args.Search("commit")
			//w := inst.Spawn.Args.Search("commit")
			if cmt == nil || len(cmt) == 0 {
				return nil, nil, errors.New("need a commit request in 'commit' argument")
			}
			instID := inst.DeriveID("")
			log.Lvlf3("Successfully verified commit request and will store in %x", instID)
			sc = append(sc, byzcoin.NewStateChange(byzcoin.Create, instID, ContractCommitID, cmt, darcID))
		default:
			return nil, nil, errors.New("can only spawn commit")
		}
		return sc, nc, nil
	default:
		return nil, nil, errors.New("not a valid operation")
	}
}
