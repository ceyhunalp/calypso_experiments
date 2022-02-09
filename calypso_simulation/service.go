package main

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"os"
	"strconv"
	"sync"
	"time"

	"github.com/BurntSushi/toml"
	"github.com/dedis/cothority"
	"github.com/dedis/cothority/byzcoin"
	"github.com/dedis/cothority/calypso"
	"github.com/dedis/cothority/darc"
	"github.com/dedis/cothority/darc/expression"
	"github.com/dedis/kyber"
	"github.com/dedis/kyber/util/random"
	"github.com/dedis/onet"
	"github.com/dedis/onet/log"
	"github.com/dedis/onet/simul/monitor"
)

const FIXED_COUNT int = 10

var wg sync.WaitGroup

type ByzcoinData struct {
	Signer darc.Signer
	Roster *onet.Roster
	Cl     *byzcoin.Client
	GMsg   *byzcoin.CreateGenesisBlock
	GDarc  *darc.Darc
	Csr    *byzcoin.CreateGenesisBlockResponse
}

// SimulationService only holds the BFTree simulation
type SimulationService struct {
	onet.SimulationBFTree
	NumTransactions      int
	NumWriteTransactions int
	NumReadTransactions  int
	NumBlocks            int
	BlockInterval        int
	BlockWait            int
}

func init() {
	onet.SimulationRegister("Calypso", NewCalypsoService)
}

// NewSimulationService returns the new simulation, where all fields are
// initialised using the config-file
func NewCalypsoService(config string) (onet.Simulation, error) {
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

func setupDarcs() (darc.Signer, darc.Signer, *darc.Darc, error) {
	writer := darc.NewSignerEd25519(nil, nil)
	reader := darc.NewSignerEd25519(nil, nil)
	writeDarc := darc.NewDarc(darc.InitRules([]darc.Identity{writer.Identity()},
		[]darc.Identity{writer.Identity()}), []byte("Writer"))
	writeDarc.Rules.AddRule(darc.Action("spawn:"+calypso.ContractWriteID),
		expression.InitOrExpr(writer.Identity().String()))
	writeDarc.Rules.AddRule(darc.Action("spawn:"+calypso.ContractReadID),
		expression.InitOrExpr(reader.Identity().String()))
	return writer, reader, writeDarc, nil
}

func setupByzcoin(r *onet.Roster, blockInterval int) (*ByzcoinData, error) {
	var err error
	byzd := &ByzcoinData{}
	byzd.Signer = darc.NewSignerEd25519(nil, nil)
	byzd.GMsg, err = byzcoin.DefaultGenesisMsg(byzcoin.CurrentVersion, r, []string{"spawn:" + byzcoin.ContractDarcID}, byzd.Signer.Identity())
	if err != nil {
		log.Errorf("SetupByzcoin error: %v", err)
		return nil, err
	}
	byzd.GMsg.BlockInterval = time.Duration(blockInterval) * time.Second
	byzd.GDarc = &byzd.GMsg.GenesisDarc
	byzd.Cl, _, err = byzcoin.NewLedger(byzd.GMsg, false)
	if err != nil {
		log.Errorf("SetupByzcoin error: %v", err)
		return nil, err
	}
	return byzd, nil
}
func (s *SimulationService) runByzgenSimulation(config *onet.SimulationConfig) error {
	txnList := make([]int, s.NumTransactions)
	blkSizeList := make([]int, s.NumBlocks)
	log.Info("Number of transactions:", s.NumTransactions)
	log.Info("Number of blocks:", s.NumBlocks)
	err := readAuxFile(txnList, blkSizeList)
	if err != nil {
		log.Info("Error in readAux:", err)
		return err
	}

	writeList := make([]*calypso.Write, FIXED_COUNT)
	fixedWriteTxnList := make([]*calypso.WriteReply, FIXED_COUNT)
	fixedWrProofList := make([]*byzcoin.Proof, FIXED_COUNT)

	writeTxnList := make([]*calypso.WriteReply, s.NumWriteTransactions)

	readTxnList := make([]*calypso.ReadReply, s.NumReadTransactions)
	readProofList := make([]*byzcoin.Proof, s.NumReadTransactions)

	byzd, err := setupByzcoin(config.Roster, s.BlockInterval)
	if err != nil {
		return err
	}

	calypsoClient := calypso.NewClient(byzd.Cl)
	ltsReply, err := calypsoClient.CreateLTS()
	if err != nil {
		return err
	}
	writer, reader, writeDarc, err := setupDarcs()
	if err != nil {
		return err
	}
	_, err = calypsoClient.SpawnDarc(byzd.Signer, *byzd.GDarc, *writeDarc, 3)
	if err != nil {
		return err
	}
	for i := 0; i < FIXED_COUNT; i++ {
		wait := 0
		if i == FIXED_COUNT-1 {
			wait = 3
		}
		var key [16]byte
		random.Bytes(key[:], random.New())
		writeList[i] = calypso.NewWrite(cothority.Suite, ltsReply.LTSID, writeDarc.GetBaseID(), ltsReply.X, key[:])
		fixedWriteTxnList[i], err = calypsoClient.AddWrite(writeList[i], writer, *writeDarc, wait)
	}
	for i := 0; i < FIXED_COUNT; i++ {
		wrProofResponse, err := byzd.Cl.GetProof(fixedWriteTxnList[i].InstanceID.Slice())
		if err != nil {
			return err
		}
		wrProof := wrProofResponse.Proof
		if !wrProof.InclusionProof.Match() {
			return errors.New("Write inclusion proof does not match")
		}
		fixedWrProofList[i] = &wrProof
	}

	for round := 0; round < s.Rounds; round++ {
		log.Lvl1("Starting round", round)

		txnIdx := 0
		blkSizeIdx := 0
		writeIdx := 0
		readIdx := 0
		fixedIdx := 0

		for txnIdx < s.NumTransactions {
			log.Info("Transaction #:", txnIdx)
			blkSize := blkSizeList[blkSizeIdx]
			writeCnt := 0
			readCnt := 0
			measureStr := "Block_" + strconv.Itoa(blkSizeIdx)
			blktime := monitor.NewTimeMeasure(measureStr)
			for i := 0; i < blkSize; i++ {
				wait := 0
				if i == blkSize-1 {
					wait = 3
				}
				if txnList[txnIdx] == 1 {
					// WRITE TXN
					writeCnt++
					writeTxnList[writeIdx], err = calypsoClient.AddWrite(writeList[fixedIdx%FIXED_COUNT], writer, *writeDarc, wait)
					if err != nil {
						return err
					}
					fixedIdx++
					writeIdx++
				} else {
					// READ TXN
					readCnt++
					readTxnList[readIdx], err = calypsoClient.AddRead(fixedWrProofList[readIdx%FIXED_COUNT], reader, *writeDarc, wait)
					if err != nil {
						return err
					}
					readIdx++
				}
				txnIdx++
			}
			blktime.Record()

			wpt := monitor.NewTimeMeasure("WriteProof")
			for j := 0; j < writeCnt; j++ {
				wrProofResponse, err := byzd.Cl.GetProof(writeTxnList[writeIdx-j-1].InstanceID.Slice())
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
				rProofResponse, err := byzd.Cl.GetProof(readTxnList[readIdx-j].InstanceID.Slice())
				if err != nil {
					return err
				}
				rProof := rProofResponse.Proof
				if !rProof.InclusionProof.Match() {
					return errors.New("Read inclusion proof does not match")
				}
				readProofList[readIdx-j] = &rProof

				dk, err := calypsoClient.DecryptKey(&calypso.DecryptKey{Read: *readProofList[readIdx-j], Write: *fixedWrProofList[(readIdx-j)%FIXED_COUNT]})
				if err != nil {
					return err
				}
				if !dk.X.Equal(ltsReply.X) {
					return errors.New("Points not same")
				}
				_, err = calypso.DecodeKey(cothority.Suite, ltsReply.X, dk.Cs, dk.XhatEnc, reader.Ed25519.Secret)
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

func (s *SimulationService) runMicrobenchmark(config *onet.SimulationConfig) error {
	writeList := make([]*calypso.Write, s.NumTransactions)
	writeTxnList := make([]*calypso.WriteReply, s.NumTransactions)
	wrProofList := make([]*byzcoin.Proof, s.NumTransactions)
	readTxnList := make([]*calypso.ReadReply, s.NumTransactions)
	readProofList := make([]*byzcoin.Proof, s.NumTransactions)

	for round := 0; round < s.Rounds; round++ {
		log.Lvl1("Starting round", round)
		byzd, err := setupByzcoin(config.Roster, s.BlockInterval)
		if err != nil {
			return err
		}
		calypsoClient := calypso.NewClient(byzd.Cl)
		ltsReply, err := calypsoClient.CreateLTS()
		if err != nil {
			return err
		}
		writer, reader, writeDarc, err := setupDarcs()
		if err != nil {
			return err
		}
		_, err = calypsoClient.SpawnDarc(byzd.Signer, *byzd.GDarc, *writeDarc, 3)
		if err != nil {
			return err
		}
		for i := 0; i < s.NumTransactions; i++ {
			var key [16]byte
			random.Bytes(key[:], random.New())
			writeList[i] = calypso.NewWrite(cothority.Suite, ltsReply.LTSID, writeDarc.GetBaseID(), ltsReply.X, key[:])
		}
		awm := monitor.NewTimeMeasure("AddWriteTxn")
		for i := 0; i < s.NumTransactions; i++ {
			wait := 0
			if i == s.NumTransactions-1 {
				wait = 3
			}
			writeTxnList[i], err = calypsoClient.AddWrite(writeList[i], writer, *writeDarc, wait)
			if err != nil {
				return err
			}
		}
		awm.Record()
		wgp := monitor.NewTimeMeasure("WriteGetProof")
		for i := 0; i < s.NumTransactions; i++ {
			wrProofResponse, err := byzd.Cl.GetProof(writeTxnList[i].InstanceID.Slice())
			if err != nil {
				return err
			}
			wrProof := wrProofResponse.Proof
			if !wrProof.InclusionProof.Match() {
				return errors.New("Write inclusion proof does not match")
			}
			wrProofList[i] = &wrProof
		}
		wgp.Record()
		arm := monitor.NewTimeMeasure("AddReadTxn")
		for i := 0; i < s.NumTransactions; i++ {
			wait := 0
			if i == s.NumTransactions-1 {
				wait = 3
			}
			readTxnList[i], err = calypsoClient.AddRead(wrProofList[i], reader, *writeDarc, wait)
			if err != nil {
				return err
			}
		}
		arm.Record()
		rgp := monitor.NewTimeMeasure("ReadGetProof")
		for i := 0; i < s.NumTransactions; i++ {
			rProofResponse, err := byzd.Cl.GetProof(readTxnList[i].InstanceID.Slice())
			if err != nil {
				return err
			}
			rProof := rProofResponse.Proof
			if !rProof.InclusionProof.Match() {
				return errors.New("Read inclusion proof does not match")
			}
			readProofList[i] = &rProof
		}
		rgp.Record()

		dkm := monitor.NewTimeMeasure("DecryptKey")
		for i := 0; i < s.NumTransactions; i++ {
			dk, err := calypsoClient.DecryptKey(&calypso.DecryptKey{Read: *readProofList[i], Write: *wrProofList[i]})
			if err != nil {
				return err
			}
			if !dk.X.Equal(ltsReply.X) {
				return errors.New("Points not same")
			}

			_, err = calypso.DecodeKey(cothority.Suite, ltsReply.X, dk.Cs, dk.XhatEnc, reader.Ed25519.Secret)
			if err != nil {
				return err
			}
			//log.Info("Keys are equal: ", bytes.Equal(decodedKey, key[:]))
		}
		dkm.Record()
	}
	return nil
}

func decrypt(idx int, bc *byzcoin.Client, X kyber.Point, key [16]byte, wProof *byzcoin.Proof, reader darc.Signer, wDarc *darc.Darc, wait int) error {
	defer wg.Done()
	cc := calypso.NewClient(bc)
	label := fmt.Sprintf("Client_%d_read", idx+1)

	readMonior := monitor.NewTimeMeasure(label)
	re, err := cc.AddRead(wProof, reader, *wDarc, wait)
	if err != nil {
		log.Errorf("Read transaction failed @%d: %v", idx, err)
		return err
	}
	rProofResponse, err := bc.GetProof(re.InstanceID.Slice())
	if err != nil {
		log.Errorf("Getting proof failed @%d: %v", idx, err)
		return err
	}
	rProof := rProofResponse.Proof
	if !rProof.InclusionProof.Match() {
		log.Errorf("Read inclusion proof does not match error @%d", idx)
		return errors.New("Read inclusion proof does not match")
	}
	readMonior.Record()
	//var header byzcoin.DataHeader
	//err = protobuf.Decode(rProof.Latest.Data, &header)
	//if err != nil {
	//log.Errorf("pbuf decode: %v", err)
	//}
	//log.Info(rProof.Latest.Index, header.Timestamp/1e6)

	label = fmt.Sprintf("Client_%d_decrypt", idx+1)
	decMonitor := monitor.NewTimeMeasure(label)
	dk, err := cc.DecryptKey(&calypso.DecryptKey{Read: rProof, Write: *wProof})
	if err != nil {
		log.Errorf("Decrypting key failed @%d: %v", idx, err)
		return err
	}
	decodedKey, err := calypso.DecodeKey(cothority.Suite, X, dk.Cs, dk.XhatEnc, reader.Ed25519.Secret)
	if err != nil {
		return err
	}
	decMonitor.Record()

	if !bytes.Equal(decodedKey, key[:]) {
		return errors.New("Keys don't match")
	}
	return nil
}

func (s *SimulationService) runMultiClientCalypso(config *onet.SimulationConfig) error {
	keyList := make([][16]byte, s.NumTransactions)
	writeList := make([]*calypso.Write, s.NumTransactions)
	writeTxnList := make([]*calypso.WriteReply, s.NumTransactions)
	wrProofList := make([]*byzcoin.Proof, s.NumTransactions)

	for round := 0; round < s.Rounds; round++ {
		log.Lvl1("Starting round", round)
		byzd, err := setupByzcoin(config.Roster, s.BlockInterval)
		if err != nil {
			return err
		}
		calypsoClient := calypso.NewClient(byzd.Cl)
		ltsReply, err := calypsoClient.CreateLTS()
		if err != nil {
			return err
		}
		writer, reader, writeDarc, err := setupDarcs()
		if err != nil {
			return err
		}
		//_, err = calypsoClient.SpawnDarc(byzd.Signer, *byzd.GDarc, *writeDarc, 3)
		_, err = calypsoClient.SpawnDarc(byzd.Signer, *byzd.GDarc, *writeDarc, s.BlockWait)
		if err != nil {
			return err
		}
		for i := 0; i < s.NumTransactions; i++ {
			var key [16]byte
			random.Bytes(key[:], random.New())
			writeList[i] = calypso.NewWrite(cothority.Suite, ltsReply.LTSID, writeDarc.GetBaseID(), ltsReply.X, key[:])
			keyList[i] = key
		}
		for i := 0; i < s.NumTransactions; i++ {
			wait := 0
			if i == s.NumTransactions-1 {
				wait = s.BlockWait
			}
			writeTxnList[i], err = calypsoClient.AddWrite(writeList[i], writer, *writeDarc, wait)
			if err != nil {
				return err
			}
		}
		for i := 0; i < s.NumTransactions; i++ {
			wrProofResponse, err := byzd.Cl.GetProof(writeTxnList[i].InstanceID.Slice())
			if err != nil {
				return err
			}
			wrProof := wrProofResponse.Proof
			if !wrProof.InclusionProof.Match() {
				return errors.New("Write inclusion proof does not match")
			}
			//var header byzcoin.DataHeader
			//err = protobuf.Decode(wrProof.Latest.Data, &header)
			//if err != nil {
			//log.Errorf("pbuf decode: %v", err)
			//}
			//log.Info("write proof index:", wrProof.Latest.Index, header.Timestamp/1e6)
			wrProofList[i] = &wrProof
		}
		wg.Add(s.NumTransactions)
		for i := 0; i < s.NumTransactions; i++ {
			byzCl := byzcoin.NewClient(byzd.Cl.ID, byzd.Cl.Roster)
			go func(idx int, cl *byzcoin.Client) {
				err := decrypt(idx, cl, ltsReply.X, keyList[idx], wrProofList[idx], reader, writeDarc, s.BlockWait)
				if err != nil {
					log.Errorf("goroutine %d error: %v", idx, err)
				}
			}(i, byzCl)
		}
		wg.Wait()
		log.Info("goroutines are finished")
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

func countTransactions(txnList []int, base int, sz int) (int, int) {
	write := 0
	read := 0
	for i := 0; i < sz; i++ {
		if txnList[base+i] == 1 {
			write++
		} else {
			read++
		}
	}
	log.Info("wcount, rcount:", write, read)
	return write, read
}

// Run is used on ehe destination machines and runs a number of
// rounds
func (s *SimulationService) Run(config *onet.SimulationConfig) error {
	//size := config.Tree.Size()
	//log.Lvl2("Size is:", size, "rounds:", s.Rounds)
	//log.Info("Roster size is:", len(config.Roster.List))

	err := s.runMultiClientCalypso(config)
	if err != nil {
		log.Info("Returned with error:", err)
		return err
	}
	time.Sleep(10 * time.Second)
	return nil
}
