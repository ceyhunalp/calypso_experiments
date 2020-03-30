package main

import (
	"bufio"
	"crypto/rand"
	"os"
	"strconv"

	"github.com/BurntSushi/toml"
	centralized "github.com/ceyhunalp/calypso_experiments/fully_centralized"
	"github.com/ceyhunalp/calypso_experiments/util"
	"github.com/dedis/cothority"
	"github.com/dedis/kyber"
	"github.com/dedis/onet"
	"github.com/dedis/onet/log"
	"github.com/dedis/onet/simul/monitor"
)

/*
 * Defines the simulation for the service-template
 */

const DATA_SIZE = 1024 * 1024

//const FIXED_COUNT int = 10

func init() {
	onet.SimulationRegister("FullyMicro", NewCentralizedCalypsoService)
}

// SimulationService only holds the BFTree simulation
type SimulationService struct {
	onet.SimulationBFTree
	BatchSize            int
	NumTransactions      int
	NumWriteTransactions int
	NumReadTransactions  int
	NumBlocks            int
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

func readAuxFile(txnList []int) error {
	//func readAuxFile(txnList []int, txnPerBlkList []int) error {
	f, err := os.Open("./txn_list.data")
	if err != nil {
		return err
	}
	idx := 0
	scanner := bufio.NewScanner(f)
	for idx < len(txnList) {
		scanner.Scan()
		txnList[idx], err = strconv.Atoi(scanner.Text())
		idx++
	}
	f.Close()
	//f, err = os.Open("./txn_per_blk.data")
	//if err != nil {
	//return err
	//}
	//idx = 0
	//scanner = bufio.NewScanner(f)
	//for idx < len(txnPerBlkList) {
	//scanner.Scan()
	//txnPerBlkList[idx], err = strconv.Atoi(scanner.Text())
	//idx++
	//}
	//f.Close()
	return nil
}

func (s *SimulationService) runCentralizedByzgen(config *onet.SimulationConfig) error {
	txnList := make([]int, s.NumTransactions)
	//blkSizeList := make([]int, s.NumBlocks)
	log.Info("Number of transactions:", s.NumTransactions)
	log.Info("Number of write transactions:", s.NumWriteTransactions)
	log.Info("Number of read transactions:", s.NumReadTransactions)
	err := readAuxFile(txnList)
	//err := readAuxFile(txnList, blkSizeList)
	if err != nil {
		log.Info("Error in readAux:", err)
		return err
	}

	serverPk := config.Roster.Publics()[0]

	//fixedWdList := make([]*util.WriteData, FIXED_COUNT)
	//fixedTxnList := make([]*util.WriteData, FIXED_COUNT)

	wdList := make([]*util.WriteData, s.NumWriteTransactions)
	writeTxnList := make([]*util.WriteData, s.NumWriteTransactions)

	readKList := make([]kyber.Point, s.NumReadTransactions)
	readCList := make([]kyber.Point, s.NumReadTransactions)

	rSk := cothority.Suite.Scalar().Pick(cothority.Suite.RandomStream())
	rPk := cothority.Suite.Point().Mul(rSk, nil)
	//for i := 0; i < FIXED_COUNT; i++ {
	//data := make([]byte, DATA_SIZE)
	//for j := 0; j < DATA_SIZE; j++ {
	//data[j] = byte(i)
	//}
	//fixedWdList[i], err = util.CreateWriteData(data, rPk, serverPk, false)
	//if err != nil {
	//return err
	//}
	//}
	//for i := 0; i < FIXED_COUNT; i++ {
	//fixedTxnList[i], err = centralized.CreateWriteTxn(config.Roster, fixedWdList[i])
	//if err != nil {
	//return err
	//}
	//}
	for i := 0; i < s.NumWriteTransactions; i++ {
		data := make([]byte, DATA_SIZE)
		for j := 0; j < DATA_SIZE; j++ {
			data[j] = byte(i)
			//data[j] = byte(FIXED_COUNT + i)
		}
		wdList[i], err = util.CreateWriteData(data, rPk, serverPk, false)
		if err != nil {
			return err
		}
	}

	for round := 0; round < s.Rounds; round++ {
		log.Lvl1("Starting round", round)

		txnIdx := 0
		lastWriteIdx := 0
		writeIdx := 0
		readIdx := 0

		simtime := monitor.NewTimeMeasure("Byzgen")
		for txnIdx < s.NumTransactions {
			if txnList[txnIdx] == 1 {
				wt := monitor.NewTimeMeasure("WriteTxn")
				writeTxnList[writeIdx], err = centralized.CreateWriteTxn(config.Roster, wdList[writeIdx])
				if err != nil {
					return err
				}
				wt.Record()
				writeIdx++
			} else {
				rt := monitor.NewTimeMeasure("ReadTxn")
				readKList[readIdx], readCList[readIdx], err = centralized.CreateReadTxn(config.Roster, wdList[lastWriteIdx].StoredKey, rSk)
				if err != nil {
					return err
				}
				rt.Record()
				rvt := monitor.NewTimeMeasure("Recover")
				_, err := util.RecoverData(wdList[lastWriteIdx].Data, rSk, readKList[readIdx], readCList[readIdx])
				if err != nil {
					return err
				}
				rvt.Record()
				readIdx++
			}
			if lastWriteIdx+1 < writeIdx {
				lastWriteIdx++
			}
			txnIdx++
		}
		simtime.Record()
	}
	return nil
}

func (s *SimulationService) runDecrypt(config *onet.SimulationConfig) error {
	var err error
	serverPk := config.Roster.Publics()[0]
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
			rand.Read(data)
			wdList[i], err = util.CreateWriteData(data, rPk, serverPk, false)
			if err != nil {
				log.Errorf("CreateWriteData failed: %v", err)
				return err
			}
		}
		for i := 0; i < s.BatchSize; i++ {
			writeTxnList[i], err = centralized.CreateWriteTxn(config.Roster, wdList[i])
			if err != nil {
				log.Errorf("CreateWriteTxn failed: %v", err)
				return err
			}
		}
		for i := 0; i < s.BatchSize; i++ {
			readKList[i], readCList[i], err = centralized.CreateReadTxn(config.Roster, wdList[i].StoredKey, rSk)
			if err != nil {
				log.Errorf("CreateReadTxn failed: %v", err)
				return err
			}
		}
		crt := monitor.NewTimeMeasure("Recoverdata")
		for i := 0; i < s.BatchSize; i++ {
			_, err := util.RecoverData(wdList[i].Data, rSk, readKList[i], readCList[i])
			if err != nil {
				log.Errorf("RecoverData failed: %v", err)
				return err
			}
			//log.LLvlf1("Recoverer data is %x", string(data))
		}
		crt.Record()
	}
	return err
}

func (s *SimulationService) runMicrobenchmark(config *onet.SimulationConfig) error {
	var err error
	log.Info("Total # of rounds is:", s.Rounds)
	serverPk := config.Roster.Publics()[0]
	size := config.Tree.Size()
	log.Info("Size of the tree:", size)

	wdList := make([]*util.WriteData, s.BatchSize)
	writeTxnList := make([]*util.WriteData, s.BatchSize)
	readKList := make([]kyber.Point, s.BatchSize)
	readCList := make([]kyber.Point, s.BatchSize)

	log.Info("Batch size is:", s.BatchSize)

	for round := 0; round < s.Rounds; round++ {
		log.Lvl1("Starting round", round)

		rSk := cothority.Suite.Scalar().Pick(cothority.Suite.RandomStream())
		rPk := cothority.Suite.Point().Mul(rSk, nil)

		for i := 0; i < s.BatchSize; i++ {
			data := make([]byte, DATA_SIZE)
			rand.Read(data)
			//log.LLvlf1("New data is %x", string(data))
			wdList[i], err = util.CreateWriteData(data, rPk, serverPk, false)
			if err != nil {
				log.Errorf("CreateWriteData failed: %v", err)
				return err
			}
		}

		cwt := monitor.NewTimeMeasure("CreateWriteTxn")
		for i := 0; i < s.BatchSize; i++ {
			writeTxnList[i], err = centralized.CreateWriteTxn(config.Roster, wdList[i])
			if err != nil {
				log.Errorf("CreateWriteTxn failed: %v", err)
				return err
			}
		}
		cwt.Record()

		crt := monitor.NewTimeMeasure("CreateReadTxn")
		for i := 0; i < s.BatchSize; i++ {
			readKList[i], readCList[i], err = centralized.CreateReadTxn(config.Roster, wdList[i].StoredKey, rSk)
			if err != nil {
				log.Errorf("CreateReadTxn failed: %v", err)
				return err
			}
			_, err := util.RecoverData(wdList[i].Data, rSk, readKList[i], readCList[i])
			if err != nil {
				log.Errorf("RecoverData failed: %v", err)
				return err
			}
			//log.LLvlf1("Recoverer data is %x", string(data))
		}
		crt.Record()
	}
	return nil
}

// Run is used on the destination machines and runs a number of
// rounds
func (s *SimulationService) Run(config *onet.SimulationConfig) error {
	err := s.runMicrobenchmark(config)
	if err != nil {
		log.Errorf("RunCentralized error: %v", err)
	}
	return nil
}
