package main

import (
	"errors"
	"github.com/BurntSushi/toml"
	lottery "github.com/ceyhunalp/calypso_experiments/calypso_lottery"
	"github.com/dedis/cothority"
	"github.com/dedis/cothority/byzcoin"
	"github.com/dedis/cothority/calypso"
	"github.com/dedis/onet"
	"github.com/dedis/onet/log"
	"github.com/dedis/onet/simul/monitor"
)

/*
 * Defines the simulation for the service-template
 */

func init() {
	onet.SimulationRegister("CalypsoLottery", NewCalypsoLotteryService)
}

// SimulationService only holds the BFTree simulation
type SimulationService struct {
	onet.SimulationBFTree
	NumTransactions int
	NumLotteries    int
}

// NewSimulationService returns the new simulation, where all fields are
// initialised using the config-file
func NewCalypsoLotteryService(config string) (onet.Simulation, error) {
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

func (s *SimulationService) runCalypsoLottery(config *onet.SimulationConfig) error {
	log.Info("Total # of rounds is:", s.Rounds)
	size := config.Tree.Size()
	log.Info("Size of the tree:", size)

	for round := 0; round < s.Rounds; round++ {
		log.Lvl1("Starting round", round)
		byzd, err := lottery.SetupByzcoin(config.Roster)
		if err != nil {
			log.Errorf("Setting up Byzcoin failed: %v", err)
			return err
		}

		calypsoClient := calypso.NewClient(byzd.Cl)
		ltsReply, err := calypsoClient.CreateLTS()
		if err != nil {
			log.Errorf("runCalypsoLottery failed: %v", err)
			return err
		}

		numTransactions := s.NumTransactions
		writerList, reader, writeDarcList, err := lottery.SetupDarcs(numTransactions)
		if err != nil {
			return err
		}
		wait := 0
		for i := 0; i < numTransactions; i++ {
			if i == numTransactions-1 {
				wait = 5
			}
			_, err := byzd.SpawnDarc(*writeDarcList[i], wait)
			if err != nil {
				log.Errorf("SpawnDarc failed: %v", err)
				return err
			}
		}

		lotteryData := make([]*lottery.LotteryData, numTransactions)
		writeTxnData := make([]*calypso.Write, numTransactions)
		//lt := monitor.NewTimeMeasure("calylot_time")
		for i := 0; i < numTransactions; i++ {
			lotteryData[i] = lottery.CreateLotteryData()
			writeTxnData[i] = calypso.NewWrite(cothority.Suite, ltsReply.LTSID, writeDarcList[i].GetBaseID(), ltsReply.X, lotteryData[i].Secret[:])
		}

		wait = 0
		//wait = 3
		writeTxnList := make([]*calypso.WriteReply, numTransactions)
		wt := monitor.NewTimeMeasure("calylot_write")
		for i := 0; i < numTransactions; i++ {
			if i == numTransactions-1 {
				wait = 3
			}
			log.Lvlf1("[CalypsoLottery] AddWrite called")
			writeTxnList[i], err = calypsoClient.AddWrite(writeTxnData[i], writerList[i], *writeDarcList[i], wait)
			if err != nil {
				log.Errorf("AddWrite failed: %v", err)
				return err
			}
		}
		wt.Record()

		writeProofList := make([]byzcoin.Proof, numTransactions)
		wp := monitor.NewTimeMeasure("calylot_write_proof")
		for i := 0; i < numTransactions; i++ {
			wrProofResponse, err := byzd.Cl.GetProof(writeTxnList[i].InstanceID.Slice())
			if err != nil {
				log.Errorf("GetProof(Write) failed: %v", err)
				return err
			}
			if !wrProofResponse.Proof.InclusionProof.Match() {
				return errors.New("Write inclusion proof does not match")
			}
			writeProofList[i] = wrProofResponse.Proof
		}
		wp.Record()

		wait = 0
		readTxnList := make([]*calypso.ReadReply, numTransactions)
		//clr := monitor.NewTimeMeasure("calylot_read")
		for i := 0; i < numTransactions; i++ {
			if i == numTransactions-1 {
				wait = 3
			}
			log.Lvl1("[CalypsoLottery] AddRead called")
			readTxnList[i], err = calypsoClient.AddRead(&writeProofList[i], reader, *writeDarcList[i], wait)
			if err != nil {
				log.Errorf("AddRead failed: %v", err)
				return err
			}
		}
		//clr.Record()
		readProofList := make([]byzcoin.Proof, numTransactions)
		//crp := monitor.NewTimeMeasure("calylot_read_proof")
		for i := 0; i < numTransactions; i++ {
			rProofResponse, err := byzd.Cl.GetProof(readTxnList[i].InstanceID.Slice())
			if err != nil {
				log.Errorf("GetProof(Read) failed: %v", err)
				return err
			}
			if !rProofResponse.Proof.InclusionProof.Match() {
				return errors.New("Read inclusion proof does not match")
			}
			readProofList[i] = rProofResponse.Proof
		}
		//crp.Record()
		decodedSecretList := make([][]byte, numTransactions)
		//dk := monitor.NewTimeMeasure("calylot_decode")
		for i := 0; i < numTransactions; i++ {
			dk, err := calypsoClient.DecryptKey(&calypso.DecryptKey{Read: readProofList[i], Write: writeProofList[i]})
			if err != nil {
				log.Errorf("DecryptKey failed: %v", err)
				return err
			}
			if !dk.X.Equal(ltsReply.X) {
				return errors.New("Points not same")
			}

			decodedSecretList[i], err = calypso.DecodeKey(cothority.Suite, ltsReply.X, dk.Cs, dk.XhatEnc, reader.Ed25519.Secret)
			if err != nil {
				log.Errorf("DecodeKey failed: %v", err)
				return err
			}
		}
		result := make([]byte, 32)
		for i := 0; i < numTransactions; i++ {
			lottery.SafeXORBytes(result, result, decodedSecretList[i])
		}
		lastDigit := int(result[31])
		//dk.Record()
		//lt.Record()
		log.Info("Winner is:", lastDigit%numTransactions)
	}
	return nil
}

func (s *SimulationService) runMultipleLottery(config *onet.SimulationConfig, byzd *lottery.ByzcoinData) error {
	calypsoClient := calypso.NewClient(byzd.Cl)
	ltsReply, err := calypsoClient.CreateLTS()
	if err != nil {
		log.Errorf("runCalypsoLottery failed: %v", err)
		return err
	}

	numTransactions := s.NumTransactions
	writerList, reader, writeDarcList, err := lottery.SetupDarcs(numTransactions)
	if err != nil {
		return err
	}
	wait := 0
	for i := 0; i < numTransactions; i++ {
		if i == numTransactions-1 {
			wait = 3
		}
		_, err := byzd.SpawnDarc(*writeDarcList[i], wait)
		if err != nil {
			log.Errorf("SpawnDarc failed: %v", err)
			return err
		}
	}

	lotteryData := make([]*lottery.LotteryData, numTransactions)
	writeTxnData := make([]*calypso.Write, numTransactions)
	for i := 0; i < numTransactions; i++ {
		lotteryData[i] = lottery.CreateLotteryData()
		writeTxnData[i] = calypso.NewWrite(cothority.Suite, ltsReply.LTSID, writeDarcList[i].GetBaseID(), ltsReply.X, lotteryData[i].Secret[:])
	}

	wait = 0
	writeTxnList := make([]*calypso.WriteReply, numTransactions)
	//wt := monitor.NewTimeMeasure("calylot_write")
	for i := 0; i < numTransactions; i++ {
		//if i == numTransactions-1 {
		//wait = 5
		//}
		//log.Lvlf1("[CalypsoLottery] AddWrite called")
		writeTxnList[i], err = calypsoClient.AddWrite(writeTxnData[i], writerList[i], *writeDarcList[i], wait)
		if err != nil {
			log.Errorf("AddWrite failed: %v", err)
			return err
		}
	}
	//wt.Record()

	///////////////////////////////////////////////////////////////////////

	writeProofReady := false
	writeProofList := make([]byzcoin.Proof, numTransactions)
	for writeProofReady == false {
		wrProofResponse, err := byzd.Cl.GetProof(writeTxnList[numTransactions-1].InstanceID.Slice())
		if err != nil {
			log.Errorf("GetProof(Write) failed: %v", err)
			return err
		}
		if wrProofResponse.Proof.InclusionProof.Match() {
			writeProofList[numTransactions-1] = wrProofResponse.Proof
			writeProofReady = true
			//return errors.New("Write inclusion proof does not match")
		} else {
			log.Lvl3("Write inclusion proof does not match")
		}
	}
	for i := 0; i < numTransactions-1; i++ {
		wrProofResponse, err := byzd.Cl.GetProof(writeTxnList[i].InstanceID.Slice())
		if err != nil {
			log.Errorf("GetProof(Write) failed: %v", err)
			return err
		}
		if !wrProofResponse.Proof.InclusionProof.Match() {
			return errors.New("Write inclusion proof does not match")
		}
		writeProofList[i] = wrProofResponse.Proof
	}

	///////////////////////////////////////////////////////////////////////

	//writeProofList := make([]byzcoin.Proof, numTransactions)
	//wp := monitor.NewTimeMeasure("calylot_write_proof")
	//for i := 0; i < numTransactions; i++ {
	//wrProofResponse, err := byzd.Cl.GetProof(writeTxnList[i].InstanceID.Slice())
	//if err != nil {
	//log.Errorf("GetProof(Write) failed: %v", err)
	//return err
	//}
	//if !wrProofResponse.Proof.InclusionProof.Match() {
	//return errors.New("Write inclusion proof does not match")
	//}
	//writeProofList[i] = wrProofResponse.Proof
	//}
	//wp.Record()

	///////////////////////////////////////////////////////////////////////

	wait = 0
	readTxnList := make([]*calypso.ReadReply, numTransactions)
	//clr := monitor.NewTimeMeasure("calylot_read")
	for i := 0; i < numTransactions; i++ {
		//if i == numTransactions-1 {
		//wait = 5
		//}
		//log.Lvl1("[CalypsoLottery] AddRead called")
		readTxnList[i], err = calypsoClient.AddRead(&writeProofList[i], reader, *writeDarcList[i], wait)
		if err != nil {
			log.Errorf("AddRead failed: %v", err)
			return err
		}
	}
	//clr.Record()

	readProofReady := false
	readProofList := make([]byzcoin.Proof, numTransactions)
	for readProofReady == false {
		rProofResponse, err := byzd.Cl.GetProof(readTxnList[numTransactions-1].InstanceID.Slice())
		if err != nil {
			log.Errorf("GetProof(Read) failed: %v", err)
			return err
		}
		if rProofResponse.Proof.InclusionProof.Match() {
			readProofList[numTransactions-1] = rProofResponse.Proof
			readProofReady = true
			//return errors.New("Read inclusion proof does not match")
		} else {
			log.Lvl3("Read inclusion proof does not match")
		}
	}
	for i := 0; i < numTransactions-1; i++ {
		rProofResponse, err := byzd.Cl.GetProof(readTxnList[i].InstanceID.Slice())
		if err != nil {
			log.Errorf("GetProof(Read) failed: %v", err)
			return err
		}
		if !rProofResponse.Proof.InclusionProof.Match() {
			return errors.New("Read inclusion proof does not match")
		}
		readProofList[i] = rProofResponse.Proof
	}

	///////////////////////////////////////////////////////////////////////

	//readProofList := make([]byzcoin.Proof, numTransactions)
	//crp := monitor.NewTimeMeasure("calylot_read_proof")
	//for i := 0; i < numTransactions; i++ {
	//rProofResponse, err := byzd.Cl.GetProof(readTxnList[i].InstanceID.Slice())
	//if err != nil {
	//log.Errorf("GetProof(Read) failed: %v", err)
	//return err
	//}
	//if !rProofResponse.Proof.InclusionProof.Match() {
	//return errors.New("Read inclusion proof does not match")
	//}
	//readProofList[i] = rProofResponse.Proof
	//}
	//crp.Record()

	///////////////////////////////////////////////////////////////////////

	decodedSecretList := make([][]byte, numTransactions)
	//dk := monitor.NewTimeMeasure("calylot_decode")
	for i := 0; i < numTransactions; i++ {
		dk, err := calypsoClient.DecryptKey(&calypso.DecryptKey{Read: readProofList[i], Write: writeProofList[i]})
		if err != nil {
			log.Errorf("DecryptKey failed: %v", err)
			return err
		}
		if !dk.X.Equal(ltsReply.X) {
			return errors.New("Points not same")
		}

		decodedSecretList[i], err = calypso.DecodeKey(cothority.Suite, ltsReply.X, dk.Cs, dk.XhatEnc, reader.Ed25519.Secret)
		if err != nil {
			log.Errorf("DecodeKey failed: %v", err)
			return err
		}
	}
	result := make([]byte, 32)
	for i := 0; i < numTransactions; i++ {
		lottery.SafeXORBytes(result, result, decodedSecretList[i])
	}
	lastDigit := int(result[31])
	//dk.Record()
	//lt.Record()
	log.Info("Winner is:", lastDigit%numTransactions)

	return nil
}

// Run is used on the destination machines and runs a number of
// rounds
func (s *SimulationService) Run(config *onet.SimulationConfig) error {
	return s.runCalypsoLottery(config)

	//byzd, err := lottery.SetupByzcoin(config.Roster)
	//if err != nil {
	//log.Errorf("Setting up Byzcoin failed: %v", err)
	//return err
	//}
	//mlt := monitor.NewTimeMeasure("multiple_lottery")
	//for i := 1; i < s.NumLotteries; i++ {
	//go s.runMultipleLottery(config, byzd)
	//}
	//s.runMultipleLottery(config, byzd)
	//mlt.Record()
	//return nil
}
