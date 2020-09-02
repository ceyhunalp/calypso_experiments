package main

import (
	"errors"
	"flag"
	"fmt"
	"os"

	sc "github.com/ceyhunalp/calypso_experiments/semi_centralized"
	"github.com/ceyhunalp/calypso_experiments/util"
	"github.com/dedis/kyber"
	"github.com/dedis/onet"
	"github.com/dedis/onet/log"
)

func runSemiCentralized(r *onet.Roster, serverKey kyber.Point, interval int) error {
	byzCl, admin, gDarc, err := sc.SetupByzcoin(r, interval)
	if err != nil {
		return err
	}
	scCl := sc.NewClient(byzCl)
	writer, reader, wDarc, err := scCl.SetupDarcs()
	if err != nil {
		return err
	}
	_, err = scCl.SpawnDarc(admin, *wDarc, gDarc, 0)
	if err != nil {
		return err
	}
	data := []byte("On Wisconsin!")
	wd, err := util.CreateWriteData(data, reader.Ed25519.Point, serverKey, true)
	if err != nil {
		return err
	}
	reply, err := scCl.StoreData(r, wd.Data, wd.DataHash)
	if err != nil {
		return err
	}
	wd.StoredKey = reply.StoredKey

	writeTxn, err := scCl.AddWriteTransaction(wd, writer, *wDarc, 5)
	if err != nil {
		return err
	}
	wrProofResponse, err := scCl.GetProof(writeTxn.InstanceID)
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
	readTxn, err := scCl.AddReadTransaction(&wrProof, reader, *wDarc, 5)
	if err != nil {
		return err
	}
	rProofResponse, err := scCl.GetProof(readTxn.InstanceID)
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

	dr, err := scCl.Decrypt(r, &wrProof, &rProof, wd.StoredKey, reader.Ed25519.Secret)
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
	intervalPtr := flag.Int("i", 10, "block interval value")
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
	err = runSemiCentralized(roster, serverKey, *intervalPtr)
	if err != nil {
		log.Errorf("Run SemiCentralized failed: %v", err)
	}
}
