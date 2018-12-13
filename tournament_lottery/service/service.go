package service

import (
	tournament "github.com/ceyhunalp/centralized_calypso/tournament_lottery"
	"github.com/dedis/cothority/byzcoin"
	"github.com/dedis/onet"
	"github.com/dedis/onet/log"
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
