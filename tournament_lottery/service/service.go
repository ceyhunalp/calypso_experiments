package service

import (
	tournament "github.com/ceyhunalp/calypso_experiments/tournament_lottery"
	"go.dedis.ch/cothority/byzcoin"
	"go.dedis.ch/onet"
	"go.dedis.ch/onet/log"
)

func init() {
	_, err := onet.RegisterNewService("tournamentLottery", newService)
	log.ErrFatal(err)
}

type Service struct {
	*onet.ServiceProcessor
}

func newService(c *onet.Context) (onet.Service, error) {
	s := &Service{
		ServiceProcessor: onet.NewServiceProcessor(c),
	}
	byzcoin.RegisterContract(c, tournament.ContractLotteryStoreID, tournament.ContractLotteryStore)
	return s, nil
}
