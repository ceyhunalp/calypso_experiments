package main

import (
	"errors"
	"flag"
	"fmt"
	"github.com/ceyhunalp/centralized_calypso/simple"
	"github.com/ceyhunalp/centralized_calypso/util"
	"github.com/dedis/kyber"
	"github.com/dedis/onet"
	"github.com/dedis/onet/log"
	"os"
	"strconv"
	"strings"
)

func runSimpleCalypso(r *onet.Roster, serverKey kyber.Point, byzd *simple.ByzcoinData, data []byte) error {
	writer, reader, wDarc, err := simple.SetupDarcs()
	if err != nil {
		return err
	}
	_, err = byzd.SpawnDarc(*wDarc, 0)
	if err != nil {
		return err
	}

	wd, err := util.CreateWriteData(data, reader.Ed25519.Point, serverKey, true)
	if err != nil {
		return err
	}
	err = simple.StoreEncryptedData(r, wd)
	if err != nil {
		return err
	}

	writeTxn, err := byzd.AddWriteTransaction(wd, writer, *wDarc, 5)
	if err != nil {
		return err
	}
	wrProofResponse, err := byzd.GetProof(writeTxn.InstanceID)
	if err != nil {
		return err
	}
	wrProof := wrProofResponse.Proof
	if !wrProof.InclusionProof.Match() {
		return errors.New("Write inclusion proof does not match")
	}
	//wrProof, err := byzd.WaitProof(writeTxn.InstanceID, time.Second, nil)
	//if err != nil {
	//return err
	//}
	//if !wrProof.InclusionProof.Match() {
	//return errors.New("Write inclusion proof does not match")
	//}
	readTxn, err := byzd.AddReadTransaction(&wrProof, reader, *wDarc, 5)
	if err != nil {
		return err
	}
	rProofResponse, err := byzd.GetProof(readTxn.InstanceID)
	if err != nil {
		return err
	}
	rProof := rProofResponse.Proof
	if !rProof.InclusionProof.Match() {
		return errors.New("Read inclusion proof does not match")
	}

	//rProof, err := byzd.WaitProof(readTxn.InstanceID, time.Second, nil)
	//if err != nil {
	//return err
	//}
	//if !rProof.InclusionProof.Match() {
	//return errors.New("Read inclusion proof does not match")
	//}

	dr, err := byzd.DecryptRequest(r, &wrProof, &rProof, wd.StoredKey, reader.Ed25519.Secret)
	if err != nil {
		return err
	}

	recvData, err := util.RecoverData(dr.Data, reader.Ed25519.Secret, dr.K, dr.C)
	if err != nil {
		return err
	}
	fmt.Println("Recovered data is:", string(recvData[:]))
	return nil
}

func main() {
	pkPtr := flag.String("p", "", "pk.txt file")
	dbgPtr := flag.Int("d", 0, "debug level")
	filePtr := flag.String("r", "", "roster.toml file")
	flag.Parse()
	log.SetDebugVisible(*dbgPtr)

	roster, err := util.ReadRoster(filePtr)
	if err != nil {
		log.Errorf("Reading roster failed: %v", err)
		os.Exit(1)
	}
	serverKey, err := util.GetServerKey(pkPtr)
	if err != nil {
		log.Errorf("Get server key failed: %v", err)
		os.Exit(1)
	}
	byzd, err := simple.SetupByzcoin(roster)
	if err != nil {
		log.Errorf("Setting up Byzcoin failed: %v", err)
		os.Exit(1)
	}
	baseStr := "On Wisconsin! -- "
	for i := 0; i < 100; i++ {
		err = runSimpleCalypso(roster, serverKey, byzd, []byte(strings.Join([]string{baseStr, strconv.Itoa(i + 1)}, "")))
		if err != nil {
			log.Errorf("Run SimpleCalypso failed: %v", err)
		}
	}
}
