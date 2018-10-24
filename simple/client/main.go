package main

import (
	"errors"
	"flag"
	"fmt"
	"github.com/ceyhunalp/centralized_calypso/util"
	"github.com/dedis/cothority/calypso"
	"github.com/dedis/cothority/darc"
	"github.com/dedis/cothority/darc/expression"
	"github.com/dedis/kyber"
	"github.com/dedis/onet"
	"github.com/dedis/onet/log"
	"os"
	"strconv"
	"strings"
	"time"
)

func setupDarcs() (darc.Signer, darc.Signer, *darc.Darc, error) {
	var writer darc.Signer
	var reader darc.Signer
	writer = darc.NewSignerEd25519(nil, nil)
	reader = darc.NewSignerEd25519(nil, nil)
	writeDarc := darc.NewDarc(darc.InitRules([]darc.Identity{writer.Identity()}, []darc.Identity{writer.Identity()}), []byte("Writer"))
	err := writeDarc.Rules.AddRule(darc.Action("spawn:"+calypso.ContractSimpleWriteID), expression.InitOrExpr(writer.Identity().String()))
	if err != nil {
		return writer, reader, nil, err
	}
	err = writeDarc.Rules.AddRule(darc.Action("spawn:"+calypso.ContractReadID), expression.InitOrExpr(reader.Identity().String()))
	if err != nil {
		return writer, reader, nil, err
	}
	return writer, reader, writeDarc, nil
}

func runSimpleCalypso(r *onet.Roster, serverKey kyber.Point, byzd *ByzcoinData, data []byte) error {
	//data := []byte("On Wisconsin!")
	//byzd, err := SetupByzcoin(r)
	//if err != nil {
	//return err
	//}
	writer, reader, wDarc, err := setupDarcs()
	if err != nil {
		return err
	}
	_, err = byzd.SpawnDarc(*wDarc, 0)
	if err != nil {
		return err
	}

	wd, err := util.CreateWriteData(data, reader.Ed25519.Point, serverKey)
	if err != nil {
		return err
	}
	err = StoreEncryptedData(r, wd)
	if err != nil {
		return err
	}

	writeTxn, err := byzd.AddWriteTransaction(wd, writer, *wDarc, 5)
	if err != nil {
		return err
	}
	//fmt.Println("Write transaction InstanceID:", writeTxn.InstanceID)

	wrProof, err := byzd.WaitProof(writeTxn.InstanceID, time.Second, nil)
	if err != nil {
		return err
	}
	if !wrProof.InclusionProof.Match() {
		return errors.New("Write inclusion proof does not match")
	}
	readTxn, err := byzd.AddReadTransaction(wrProof, reader, *wDarc, 5)
	if err != nil {
		return err
	}
	//fmt.Println("Read transaction InstanceID:", readTxn.InstanceID)

	rProof, err := byzd.WaitProof(readTxn.InstanceID, time.Second, nil)
	if err != nil {
		return err
	}
	if !rProof.InclusionProof.Match() {
		return errors.New("Read inclusion proof does not match")
	}

	dr, err := byzd.DecryptRequest(r, wrProof, rProof, wd.StoredKey, reader.Ed25519.Secret)
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

func getServerKey(pkPtr *string) (kyber.Point, error) {
	return util.GetServerKey(pkPtr)
}

func readRoster(filePtr *string) (*onet.Roster, error) {
	return util.ReadRoster(filePtr)
}

func main() {
	pkPtr := flag.String("p", "", "pk.txt file")
	dbgPtr := flag.Int("d", 0, "debug level")
	filePtr := flag.String("r", "", "roster.toml file")
	flag.Parse()
	log.SetDebugVisible(*dbgPtr)

	roster, err := readRoster(filePtr)
	if err != nil {
		log.Errorf("Reading roster failed: %v", err)
		os.Exit(1)
	}
	serverKey, err := getServerKey(pkPtr)
	if err != nil {
		log.Errorf("Get server key failed: %v", err)
		os.Exit(1)
	}
	byzd, err := SetupByzcoin(roster)
	if err != nil {
		log.Errorf("Setting up Byzcoin failed: %v", err)
		os.Exit(1)
	}
	baseStr := "On Wisconsin! -- "
	for i := 0; i < 70; i++ {
		err = runSimpleCalypso(roster, serverKey, byzd, []byte(strings.Join([]string{baseStr, strconv.Itoa(i + 1)}, "")))
		if err != nil {
			log.Errorf("Run SimpleCalypso failed: %v", err)
		}
	}
}
