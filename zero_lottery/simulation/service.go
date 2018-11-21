package main

import (
	"bytes"
	"crypto/sha256"
	"errors"
	"fmt"
	"github.com/BurntSushi/toml"
	zerolottery "github.com/ceyhunalp/centralized_calypso/zero_lottery"
	"github.com/dedis/cothority"
	"github.com/dedis/cothority/byzcoin"
	"github.com/dedis/onet"
	"github.com/dedis/onet/log"
	"github.com/dedis/onet/simul/monitor"
	"math"
)

/*
 * Defines the simulation for the service-template
 */

func init() {
	onet.SimulationRegister("ZeroLottery", NewZeroLotteryService)
}

// SimulationService only holds the BFTree simulation
type SimulationService struct {
	onet.SimulationBFTree
	NumParticipant int
}

// NewSimulationService returns the new simulation, where all fields are
// initialised using the config-file
func NewZeroLotteryService(config string) (onet.Simulation, error) {
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
		byzd, err := zerolottery.SetupByzcoin(config.Roster)
		numParticipant := s.NumParticipant
		numRounds := int(math.Log2(float64(numParticipant)))
		numParticipantLeft := numParticipant
		participantList := make([]int, numParticipant)
		for i := 0; i < numParticipant; i++ {
			participantList[i] = 1
		}

		lt := monitor.NewTimeMeasure("zero_lottery_time")
		for i := 0; i < numRounds; i++ {
			lotteryData := make([]*zerolottery.LotteryData, numParticipantLeft)
			commitTxnList := make([]*zerolottery.TransactionReply, numParticipantLeft)
			for i := 0; i < numParticipantLeft; i++ {
				lotteryData[i] = zerolottery.CreateLotteryData()
				wait := 0
				if i == numParticipantLeft-1 {
					wait = 3
				}
				commitTxnList[i], err = byzd.AddCommitTransaction(lotteryData[i], wait)
				if err != nil {
					log.Errorf("AddCommitTransaction failed: %v", err)
					return err
				}
			}

			commitProofList := make([]byzcoin.Proof, numParticipantLeft)
			for i := 0; i < numParticipantLeft; i++ {
				commitProofResp, err := byzd.Cl.GetProof(commitTxnList[i].InstanceID.Slice())
				if err != nil {
					log.Errorf("GetProof(Commit) failed: %v", err)
					return err
				}
				if !commitProofResp.Proof.InclusionProof.Match() {
					return errors.New("Commit inclusion proof does not match")
				}
				commitProofList[i] = commitProofResp.Proof
			}

			secretTxnList := make([]*zerolottery.TransactionReply, numParticipantLeft)
			for i := 0; i < numParticipantLeft; i++ {
				wait := 0
				if i == numParticipantLeft-1 {
					wait = 3
				}
				secretTxnList[i], err = byzd.AddSecretTransaction(lotteryData[i], wait)
				if err != nil {
					log.Errorf("AddSecretTransaction failed: %v", err)
					return err
				}
			}

			secretProofList := make([]byzcoin.Proof, numParticipantLeft)
			for i := 0; i < numParticipantLeft; i++ {
				secretProofResp, err := byzd.Cl.GetProof(secretTxnList[i].InstanceID.Slice())
				if err != nil {
					log.Errorf("GetProof(Secret) failed: %v", err)
					return err
				}
				if !secretProofResp.Proof.InclusionProof.Match() {
					return errors.New("Secret inclusion proof does not match")
				}
				secretProofList[i] = secretProofResp.Proof
			}

			revealedCommitList := make([]zerolottery.DataStore, numParticipantLeft)
			revealedSecretList := make([]zerolottery.DataStore, numParticipantLeft)
			for i := 0; i < numParticipantLeft; i++ {
				err = commitProofList[i].ContractValue(cothority.Suite, zerolottery.ContractLotteryStoreID, &revealedCommitList[i])
				if err != nil {
					log.Errorf("did not get a commit instance" + err.Error())
					return errors.New("did not get a commit instance" + err.Error())
				}
				err = secretProofList[i].ContractValue(cothority.Suite, zerolottery.ContractLotteryStoreID, &revealedSecretList[i])
				if err != nil {
					log.Errorf("did not get a secret instance" + err.Error())
					return errors.New("did not get a secret instance" + err.Error())
				}
			}

			var winnerList []int
			for i := 0; i < numParticipantLeft; {
				//These are the hashes
				leftSecret := revealedSecretList[i].Data
				rightSecret := revealedSecretList[i+1].Data
				leftDigest := sha256.Sum256(leftSecret[:])
				rightDigest := sha256.Sum256(rightSecret[:])
				if bytes.Compare(leftDigest[:], revealedCommitList[i].Data[:]) != 0 {
					fmt.Println("Digests do not match - winner is", i+1)
					winnerList = append(winnerList, i+1)
				} else {
					if bytes.Compare(rightDigest[:], revealedCommitList[i+1].Data[:]) != 0 {
						fmt.Println("Digests do not match - winner is", i)
						winnerList = append(winnerList, i)
					} else {
						result := make([]byte, 32)
						zerolottery.SafeXORBytes(result, leftSecret[:], rightSecret[:])
						lastDigit := int(result[31]) % 2
						if lastDigit == 0 {
							winnerList = append(winnerList, i)
						} else {
							winnerList = append(winnerList, i+1)
						}
					}
				}
				i += 2
			}
			zerolottery.OrganizeList(participantList, winnerList)
			numParticipantLeft = numParticipantLeft / 2
		}
		lt.Record()
		for i := 0; i < numParticipant; i++ {
			if participantList[i] == 1 {
				fmt.Println("Winner is", i)
				break
			}
		}
	}
	return nil
}
