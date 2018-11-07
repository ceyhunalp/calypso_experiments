package main

import (
	"github.com/BurntSushi/toml"
	"github.com/ceyhunalp/centralized_calypso/centralized"
	"github.com/ceyhunalp/centralized_calypso/util"
	"github.com/dedis/cothority"
	"github.com/dedis/onet"
	"github.com/dedis/onet/log"
	"github.com/dedis/onet/simul/monitor"
)

/*
 * Defines the simulation for the service-template
 */

const MESG_SIZE = 1024 * 1024

func init() {
	onet.SimulationRegister("CentralizedCalypso", NewCentralizedCalypsoService)
}

// SimulationService only holds the BFTree simulation
type SimulationService struct {
	onet.SimulationBFTree
}

// NewSimulationService returns the new simulation, where all fields are
// initialised using the config-file
func NewCentralizedCalypsoService(config string) (onet.Simulation, error) {
	es := &SimulationService{}
	_, err := toml.Decode(config, es)
	if err != nil {
		return nil, err
	}
	return es, nil
}

// Setup creates the tree used for that simulation
func (s *SimulationService) Setup(dir string, hosts []string) (
	*onet.SimulationConfig, error) {
	sc := &onet.SimulationConfig{}
	s.CreateRoster(sc, hosts, 2000)
	err := s.CreateTree(sc)
	if err != nil {
		return nil, err
	}
	return sc, nil
}

// Node can be used to initialize each node before it will be run
// by the server. Here we call the 'Node'-method of the
// SimulationBFTree structure which will load the roster- and the
// tree-structure to speed up the first round.
func (s *SimulationService) Node(config *onet.SimulationConfig) error {
	index, _ := config.Roster.Search(config.Server.ServerIdentity.ID)
	if index < 0 {
		log.Fatal("Didn't find this node in roster")
	}
	log.Lvl3("Initializing node-index", index)
	return s.SimulationBFTree.Node(config)
}

// Run is used on the destination machines and runs a number of
// rounds
func (s *SimulationService) Run(config *onet.SimulationConfig) error {
	log.Info("Total # of rounds is:", s.Rounds)
	serverPk := config.Roster.Publics()[0]
	size := config.Tree.Size()
	log.Info("Size of the tree:", size)

	mesg := make([]byte, MESG_SIZE)
	for i := 0; i < MESG_SIZE; i++ {
		mesg[i] = 'w'
	}

	for round := 0; round < s.Rounds; round++ {
		log.Lvl1("Starting round", round)

		rSk := cothority.Suite.Scalar().Pick(cothority.Suite.RandomStream())
		rPk := cothority.Suite.Point().Mul(rSk, nil)
		wd, err := util.CreateWriteData(mesg, rPk, serverPk)
		if err != nil {
			return err
		}
		create_write_txn := monitor.NewTimeMeasure("CreateWriteTxn")
		wd, err = centralized.CreateWriteTxn(config.Roster, wd)
		create_write_txn.Record()
		if err != nil {
			return err
		}
		log.Info("Write transaction success:", wd.StoredKey)
		kRead, cRead, err := centralized.CreateReadTxn(config.Roster, wd.StoredKey, rSk)
		if err != nil {
			return err
		}
		recvData, err := util.RecoverData(wd.Data, rSk, kRead, cRead)
		if err != nil {
			return err
		}
		log.Info("Recovered data length:", len(recvData))
	}
	//size := config.Tree.Size()
	//log.Lvl2("Size is:", size, "rounds:", s.Rounds)
	//c := template.NewClient()
	//for round := 0; round < s.Rounds; round++ {
	//log.Lvl1("Starting round", round)
	//round := monitor.NewTimeMeasure("round")
	//resp, err := c.Clock(config.Roster)
	//log.ErrFatal(err)
	//if resp.Time <= 0 {
	//log.Fatal("0 time elapsed")
	//}
	//round.Record()
	//}
	return nil
}
