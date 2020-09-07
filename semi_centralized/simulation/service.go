package main

import (
	"bufio"
	"bytes"
	"crypto/rand"
	"errors"
	"fmt"
	"os"
	"strconv"
	"sync"

	"github.com/BurntSushi/toml"
	sc "github.com/ceyhunalp/calypso_experiments/semi_centralized"
	"github.com/ceyhunalp/calypso_experiments/util"
	"github.com/dedis/cothority/byzcoin"
	"github.com/dedis/cothority/darc"
	"github.com/dedis/kyber"
	"github.com/dedis/onet"
	"github.com/dedis/onet/log"
	"github.com/dedis/onet/simul/monitor"
)

/*
 * Defines the simulation for the service-template
 */

const DATA_SIZE = 1024 * 1024
const FIXED_COUNT int = 10

var wg sync.WaitGroup

// SimulationService only holds the BFTree simulation
type SimulationService struct {
	onet.SimulationBFTree
	BatchSize            int
	NumTransactions      int
	NumWriteTransactions int
	NumReadTransactions  int
	NumBlocks            int
	BlockInterval        int
	BlockWait            int
}

func init() {
	onet.SimulationRegister("Semi", NewSemiCentralizedService)
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
	//buf, err := ioutil.ReadFile("./txn_list_82.data")
	//if err != nil {
	//return nil, err
	//}
	//err = ioutil.WriteFile(dir+"/txn_list_82.data", buf, 0777)
	//if err != nil {
	//return nil, err
	//}
	//buf, err = ioutil.ReadFile("./txn_per_blk_82.data")
	//if err != nil {
	//return nil, err
	//}
	//err = ioutil.WriteFile(dir+"/txn_per_blk_82.data", buf, 0777)
	//if err != nil {
	//return nil, err
	//}
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
	dList := make([][]byte, s.BatchSize)
	wdList := make([]*util.WriteData, s.BatchSize)
	writeTxnList := make([]*sc.TransactionReply, s.BatchSize)
	readTxnList := make([]*sc.TransactionReply, s.BatchSize)
	wrProofList := make([]*byzcoin.Proof, s.BatchSize)
	readProofList := make([]*byzcoin.Proof, s.BatchSize)

	log.Info("Roster size is:", len(config.Roster.List))

	for round := 0; round < s.Rounds; round++ {
		log.Lvl1("Starting round", round)
		byzCl, admin, gDarc, err := sc.SetupByzcoin(config.Roster, s.BlockInterval)
		if err != nil {
			log.Errorf("Setting up Byzcoin failed: %v", err)
			return err
		}

		scCl := sc.NewClient(byzCl)
		writer, reader, wDarc, err := scCl.SetupDarcs()
		if err != nil {
			return err
		}
		_, err = scCl.SpawnDarc(admin, *wDarc, gDarc, 4)
		if err != nil {
			return err
		}
		for i := 0; i < s.BatchSize; i++ {
			data := make([]byte, DATA_SIZE)
			rand.Read(data)
			dList[i] = data
			wdList[i], err = util.CreateWriteData(data, reader.Ed25519.Point, serverPk, true)
			if err != nil {
				return err
			}
		}
		awt := monitor.NewTimeMeasure("AddWriteTxn")
		for i := 0; i < s.BatchSize; i++ {
			//reply, err := scCl.StoreData(config.Roster, wdList[i].Data, wdList[i].DataHash)
			reply, err := scCl.StoreData(wdList[i].Data, wdList[i].DataHash)
			if err != nil {
				return err
			}
			wdList[i].StoredKey = reply.StoredKey
		}
		for i := 0; i < s.BatchSize; i++ {
			wait := 0
			if i == s.BatchSize-1 {
				wait = 3
			}
			writeTxnList[i], err = scCl.AddWriteTransaction(wdList[i], writer, *wDarc, wait)
			if err != nil {
				return err
			}
		}
		awt.Record()

		wwp := monitor.NewTimeMeasure("WriteGetProof")
		for i := 0; i < s.BatchSize; i++ {
			wrProofResponse, err := scCl.GetProof(writeTxnList[i].InstanceID)
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
			readTxnList[i], err = scCl.AddReadTransaction(wrProofList[i], reader, *wDarc, wait)
			if err != nil {
				return err
			}
		}
		art.Record()

		rwp := monitor.NewTimeMeasure("ReadGetProof")
		for i := 0; i < s.BatchSize; i++ {
			rProofResponse, err := scCl.GetProof(readTxnList[i].InstanceID)
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
			//dr, err := scCl.Decrypt(config.Roster, wrProofList[i], readProofList[i], wdList[i].StoredKey, reader.Ed25519.Secret)
			dr, err := scCl.Decrypt(wrProofList[i], readProofList[i], wdList[i].StoredKey, reader.Ed25519.Secret)
			if err != nil {
				return err
			}
			data, err := util.RecoverData(dr.Data, reader.Ed25519.Secret, dr.K, dr.C)
			if err != nil {
				return err
			}
			log.Info("Data recovered:", bytes.Equal(data, dList[i]))
		}
		decReq.Record()
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

	byzCl, admin, gDarc, err := sc.SetupByzcoin(config.Roster, s.BlockInterval)
	if err != nil {
		log.Errorf("Setting up Byzcoin failed: %v", err)
		return err
	}
	scCl := sc.NewClient(byzCl)
	writer, reader, wDarc, err := scCl.SetupDarcs()
	if err != nil {
		return err
	}
	_, err = scCl.SpawnDarc(admin, *wDarc, gDarc, 4)
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
		//reply, err := scCl.StoreData(config.Roster, fixedWdList[i].Data, fixedWdList[i].DataHash)
		reply, err := scCl.StoreData(fixedWdList[i].Data, fixedWdList[i].DataHash)
		if err != nil {
			return err
		}
		fixedWdList[i].StoredKey = reply.StoredKey
	}
	for i := 0; i < FIXED_COUNT; i++ {
		wait := 0
		if i == FIXED_COUNT-1 {
			wait = 3
		}
		fixedTxnList[i], err = scCl.AddWriteTransaction(fixedWdList[i], writer, *wDarc, wait)
		if err != nil {
			return err
		}
	}
	for i := 0; i < FIXED_COUNT; i++ {
		wrProofResponse, err := scCl.GetProof(fixedTxnList[i].InstanceID)
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
					//reply, err := scCl.StoreData(config.Roster, wdList[writeIdx].Data, wdList[writeIdx].DataHash)
					reply, err := scCl.StoreData(wdList[writeIdx].Data, wdList[writeIdx].DataHash)
					if err != nil {
						return err
					}
					wdList[writeIdx].StoredKey = reply.StoredKey
					writeTxnList[writeIdx], err = scCl.AddWriteTransaction(wdList[writeIdx], writer, *wDarc, wait)
					if err != nil {
						return err
					}
					writeCnt++
					writeIdx++
				} else {
					// READ TXN
					readTxnList[readIdx], err = scCl.AddReadTransaction(fixedProofList[readIdx%FIXED_COUNT], reader, *wDarc, wait)
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
				wrProofResponse, err := scCl.GetProof(writeTxnList[writeIdx-j-1].InstanceID)
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
				rProofResponse, err := scCl.GetProof(readTxnList[readIdx-j].InstanceID)
				if err != nil {
					return err
				}
				rProof := rProofResponse.Proof
				if !rProof.InclusionProof.Match() {
					return errors.New("Read inclusion proof does not match")
				}
				readProofList[readIdx-j] = &rProof
				//dr, err := scCl.Decrypt(config.Roster, fixedProofList[(readIdx-j)%FIXED_COUNT], readProofList[readIdx-j], fixedWdList[(readIdx-j)%FIXED_COUNT].StoredKey, reader.Ed25519.Secret)
				dr, err := scCl.Decrypt(fixedProofList[(readIdx-j)%FIXED_COUNT], readProofList[readIdx-j], fixedWdList[(readIdx-j)%FIXED_COUNT].StoredKey, reader.Ed25519.Secret)
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

func (s *SimulationService) runMultiClientSimulation(config *onet.SimulationConfig, serverPk kyber.Point) error {
	dList := make([][]byte, s.NumTransactions)
	wdList := make([]*util.WriteData, s.NumTransactions)
	writeTxnList := make([]*sc.TransactionReply, s.NumTransactions)
	wrProofList := make([]*byzcoin.Proof, s.NumTransactions)

	for round := 0; round < s.Rounds; round++ {
		log.Lvl1("Starting round", round)
		byzCl, admin, gDarc, err := sc.SetupByzcoin(config.Roster, s.BlockInterval)
		if err != nil {
			log.Errorf("Setting up Byzcoin failed: %v", err)
			return err
		}
		scCl := sc.NewClient(byzCl)
		writer, reader, wDarc, err := scCl.SetupDarcs()
		if err != nil {
			return err
		}
		_, err = scCl.SpawnDarc(admin, *wDarc, gDarc, 4)
		if err != nil {
			return err
		}
		for i := 0; i < s.NumTransactions; i++ {
			data := make([]byte, DATA_SIZE)
			rand.Read(data)
			dList[i] = data
			wdList[i], err = util.CreateWriteData(data, reader.Ed25519.Point, serverPk, true)
			if err != nil {
				return err
			}
		}
		for i := 0; i < s.NumTransactions; i++ {
			//reply, err := scCl.StoreData(config.Roster, wdList[i].Data, wdList[i].DataHash)
			reply, err := scCl.StoreData(wdList[i].Data, wdList[i].DataHash)
			if err != nil {
				return err
			}
			wdList[i].StoredKey = reply.StoredKey
		}
		for i := 0; i < s.NumTransactions; i++ {
			wait := 0
			if i == s.NumTransactions-1 {
				wait = s.BlockWait
			}
			writeTxnList[i], err = scCl.AddWriteTransaction(wdList[i], writer, *wDarc, wait)
			if err != nil {
				return err
			}
		}

		for i := 0; i < s.NumTransactions; i++ {
			wrProofResponse, err := scCl.GetProof(writeTxnList[i].InstanceID)
			if err != nil {
				return err
			}
			wrProof := wrProofResponse.Proof
			if !wrProof.InclusionProof.Match() {
				return errors.New("Write inclusion proof does not match")
			}
			wrProofList[i] = &wrProof
		}
		wg.Add(s.NumTransactions)
		for i := 0; i < s.NumTransactions; i++ {
			byzCl := byzcoin.NewClient(scCl.BcClient.ID, scCl.BcClient.Roster)
			go func(idx int, cl *byzcoin.Client) {
				err := decrypt(idx, cl, wdList[idx], wrProofList[idx], reader, wDarc, dList[idx], s.BlockWait)
				if err != nil {
					log.Errorf("goroutine %d error: %v", idx, err)
				}
			}(i, byzCl)
		}
		wg.Wait()
		log.Info("goroutines are finished")

		//art := monitor.NewTimeMeasure("AddReadTxn")
		//for i := 0; i < s.NumTransactions; i++ {
		//wait := 0
		//if i == s.NumTransactions-1 {
		//wait = 3
		//}
		//readTxnList[i], err = scCl.AddReadTransaction(wrProofList[i], reader, *wDarc, wait)
		//if err != nil {
		//return err
		//}
		//}
		//art.Record()

		//rwp := monitor.NewTimeMeasure("ReadGetProof")
		//for i := 0; i < s.NumTransactions; i++ {
		//rProofResponse, err := scCl.GetProof(readTxnList[i].InstanceID)
		//if err != nil {
		//return err
		//}
		//rProof := rProofResponse.Proof
		//if !rProof.InclusionProof.Match() {
		//return errors.New("Read inclusion proof does not match")
		//}
		//readProofList[i] = &rProof
		//}
		//rwp.Record()

		//decReq := monitor.NewTimeMeasure("DecRequest")
		//for i := 0; i < s.NumTransactions; i++ {
		//dr, err := scCl.Decrypt(config.Roster, wrProofList[i], readProofList[i], wdList[i].StoredKey, reader.Ed25519.Secret)
		//if err != nil {
		//return err
		//}
		//data, err := util.RecoverData(dr.Data, reader.Ed25519.Secret, dr.K, dr.C)
		//if err != nil {
		//return err
		//}
		//log.Info("Data recovered:", bytes.Equal(data, dList[i]))
		//}
		//decReq.Record()
	}
	return nil
}

func decrypt(idx int, bc *byzcoin.Client, wd *util.WriteData, wrProof *byzcoin.Proof, reader darc.Signer, wDarc *darc.Darc, d []byte, wait int) error {
	defer wg.Done()
	scCl := sc.NewClient(bc)
	label := fmt.Sprintf("Client_%d_read", idx+1)

	readMonitor := monitor.NewTimeMeasure(label)
	re, err := scCl.AddReadTransaction(wrProof, reader, *wDarc, wait)
	if err != nil {
		log.Errorf("Read transaction failed @%d: %v", idx, err)
		return err
	}
	rProofResponse, err := scCl.GetProof(re.InstanceID)
	if err != nil {
		log.Errorf("Getting proof failed @%d: %v", idx, err)
		return err
	}
	rProof := rProofResponse.Proof
	if !rProof.InclusionProof.Match() {
		log.Errorf("Read inclusion proof does not match error @%d", idx)
		return errors.New("Read inclusion proof does not match")
	}
	readMonitor.Record()

	label = fmt.Sprintf("Client_%d_decrypt", idx+1)

	decMonitor := monitor.NewTimeMeasure(label)
	dr, err := scCl.Decrypt(wrProof, &rProof, wd.StoredKey, reader.Ed25519.Secret)
	if err != nil {
		return err
	}
	data, err := util.RecoverData(dr.Data, reader.Ed25519.Secret, dr.K, dr.C)
	if err != nil {
		return err
	}
	decMonitor.Record()

	if !bytes.Equal(data, d) {
		return errors.New("Incorrect decryption")
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

	err := s.runMultiClientSimulation(config, serverPk)

	if err != nil {
		log.Info("Simulation error:", err)
		return err
	}
	return nil
}
