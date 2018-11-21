package service

import (
	"github.com/ceyhunalp/centralized_calypso/zero_lottery"
	"github.com/dedis/cothority/byzcoin"
	"github.com/dedis/onet"
	"github.com/dedis/onet/log"
)

func init() {
	_, err := onet.RegisterNewService("zeroLottery", newService)
	log.ErrFatal(err)
}

type Service struct {
	*onet.ServiceProcessor
}

func newService(c *onet.Context) (onet.Service, error) {
	s := &Service{
		ServiceProcessor: onet.NewServiceProcessor(c),
	}
	byzcoin.RegisterContract(c, zerolottery.ContractLotteryStoreID, zerolottery.ContractLotteryStore)
	return s, nil
}
