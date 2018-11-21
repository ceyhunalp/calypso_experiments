package main

import (
	"github.com/BurntSushi/toml"
	"github.com/ceyhunalp/centralized_calypso/centralized"
	"github.com/ceyhunalp/centralized_calypso/util"
	"github.com/dedis/cothority"
	"github.com/dedis/kyber"
	"github.com/dedis/onet"
	"github.com/dedis/onet/log"
	"github.com/dedis/onet/simul/monitor"
)

/*
 * Defines the simulation for the service-template
 */

//const DATA_SIZE = 15
const DATA_SIZE = 1024 * 1024

func init() {
	onet.SimulationRegister("CentralizedCalypso", NewCentralizedCalypsoService)
}

// SimulationService only holds the BFTree simulation
type SimulationService struct {
	onet.SimulationBFTree
	BatchSize int
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
	var err error
	log.Info("Total # of rounds is:", s.Rounds)
	serverPk := config.Roster.Publics()[0]
	size := config.Tree.Size()
	log.Info("Size of the tree:", size)

	//data := make([]byte, DATA_SIZE)
	//for i := 0; i < DATA_SIZE; i++ {
	//data[i] = 'w'
	//}

	//wdList := make([][]byte, s.BatchSize)
	wdList := make([]*util.WriteData, s.BatchSize)
	writeTxnList := make([]*util.WriteData, s.BatchSize)
	readKList := make([]kyber.Point, s.BatchSize)
	readCList := make([]kyber.Point, s.BatchSize)

	for round := 0; round < s.Rounds; round++ {
		log.Lvl1("Starting round", round)

		rSk := cothority.Suite.Scalar().Pick(cothority.Suite.RandomStream())
		rPk := cothority.Suite.Point().Mul(rSk, nil)

		for i := 0; i < s.BatchSize; i++ {
			data := make([]byte, DATA_SIZE)
			for j := 0; j < DATA_SIZE; j++ {
				data[j] = byte(i)
			}
			//fmt.Println("Data is:", data)
			wdList[i], err = util.CreateWriteData(data, rPk, serverPk, false)
			if err != nil {
				return err
			}
		}

		cwt := monitor.NewTimeMeasure("CreateWriteTxn")
		for i := 0; i < s.BatchSize; i++ {
			writeTxnList[i], err = centralized.CreateWriteTxn(config.Roster, wdList[i])
			if err != nil {
				return err
			}
		}
		cwt.Record()

		crt := monitor.NewTimeMeasure("CreateReadTxn")
		for i := 0; i < s.BatchSize; i++ {
			readKList[i], readCList[i], err = centralized.CreateReadTxn(config.Roster, wdList[i].StoredKey, rSk)
			if err != nil {
				return err
			}
			_, err := util.RecoverData(wdList[i].Data, rSk, readKList[i], readCList[i])
			if err != nil {
				return err
			}
			//log.Info("Data recovered: ", bytes.Compare(recvData, data))
			//fmt.Println("Data recovered:", recvData)
		}
		crt.Record()
	}
	return nil
}
