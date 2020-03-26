package main

import (
	"bytes"
	"crypto/sha256"
	"errors"
	"flag"
	"fmt"
	tournament "github.com/ceyhunalp/calypso_experiments/tournament_lottery"
	"github.com/ceyhunalp/calypso_experiments/util"
	"go.dedis.ch/cothority"
	"go.dedis.ch/cothority/byzcoin"
	"go.dedis.ch/onet"
	"go.dedis.ch/onet/log"
	"math"
	"os"
)

func runTournamentLottery(r *onet.Roster, byzd *tournament.ByzcoinData, numParticipant int) error {
	var err error
	numRounds := int(math.Log2(float64(numParticipant)))
	numParticipantLeft := numParticipant
	//writeTxnData := make([]*calypso.Write, numParticipant)
	participantList := make([]int, numParticipant)
	for i := 0; i < numParticipant; i++ {
		participantList[i] = 1
	}

	for i := 0; i < numRounds; i++ {
		lotteryData := make([]*tournament.LotteryData, numParticipantLeft)
		commitTxnList := make([]*tournament.TransactionReply, numParticipantLeft)
		for i := 0; i < numParticipantLeft; i++ {
			lotteryData[i] = tournament.CreateLotteryData()
			wait := 0
			if i == numParticipantLeft-1 {
				wait = 3
			}
			commitTxnList[i], err = byzd.AddCommitTransaction(lotteryData[i], wait)
			if err != nil {
				log.Errorf("AddCommitTransaction failed: %v", err)
				return err
			}
			fmt.Println("Commit is:", lotteryData[i].Digest)
			//writeTxnData[i] = calypso.NewWrite(cothority.Suite, ltsReply.LTSID, writeDarcList[i].GetBaseID(), ltsReply.X, lotteryData[i].Secret[:])
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

		secretTxnList := make([]*tournament.TransactionReply, numParticipantLeft)
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
			fmt.Println("Secret is:", lotteryData[i].Secret)
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

		revealedCommitList := make([]tournament.DataStore, numParticipantLeft)
		revealedSecretList := make([]tournament.DataStore, numParticipantLeft)
		//revealedCommitList := make([]tournament.Commit, numParticipantLeft)
		for i := 0; i < numParticipantLeft; i++ {
			err = commitProofList[i].ContractValue(cothority.Suite, tournament.ContractLotteryStoreID, &revealedCommitList[i])
			if err != nil {
				log.Errorf("did not get a commit instance" + err.Error())
				return errors.New("did not get a commit instance" + err.Error())
			}
			//fmt.Println("Revealed commit:", revealedCommitList[i])
			err = secretProofList[i].ContractValue(cothority.Suite, tournament.ContractLotteryStoreID, &revealedSecretList[i])
			if err != nil {
				log.Errorf("did not get a secret instance" + err.Error())
				return errors.New("did not get a secret instance" + err.Error())
			}
			//fmt.Println("Revealed secret:", revealedSecretList[i])
		}

		var winnerList []int
		for i := 0; i < numParticipantLeft; {
			//These are the hashes
			//leftSecret := lotteryData[i].Secret
			//rightSecret := lotteryData[i+1].Secret
			leftSecret := revealedSecretList[i].Data
			rightSecret := revealedSecretList[i+1].Data
			leftDigest := sha256.Sum256(leftSecret[:])
			rightDigest := sha256.Sum256(rightSecret[:])
			//fmt.Println("Left secret:", leftSecret)
			//fmt.Println("Left commit:", leftDigest)
			//fmt.Println("Right secret:", rightSecret)
			//fmt.Println("Right commit:", rightDigest)
			//if bytes.Compare(leftDigest[:], revealedCommitList[i].SecretHash[:]) != 0 {
			if bytes.Compare(leftDigest[:], revealedCommitList[i].Data[:]) != 0 {
				fmt.Println("Digests do not match - winner is", i+1)
				winnerList = append(winnerList, i+1)
				//winnerList = append(winnerList, 2)
			} else {
				//if bytes.Compare(rightDigest[:], revealedCommitList[i+1].SecretHash[:]) != 0 {
				if bytes.Compare(rightDigest[:], revealedCommitList[i+1].Data[:]) != 0 {
					fmt.Println("Digests do not match - winner is", i)
					winnerList = append(winnerList, i)
					//winnerList = append(winnerList, 1)
				} else {
					result := make([]byte, 32)
					tournament.SafeXORBytes(result, leftSecret[:], rightSecret[:])
					lastDigit := int(result[31]) % 2
					fmt.Println("Last digit is", int(result[31]))
					if lastDigit == 0 {
						fmt.Println("Winner is", i)
						winnerList = append(winnerList, i)
						//winnerList = append(winnerList, 1)
					} else {
						fmt.Println("Winner is", i+1)
						winnerList = append(winnerList, i+1)
						//winnerList = append(winnerList, 2)
					}
				}
			}
			i += 2
		}
		tournament.OrganizeList(participantList, winnerList)
		numParticipantLeft = numParticipantLeft / 2
		//for i := 0; i < numParticipantLeft/2; i++ {
		//fmt.Print(winnerList[i], " ")
		//}
		//fmt.Println()
	}
	for i := 0; i < numParticipant; i++ {
		if participantList[i] == 1 {
			fmt.Println("Winner is", i)
			break
		}
	}
	return nil
}

//func organizeList(participantList []int, winnerList []int) {
//ctr := 0
//idx := 0
//seen := 0
//clean := false
//sz := len(winnerList)
//numPart := len(participantList)
//for ctr < sz {
//win := winnerList[ctr]
//for idx < numPart {
//val := participantList[idx]
//if val == 1 {
//seen++
//if seen > ctr {
//if clean {
//participantList[idx] = 0
//clean = false
//idx = numPart
//} else {
//if win == 1 {
////Keep this next one is eliminated
//clean = true
//} else {
//participantList[idx] = 0
//idx = numPart
//}
//}
//}
//}
//idx++
//}
//ctr++
//seen = 0
//idx = 0
//}
//}

func main() {
	numParticipant := flag.Int("n", 0, "number of participants")
	dbgPtr := flag.Int("d", 0, "debug level")
	filePtr := flag.String("r", "", "roster.toml file")
	flag.Parse()
	log.SetDebugVisible(*dbgPtr)

	roster, err := util.ReadRoster(filePtr)
	if err != nil {
		log.Errorf("Reading roster failed: %v", err)
		os.Exit(1)
	}
	byzd, err := tournament.SetupByzcoin(roster)
	if err != nil {
		log.Errorf("Setting up Byzcoin failed: %v", err)
		os.Exit(1)
	}

	err = runTournamentLottery(roster, byzd, *numParticipant)
}
