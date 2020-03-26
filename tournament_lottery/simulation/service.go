package main

import (
	"bytes"
	"crypto/sha256"
	"errors"
	"fmt"
	"github.com/BurntSushi/toml"
	tournament "github.com/ceyhunalp/calypso_experiments/tournament_lottery"
	"go.dedis.ch/cothority"
	"go.dedis.ch/cothority/byzcoin"
	"go.dedis.ch/onet"
	"go.dedis.ch/onet/log"
	"go.dedis.ch/onet/simul/monitor"
	"math"
)

/*
 * Defines the simulation for the service-template
 */

func init() {
	onet.SimulationRegister("TournamentLottery", NewTournamentService)
}

// SimulationService only holds the BFTree simulation
type SimulationService struct {
	onet.SimulationBFTree
	NumTransactions int
}

// NewSimulationService returns the new simulation, where all fields are
// initialised using the config-file
func NewTournamentService(config string) (onet.Simulation, error) {
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
		byzd, err := tournament.SetupByzcoin(config.Roster)
		numTransactions := s.NumTransactions
		//numRounds := 1
		numRounds := int(math.Ceil(math.Log2(float64(numTransactions))))
		numTransactionsLeft := numTransactions
		participantList := make([]int, numTransactions)
		for i := 0; i < numTransactions; i++ {
			participantList[i] = 1
		}

		isOdd := false
		fmt.Println("Number of rounds is:", numRounds)
		for i := 0; i < numRounds; i++ {
			if numTransactionsLeft%2 != 0 {
				numTransactionsLeft -= 1
				isOdd = true
			}
			lotteryData := make([]*tournament.LotteryData, numTransactionsLeft)
			commitTxnList := make([]*tournament.TransactionReply, numTransactionsLeft)
			//wait := 3
			wait := 0
			comtime := monitor.NewTimeMeasure("commit_time")
			for i := 0; i < numTransactionsLeft; i++ {
				lotteryData[i] = tournament.CreateLotteryData()
				if i == numTransactionsLeft-1 {
					wait = 3
				}
				log.Lvl1("[TournamentLottery] AddCommit called")
				commitTxnList[i], err = byzd.AddCommitTransaction(lotteryData[i], wait)
				if err != nil {
					log.Errorf("AddCommitTransaction failed: %v", err)
					return err
				}
			}
			comtime.Record()

			commitProofList := make([]byzcoin.Proof, numTransactionsLeft)
			wrproof := monitor.NewTimeMeasure("write_proof")
			for i := 0; i < numTransactionsLeft; i++ {
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
			wrproof.Record()
			wait = 0
			secretTxnList := make([]*tournament.TransactionReply, numTransactionsLeft)
			trt := monitor.NewTimeMeasure("tournament_reveal")
			for i := 0; i < numTransactionsLeft; i++ {
				if i == numTransactionsLeft-1 {
					wait = 3
				}
				log.Lvl1("[TournametLottery] AddSecret called")
				secretTxnList[i], err = byzd.AddSecretTransaction(lotteryData[i], wait)
				if err != nil {
					log.Errorf("AddSecretTransaction failed: %v", err)
					return err
				}
			}
			trt.Record()

			secretProofList := make([]byzcoin.Proof, numTransactionsLeft)
			tspt := monitor.NewTimeMeasure("tournament_proof")
			for i := 0; i < numTransactionsLeft; i++ {
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
			tspt.Record()

			trvt := monitor.NewTimeMeasure("tournament_get_winner")
			revealedCommitList := make([]tournament.DataStore, numTransactionsLeft)
			revealedSecretList := make([]tournament.DataStore, numTransactionsLeft)
			for i := 0; i < numTransactionsLeft; i++ {
				err = commitProofList[i].ContractValue(cothority.Suite, tournament.ContractLotteryStoreID, &revealedCommitList[i])
				if err != nil {
					log.Errorf("did not get a commit instance" + err.Error())
					return errors.New("did not get a commit instance" + err.Error())
				}
				err = secretProofList[i].ContractValue(cothority.Suite, tournament.ContractLotteryStoreID, &revealedSecretList[i])
				if err != nil {
					log.Errorf("did not get a secret instance" + err.Error())
					return errors.New("did not get a secret instance" + err.Error())
				}
			}

			var winnerList []int
			for i := 0; i < numTransactionsLeft; {
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
						tournament.SafeXORBytes(result, leftSecret[:], rightSecret[:])
						lastDigit := int(result[31]) % 2
						if lastDigit == 0 {
							winnerList = append(winnerList, i)
							//fmt.Println("Winner is", i)
						} else {
							winnerList = append(winnerList, i+1)
							//fmt.Println("Winner is", i+1)
						}
					}
				}
				i += 2
			}
			if isOdd {
				winnerList = append(winnerList, numTransactionsLeft)
				numTransactionsLeft += 1
			}
			numTransactionsLeft = int(math.Ceil(float64(numTransactionsLeft) / 2))
			isOdd = false
			trvt.Record()
			tournament.OrganizeList(participantList, winnerList)
			//for i := 0; i < numTransactions; i++ {
			//fmt.Print(participantList[i], " ")
			//}
		}
		//lt.Record()
		for i := 0; i < numTransactions; i++ {
			if participantList[i] == 1 {
				fmt.Println("Winner is", i)
				break
			}
		}
	}
	return nil
}
