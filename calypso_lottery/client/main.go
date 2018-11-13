package main

import (
	"errors"
	"flag"
	"fmt"
	lottery "github.com/ceyhunalp/centralized_calypso/calypso_lottery"
	"github.com/ceyhunalp/centralized_calypso/util"
	"github.com/dedis/cothority"
	"github.com/dedis/cothority/byzcoin"
	"github.com/dedis/cothority/calypso"
	"github.com/dedis/onet"
	"github.com/dedis/onet/log"
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

func runCalypsoLottery(r *onet.Roster, calypsoClient *calypso.Client, byzd *lottery.ByzcoinData, ltsReply *calypso.CreateLTSReply, numParticipant int) error {

	writerList, reader, writeDarcList, err := lottery.SetupDarcs(numParticipant)
	if err != nil {
		return err
	}
	for i := 0; i < numParticipant; i++ {
		_, err := byzd.SpawnDarc(*writeDarcList[i], 0)
		if err != nil {
			return err
		}
	}

	lotteryData := make([]*lottery.LotteryData, numParticipant)
	writeTxnData := make([]*calypso.Write, numParticipant)
	for i := 0; i < numParticipant; i++ {
		lotteryData[i] = lottery.CreateLotteryData()
		writeTxnData[i] = calypso.NewWrite(cothority.Suite, ltsReply.LTSID, writeDarcList[i].GetBaseID(), ltsReply.X, lotteryData[i].Secret[:])
	}

	for i := 0; i < numParticipant; i++ {
		fmt.Println("Lottery ticket #", i, "is:", lotteryData[i].Secret)
	}

	writeTxnList := make([]*calypso.WriteReply, numParticipant)
	for i := 0; i < numParticipant; i++ {
		writeTxnList[i], err = calypsoClient.AddWrite(writeTxnData[i], writerList[i], *writeDarcList[i], 2)
		if err != nil {
			return err
		}
	}

	writeProofList := make([]byzcoin.Proof, numParticipant)
	for i := 0; i < numParticipant; i++ {
		wrProofResponse, err := byzd.Cl.GetProof(writeTxnList[i].InstanceID.Slice())
		if err != nil {
			return err
		}
		if !wrProofResponse.Proof.InclusionProof.Match() {
			return errors.New("Write inclusion proof does not match")
		}
		writeProofList[i] = wrProofResponse.Proof
	}

	readTxnList := make([]*calypso.ReadReply, numParticipant)
	for i := 0; i < numParticipant; i++ {
		readTxnList[i], err = calypsoClient.AddRead(&writeProofList[i], reader, *writeDarcList[i], 2)
		if err != nil {
			return err
		}
	}

	readProofList := make([]byzcoin.Proof, numParticipant)
	for i := 0; i < numParticipant; i++ {
		rProofResponse, err := byzd.Cl.GetProof(readTxnList[i].InstanceID.Slice())
		if err != nil {
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
			return err
		}
		if !dk.X.Equal(ltsReply.X) {
			return errors.New("Points not same")
		}

		decodedSecretList[i], err = calypso.DecodeKey(cothority.Suite, ltsReply.X, dk.Cs, dk.XhatEnc, reader.Ed25519.Secret)
		if err != nil {
			return err
		}
	}

	result := make([]byte, 32)
	for i := 0; i < numParticipant; i++ {
		safeXORBytes(result, result, decodedSecretList[i])
	}

	fmt.Println(result)
	lastDigit := int(result[31])
	fmt.Println(lastDigit)
	fmt.Println(lastDigit % numParticipant)

	return nil
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
	byzd, err := lottery.SetupByzcoin(roster)
	if err != nil {
		log.Errorf("Setting up Byzcoin failed: %v", err)
		os.Exit(1)
	}

	calypsoClient := calypso.NewClient(byzd.Cl)
	ltsReply, err := calypsoClient.CreateLTS()
	if err != nil {
		os.Exit(1)
	}

	err = runCalypsoLottery(roster, calypsoClient, byzd, ltsReply, *numParticipant)
}
