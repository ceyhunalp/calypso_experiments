package main

import (
	"bufio"
	"crypto/rand"
	"errors"
	"io/ioutil"
	"os"
	"strconv"

	"github.com/BurntSushi/toml"
	sc "github.com/ceyhunalp/calypso_experiments/semi_centralized"
	"github.com/ceyhunalp/calypso_experiments/util"
	"go.dedis.ch/cothority/v3/byzcoin"
	"go.dedis.ch/kyber/v3"
	"go.dedis.ch/onet/v3"
	"go.dedis.ch/onet/v3/log"
	"go.dedis.ch/onet/v3/simul/monitor"
)

/*
 * Defines the simulation for the service-template
 */

const DATA_SIZE = 1024 * 1024
const FIXED_COUNT int = 10

func init() {
	onet.SimulationRegister("SemiCentralized", NewSemiCentralizedService)
}

// SimulationService only holds the BFTree simulation
type SimulationService struct {
	onet.SimulationBFTree
	BatchSize            int
	NumTransactions      int
	NumWriteTransactions int
	NumReadTransactions  int
	NumBlocks            int
	BlockInterval        int
}

// NewSimulationService returns the new simulation, where all fields are
// initialised using the config-file
func NewSemiCentralizedService(config string) (onet.Simulation, error) {
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
	buf, err := ioutil.ReadFile("./txn_list_82.data")
	if err != nil {
		return nil, err
	}
	err = ioutil.WriteFile(dir+"/txn_list_82.data", buf, 0777)
	if err != nil {
		return nil, err
	}
	buf, err = ioutil.ReadFile("./txn_per_blk_82.data")
	if err != nil {
		return nil, err
	}
	err = ioutil.WriteFile(dir+"/txn_per_blk_82.data", buf, 0777)
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

func (s *SimulationService) runMicrobenchmark(config *onet.SimulationConfig, serverPk kyber.Point) error {
	wdList := make([]*util.WriteData, s.BatchSize)
	writeTxnList := make([]*sc.TransactionReply, s.BatchSize)
	readTxnList := make([]*sc.TransactionReply, s.BatchSize)
	wrProofList := make([]*byzcoin.Proof, s.BatchSize)
	readProofList := make([]*byzcoin.Proof, s.BatchSize)
	//decReqList := make([]*simpServ.DecryptReply, s.BatchSize)

	log.Info("Roster size is:", len(config.Roster.List))

	for round := 0; round < s.Rounds; round++ {
		log.Lvl1("Starting round", round)
		byzd, err := sc.SetupByzcoin(config.Roster, s.BlockInterval)
		if err != nil {
			log.Errorf("Setting up Byzcoin failed: %v", err)
			return err
		}

		writer, reader, wDarc, err := sc.SetupDarcs()
		if err != nil {
			return err
		}

		_, err = byzd.SpawnDarc(*wDarc, 4)
		if err != nil {
			return err
		}

		for i := 0; i < s.BatchSize; i++ {
			data := make([]byte, DATA_SIZE)
			rand.Read(data)
			wdList[i], err = util.CreateWriteData(data, reader.Ed25519.Point, serverPk, true)
			if err != nil {
				return err
			}
		}

		//sed := monitor.NewTimeMeasure("store_enc_data")

		awt := monitor.NewTimeMeasure("AddWriteTxn")
		for i := 0; i < s.BatchSize; i++ {
			err = sc.StoreEncryptedData(config.Roster, wdList[i])
			if err != nil {
				return err
			}
		}
		for i := 0; i < s.BatchSize; i++ {
			wait := 0
			if i == s.BatchSize-1 {
				wait = 3
			}
			writeTxnList[i], err = byzd.AddWriteTransaction(wdList[i], writer, *wDarc, wait)
			if err != nil {
				return err
			}
		}
		awt.Record()

		wwp := monitor.NewTimeMeasure("WriteGetProof")
		for i := 0; i < s.BatchSize; i++ {
			wrProofResponse, err := byzd.GetProof(writeTxnList[i].InstanceID)
			if err != nil {
				return err
			}
			wrProof := wrProofResponse.Proof
			if !wrProof.InclusionProof.Match() {
				return errors.New("Write inclusion proof does not match")
			}
			wrProofList[i] = &wrProof
		}
		wwp.Record()

		art := monitor.NewTimeMeasure("AddReadTxn")
		for i := 0; i < s.BatchSize; i++ {
			wait := 0
			if i == s.BatchSize-1 {
				wait = 3
			}
			readTxnList[i], err = byzd.AddReadTransaction(wrProofList[i], reader, *wDarc, wait)
			if err != nil {
				return err
			}
		}
		art.Record()

		rwp := monitor.NewTimeMeasure("ReadGetProof")
		for i := 0; i < s.BatchSize; i++ {
			rProofResponse, err := byzd.GetProof(readTxnList[i].InstanceID)
			if err != nil {
				return err
			}
			rProof := rProofResponse.Proof
			if !rProof.InclusionProof.Match() {
				return errors.New("Read inclusion proof does not match")
			}
			readProofList[i] = &rProof
		}
		rwp.Record()

		decReq := monitor.NewTimeMeasure("DecRequest")
		for i := 0; i < s.BatchSize; i++ {
			dr, err := byzd.DecryptRequest(config.Roster, wrProofList[i], readProofList[i], wdList[i].StoredKey, reader.Ed25519.Secret)
			//dr, err := byzd.DecryptRequest(config.Roster, wrProofList[i], readProofList[i], wd.StoredKey, reader.Ed25519.Secret)
			if err != nil {
				return err
			}

			//dr := decReqList[i]
			_, err = util.RecoverData(dr.Data, reader.Ed25519.Secret, dr.K, dr.C)
			if err != nil {
				return err
			}
			//log.Info("Data recovered: ", bytes.Compare(recvData, data))
		}
		decReq.Record()

		//rd.Record()

		//awt := monitor.NewTimeMeasure("add_write_txn")
		//writeTxn, err := byzd.AddWriteTransaction(wd, writer, *wDarc, 3)
		//awt.Record()
		//if err != nil {
		//return err
		//}

		//wwp := monitor.NewTimeMeasure("write_get_proof")
		//wrProofResponse, err := byzd.GetProof(writeTxn.InstanceID)
		//if err != nil {
		//return err
		//}
		//wrProof := wrProofResponse.Proof
		//if !wrProof.InclusionProof.Match() {
		//return errors.New("Write inclusion proof does not match")
		//}
		//wwp.Record()

		//art := monitor.NewTimeMeasure("add_read_txn")
		//readTxn, err := byzd.AddReadTransaction(&wrProof, reader, *wDarc, 3)
		//art.Record()
		//if err != nil {
		//return err
		//}

		//rwp := monitor.NewTimeMeasure("read_get_proof")
		//rProofResponse, err := byzd.GetProof(readTxn.InstanceID)
		//if err != nil {
		//return err
		//}
		//rProof := rProofResponse.Proof
		//if !rProof.InclusionProof.Match() {
		//return errors.New("Read inclusion proof does not match")
		//}
		//rwp.Record()

		//decReq := monitor.NewTimeMeasure("dec_req")
		//dr, err := byzd.DecryptRequest(config.Roster, &wrProof, &rProof, wd.StoredKey, reader.Ed25519.Secret)
		//decReq.Record()
		//if err != nil {
		//return err
		//}

		//rd := monitor.NewTimeMeasure("rec_data")
		//recvData, err := util.RecoverData(dr.Data, reader.Ed25519.Secret, dr.K, dr.C)
		//rd.Record()

		//if err != nil {
		//return err
		//}
		//log.Info("Data recovered: ", bytes.Compare(recvData, data))
	}
	return nil
}

func readAuxFile(txnList []int, txnPerBlkList []int) error {
	f, err := os.Open("./txn_list_82.data")
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

	f, err = os.Open("./txn_per_blk_82.data")
	if err != nil {
		return err
	}

	idx = 0
	scanner = bufio.NewScanner(f)
	for idx < len(txnPerBlkList) {
		scanner.Scan()
		txnPerBlkList[idx], err = strconv.Atoi(scanner.Text())
		idx++
	}
	f.Close()

	return nil
}

//func (s *SimulationService) runByzgenSimulation(config *onet.SimulationConfig, serverPk kyber.Point) error {

//txnList := make([]int, s.NumTransactions)
//blkSizeList := make([]int, s.NumBlocks)
//log.Info("Number of transactions:", s.NumTransactions)
//log.Info("Number of blocks:", s.NumBlocks)
//err := readAuxFile(txnList, blkSizeList)
//if err != nil {
//log.Info("Error in readAux:", err)
//return err
//}

//fixedWdList := make([]*util.WriteData, FIXED_COUNT)
//fixedTxnList := make([]*sc.TransactionReply, FIXED_COUNT)
//fixedProofList := make([]*byzcoin.Proof, FIXED_COUNT)

//wdList := make([]*util.WriteData, s.NumWriteTransactions)
//writeTxnList := make([]*sc.TransactionReply, s.NumWriteTransactions)
//readTxnList := make([]*sc.TransactionReply, s.NumReadTransactions)
//readProofList := make([]*byzcoin.Proof, s.NumReadTransactions)

//byzd, err := sc.SetupByzcoin(config.Roster)
//if err != nil {
//log.Errorf("Setting up Byzcoin failed: %v", err)
//return err
//}
//writer, reader, wDarc, err := sc.SetupDarcs()
//if err != nil {
//return err
//}
//_, err = byzd.SpawnDarc(*wDarc, 4)
//if err != nil {
//return err
//}

//for i := 0; i < FIXED_COUNT; i++ {
//data := make([]byte, DATA_SIZE)
//for j := 0; j < DATA_SIZE; j++ {
//data[j] = byte(i)
//}
//fixedWdList[i], err = util.CreateWriteData(data, reader.Ed25519.Point, serverPk, true)
//if err != nil {
//return err
//}
//}
//for i := 0; i < FIXED_COUNT; i++ {
//err = sc.StoreEncryptedData(config.Roster, fixedWdList[i])
//if err != nil {
//return err
//}
//}
//for i := 0; i < FIXED_COUNT; i++ {
//wait := 0
//if i == FIXED_COUNT-1 {
//wait = 3
//}
//fixedTxnList[i], err = byzd.AddWriteTransaction(fixedWdList[i], writer, *wDarc, wait)
//if err != nil {
//return err
//}
//}
//for i := 0; i < FIXED_COUNT; i++ {
//wrProofResponse, err := byzd.GetProof(fixedTxnList[i].InstanceID)
//if err != nil {
//return err
//}
//wrProof := wrProofResponse.Proof
//if !wrProof.InclusionProof.Match() {
//return errors.New("Write inclusion proof does not match")
//}
//fixedProofList[i] = &wrProof
//}

//for i := 0; i < s.NumWriteTransactions; i++ {
//data := make([]byte, DATA_SIZE)
//for j := 0; j < DATA_SIZE; j++ {
//data[j] = byte(FIXED_COUNT + i)
//}
//wdList[i], err = util.CreateWriteData(data, reader.Ed25519.Point, serverPk, true)
//if err != nil {
//return err
//}
//}

//for round := 0; round < s.Rounds; round++ {
//log.Lvl1("Starting round", round)

//txnIdx := 0
//blkSizeIdx := 0
//writeIdx := 0
//readIdx := 0

//simtime := monitor.NewTimeMeasure("Byzgen_Semi")
//for txnIdx < s.NumTransactions {
////log.Info(txnIdx)
//blkSize := blkSizeList[blkSizeIdx]
//writeCnt := 0
//readCnt := 0

//for i := 0; i < blkSize; i++ {
//wait := 0
//if i == blkSize-1 {
//wait = 3
//}
//if txnList[txnIdx] == 1 {
//// WRITE TXN
//wt := monitor.NewTimeMeasure("AddWriteTxn")
//err = sc.StoreEncryptedData(config.Roster, wdList[writeIdx])
//if err != nil {
//return err
//}
//writeTxnList[writeIdx], err = byzd.AddWriteTransaction(wdList[writeIdx], writer, *wDarc, wait)
//if err != nil {
//return err
//}
//wt.Record()
//writeCnt++
//writeIdx++
//} else {
//// READ TXN
//rt := monitor.NewTimeMeasure("AddReadTxn")
//readTxnList[readIdx], err = byzd.AddReadTransaction(fixedProofList[readIdx%FIXED_COUNT], reader, *wDarc, wait)
//if err != nil {
//return err
//}
//rt.Record()
//readCnt++
//readIdx++
//}
//txnIdx++
//}
//wpt := monitor.NewTimeMeasure("WriteGetProof")
//for j := 0; j < writeCnt; j++ {
//wrProofResponse, err := byzd.GetProof(writeTxnList[writeIdx-j-1].InstanceID)
//if err != nil {
//return err
//}
//wrProof := wrProofResponse.Proof
//if !wrProof.InclusionProof.Match() {
//return errors.New("Write inclusion proof does not match")
//}
////wrProofList[j] = &wrProof
//}
//wpt.Record()
//dt := monitor.NewTimeMeasure("Decrypt")
//for j := 1; j <= readCnt; j++ {
//rProofResponse, err := byzd.GetProof(readTxnList[readIdx-j].InstanceID)
//if err != nil {
//return err
//}
//rProof := rProofResponse.Proof
//if !rProof.InclusionProof.Match() {
//return errors.New("Read inclusion proof does not match")
//}
//readProofList[readIdx-j] = &rProof
//dr, err := byzd.DecryptRequest(config.Roster, fixedProofList[(readIdx-j)%FIXED_COUNT], readProofList[readIdx-j], fixedWdList[(readIdx-j)%FIXED_COUNT].StoredKey, reader.Ed25519.Secret)
//if err != nil {
//return err
//}

//_, err = util.RecoverData(dr.Data, reader.Ed25519.Secret, dr.K, dr.C)
//if err != nil {
//return err
//}
//}
//dt.Record()
//blkSizeIdx++
//}
//simtime.Record()
//log.Info("I am done", blkSizeIdx, txnIdx)
//}
//return nil
//}

func (s *SimulationService) runByzgenSimulation(config *onet.SimulationConfig, serverPk kyber.Point) error {

	txnList := make([]int, s.NumTransactions)
	blkSizeList := make([]int, s.NumBlocks)
	log.Info("Number of transactions:", s.NumTransactions)
	log.Info("Number of blocks:", s.NumBlocks)
	err := readAuxFile(txnList, blkSizeList)
	if err != nil {
		log.Info("Error in readAux:", err)
		return err
	}

	fixedWdList := make([]*util.WriteData, FIXED_COUNT)
	fixedTxnList := make([]*sc.TransactionReply, FIXED_COUNT)
	fixedProofList := make([]*byzcoin.Proof, FIXED_COUNT)

	wdList := make([]*util.WriteData, s.NumWriteTransactions)
	writeTxnList := make([]*sc.TransactionReply, s.NumWriteTransactions)
	readTxnList := make([]*sc.TransactionReply, s.NumReadTransactions)
	readProofList := make([]*byzcoin.Proof, s.NumReadTransactions)

	byzd, err := sc.SetupByzcoin(config.Roster, s.BlockInterval)
	if err != nil {
		log.Errorf("Setting up Byzcoin failed: %v", err)
		return err
	}
	writer, reader, wDarc, err := sc.SetupDarcs()
	if err != nil {
		return err
	}
	_, err = byzd.SpawnDarc(*wDarc, 3)
	if err != nil {
		return err
	}

	for i := 0; i < FIXED_COUNT; i++ {
		data := make([]byte, DATA_SIZE)
		for j := 0; j < DATA_SIZE; j++ {
			data[j] = byte(i)
		}
		fixedWdList[i], err = util.CreateWriteData(data, reader.Ed25519.Point, serverPk, true)
		if err != nil {
			return err
		}
	}
	for i := 0; i < FIXED_COUNT; i++ {
		err = sc.StoreEncryptedData(config.Roster, fixedWdList[i])
		if err != nil {
			return err
		}
	}
	for i := 0; i < FIXED_COUNT; i++ {
		wait := 0
		if i == FIXED_COUNT-1 {
			wait = 3
		}
		fixedTxnList[i], err = byzd.AddWriteTransaction(fixedWdList[i], writer, *wDarc, wait)
		if err != nil {
			return err
		}
	}
	for i := 0; i < FIXED_COUNT; i++ {
		wrProofResponse, err := byzd.GetProof(fixedTxnList[i].InstanceID)
		if err != nil {
			return err
		}
		wrProof := wrProofResponse.Proof
		if !wrProof.InclusionProof.Match() {
			return errors.New("Write inclusion proof does not match")
		}
		fixedProofList[i] = &wrProof
	}

	for i := 0; i < s.NumWriteTransactions; i++ {
		data := make([]byte, DATA_SIZE)
		for j := 0; j < DATA_SIZE; j++ {
			data[j] = byte(FIXED_COUNT + i)
		}
		wdList[i], err = util.CreateWriteData(data, reader.Ed25519.Point, serverPk, true)
		if err != nil {
			return err
		}
	}

	for round := 0; round < s.Rounds; round++ {
		log.Lvl1("Starting round", round)

		txnIdx := 0
		blkSizeIdx := 0
		writeIdx := 0
		readIdx := 0

		for txnIdx < s.NumTransactions {
			//log.Info(txnIdx)
			blkSize := blkSizeList[blkSizeIdx]
			writeCnt := 0
			readCnt := 0

			measureStr := "Block_" + strconv.Itoa(blkSizeIdx)
			blkTime := monitor.NewTimeMeasure(measureStr)
			for i := 0; i < blkSize; i++ {
				wait := 0
				if i == blkSize-1 {
					wait = 3
				}
				if txnList[txnIdx] == 1 {
					// WRITE TXN
					err = sc.StoreEncryptedData(config.Roster, wdList[writeIdx])
					if err != nil {
						return err
					}
					writeTxnList[writeIdx], err = byzd.AddWriteTransaction(wdList[writeIdx], writer, *wDarc, wait)
					if err != nil {
						return err
					}
					writeCnt++
					writeIdx++
				} else {
					// READ TXN
					readTxnList[readIdx], err = byzd.AddReadTransaction(fixedProofList[readIdx%FIXED_COUNT], reader, *wDarc, wait)
					if err != nil {
						return err
					}
					readCnt++
					readIdx++
				}
				txnIdx++
			}
			blkTime.Record()

			wpt := monitor.NewTimeMeasure("WriteProof")
			for j := 0; j < writeCnt; j++ {
				wrProofResponse, err := byzd.GetProof(writeTxnList[writeIdx-j-1].InstanceID)
				if err != nil {
					return err
				}
				wrProof := wrProofResponse.Proof
				if !wrProof.InclusionProof.Match() {
					return errors.New("Write inclusion proof does not match")
				}
			}
			wpt.Record()
			dt := monitor.NewTimeMeasure("Decrypt")
			for j := 1; j <= readCnt; j++ {
				rProofResponse, err := byzd.GetProof(readTxnList[readIdx-j].InstanceID)
				if err != nil {
					return err
				}
				rProof := rProofResponse.Proof
				if !rProof.InclusionProof.Match() {
					return errors.New("Read inclusion proof does not match")
				}
				readProofList[readIdx-j] = &rProof
				dr, err := byzd.DecryptRequest(config.Roster, fixedProofList[(readIdx-j)%FIXED_COUNT], readProofList[readIdx-j], fixedWdList[(readIdx-j)%FIXED_COUNT].StoredKey, reader.Ed25519.Secret)
				if err != nil {
					return err
				}

				_, err = util.RecoverData(dr.Data, reader.Ed25519.Secret, dr.K, dr.C)
				if err != nil {
					return err
				}
			}
			dt.Record()
			blkSizeIdx++
		}
		log.Info("I am done", blkSizeIdx, txnIdx)
	}
	return nil
}

// Run is used on the destination machines and runs a number of
// rounds
func (s *SimulationService) Run(config *onet.SimulationConfig) error {
	log.Info("Total # of rounds is:", s.Rounds)
	serverPk := config.Roster.Publics()[0]
	size := config.Tree.Size()
	log.Info("Size of the tree:", size)

	//err := s.runByzgenSimulation(config, serverPk)
	err := s.runMicrobenchmark(config, serverPk)
	if err != nil {
		log.Info("Simulation error:", err)
		return err
	}
	return nil
}
