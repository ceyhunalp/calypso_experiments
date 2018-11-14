package main

import (
	"bytes"
	"crypto/sha256"
	"errors"
	"flag"
	"fmt"
	"github.com/ceyhunalp/centralized_calypso/util"
	zerolottery "github.com/ceyhunalp/centralized_calypso/zero_lottery"
	"github.com/dedis/cothority"
	"github.com/dedis/cothority/byzcoin"
	"github.com/dedis/onet"
	"github.com/dedis/onet/log"
	"math"
	"os"
)

func safeXORBytes(dst, a, b []byte) int {
	n := len(a)
	if len(b) < n {
		n = len(b)
	}
	for i := 0; i < n; i++ {
		dst[i] = a[i] ^ b[i]
	}
	return n
}

func runZeroLottery(r *onet.Roster, byzd *zerolottery.ByzcoinData, numParticipant int) error {
	var err error
	numRounds := int(math.Log2(float64(numParticipant)))
	numParticipantLeft := numParticipant
	//writeTxnData := make([]*calypso.Write, numParticipant)
	participantList := make([]int, numParticipant)
	for i := 0; i < numParticipant; i++ {
		participantList[i] = 1
	}

	for i := 0; i < numRounds; i++ {
		lotteryData := make([]*zerolottery.LotteryData, numParticipantLeft)
		commitTxnList := make([]*zerolottery.TransactionReply, numParticipantLeft)
		for i := 0; i < numParticipantLeft; i++ {
			lotteryData[i] = zerolottery.CreateLotteryData()
			commitTxnList[i], err = byzd.AddCommitTransaction(lotteryData[i], 2)
			if err != nil {
				return err
			}
			fmt.Println("Secret is:", lotteryData[i].Secret)
			//writeTxnData[i] = calypso.NewWrite(cothority.Suite, ltsReply.LTSID, writeDarcList[i].GetBaseID(), ltsReply.X, lotteryData[i].Secret[:])
		}

		commitProofList := make([]byzcoin.Proof, numParticipantLeft)
		for i := 0; i < numParticipantLeft; i++ {
			commitProofResp, err := byzd.Cl.GetProof(commitTxnList[i].InstanceID.Slice())
			if err != nil {
				return err
			}
			if !commitProofResp.Proof.InclusionProof.Match() {
				return errors.New("Commit inclusion proof does not match")
			}
			commitProofList[i] = commitProofResp.Proof
		}

		revealedCommitList := make([]zerolottery.Commit, numParticipantLeft)
		for i := 0; i < numParticipantLeft; i++ {
			err = commitProofList[i].ContractValue(cothority.Suite, zerolottery.ContractCommitID, &revealedCommitList[i])
			if err != nil {
				log.Errorf("did not get a commit instance" + err.Error())
				return errors.New("did not get a commit instance" + err.Error())
			}
		}
		//for i := 0; i < numParticipantLeft; i++ {
		//fmt.Println("Secret hash is:", revealedCommitList[i].SecretHash)
		//}

		var winnerList []int
		for i := 0; i < numParticipantLeft; {
			//These are the hashes
			leftSecret := lotteryData[i].Secret
			rightSecret := lotteryData[i+1].Secret
			leftDigest := sha256.Sum256(leftSecret[:])
			rightDigest := sha256.Sum256(rightSecret[:])
			if bytes.Compare(leftDigest[:], revealedCommitList[i].SecretHash[:]) != 0 {
				fmt.Println("Digests do not match - winner is", i+1)
				winnerList = append(winnerList, 2)
			} else {
				if bytes.Compare(rightDigest[:], revealedCommitList[i+1].SecretHash[:]) != 0 {
					fmt.Println("Digests do not match - winner is", i)
					winnerList = append(winnerList, 1)
				} else {
					result := make([]byte, 32)
					safeXORBytes(result, leftSecret[:], rightSecret[:])
					lastDigit := int(result[31]) % 2
					fmt.Println("Last digit is", int(result[31]))
					if lastDigit == 0 {
						//fmt.Println("Winner is", i)
						winnerList = append(winnerList, 1)
					} else {
						//fmt.Println("Winner is", i+1)
						winnerList = append(winnerList, 2)
					}
				}
			}
			i += 2
		}

		//for i := 0; i < numParticipantLeft/2; i++ {
		//fmt.Print(winnerList[i], " ")
		//}
		//fmt.Println()
		organizeList(participantList, winnerList)
		numParticipantLeft = numParticipantLeft / 2
	}
	for i := 0; i < numParticipant; i++ {
		if participantList[i] == 1 {
			fmt.Println("Winner is", i)
			break
		}
	}
	return nil
}

func organizeList(participantList []int, winnerList []int) {
	ctr := 0
	idx := 0
	seen := 0
	clean := false
	sz := len(winnerList)
	numPart := len(participantList)
	for ctr < sz {
		win := winnerList[ctr]
		for idx < numPart {
			val := participantList[idx]
			if val == 1 {
				seen++
				if seen > ctr {
					if clean {
						participantList[idx] = 0
						clean = false
						idx = numPart
					} else {
						if win == 1 {
							//Keep this next one is eliminated
							clean = true
						} else {
							participantList[idx] = 0
							idx = numPart
						}
					}
				}
			}
			idx++
		}
		ctr++
		seen = 0
		idx = 0
	}
}

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
	byzd, err := zerolottery.SetupByzcoin(roster)
	if err != nil {
		log.Errorf("Setting up Byzcoin failed: %v", err)
		os.Exit(1)
	}

	err = runZeroLottery(roster, byzd, *numParticipant)
}
