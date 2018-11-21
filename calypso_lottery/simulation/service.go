package main

import (
	"errors"
	"github.com/BurntSushi/toml"
	lottery "github.com/ceyhunalp/centralized_calypso/calypso_lottery"
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
	NumParticipant int
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

// Run is used on the destination machines and runs a number of
// rounds
func (s *SimulationService) Run(config *onet.SimulationConfig) error {
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

		numParticipant := s.NumParticipant
		writerList, reader, writeDarcList, err := lottery.SetupDarcs(numParticipant)
		if err != nil {
			return err
		}
		for i := 0; i < numParticipant; i++ {
			wait := 0
			if i == numParticipant-1 {
				wait = 3
			}
			_, err := byzd.SpawnDarc(*writeDarcList[i], wait)
			if err != nil {
				log.Errorf("SpawnDarc failed: %v", err)
				return err
			}
		}

		lt := monitor.NewTimeMeasure("lottery_time")
		lotteryData := make([]*lottery.LotteryData, numParticipant)
		writeTxnData := make([]*calypso.Write, numParticipant)
		for i := 0; i < numParticipant; i++ {
			lotteryData[i] = lottery.CreateLotteryData()
			writeTxnData[i] = calypso.NewWrite(cothority.Suite, ltsReply.LTSID, writeDarcList[i].GetBaseID(), ltsReply.X, lotteryData[i].Secret[:])
		}

		//log.Info("Starting addwrite")
		writeTxnList := make([]*calypso.WriteReply, numParticipant)
		for i := 0; i < numParticipant; i++ {
			wait := 0
			if i == numParticipant-1 {
				wait = 3
			}
			writeTxnList[i], err = calypsoClient.AddWrite(writeTxnData[i], writerList[i], *writeDarcList[i], wait)
			if err != nil {
				log.Errorf("AddWrite failed: %v", err)
				return err
			}
		}
		//log.Info("addwrite finished")

		writeProofList := make([]byzcoin.Proof, numParticipant)
		for i := 0; i < numParticipant; i++ {
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

		//log.Info("Starting addread")
		readTxnList := make([]*calypso.ReadReply, numParticipant)
		for i := 0; i < numParticipant; i++ {
			wait := 0
			if i == numParticipant-1 {
				wait = 3
			}
			readTxnList[i], err = calypsoClient.AddRead(&writeProofList[i], reader, *writeDarcList[i], wait)
			if err != nil {
				log.Errorf("AddRead failed: %v", err)
				return err
			}
		}
		//log.Info("addread finished")
		readProofList := make([]byzcoin.Proof, numParticipant)
		for i := 0; i < numParticipant; i++ {
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

		decodedSecretList := make([][]byte, numParticipant)
		for i := 0; i < numParticipant; i++ {
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
		for i := 0; i < numParticipant; i++ {
			lottery.SafeXORBytes(result, result, decodedSecretList[i])
		}

		lastDigit := int(result[31])
		log.Info("Winner is:", lastDigit%numParticipant)
		lt.Record()
		//fmt.Println("XOR result:", result)
		//fmt.Println("Last digit is:", lastDigit)
	}
	return nil
}
