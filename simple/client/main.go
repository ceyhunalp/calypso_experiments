package main

import (
	"crypto/sha256"
	"errors"
	"flag"
	"fmt"
	"github.com/ceyhunalp/centralized_calypso/util"
	"github.com/dedis/cothority/calypso"
	"github.com/dedis/cothority/darc"
	"github.com/dedis/cothority/darc/expression"
	"github.com/dedis/kyber"
	"github.com/dedis/kyber/group/edwards25519"
	"github.com/dedis/kyber/util/random"
	"github.com/dedis/onet"
	"github.com/dedis/onet/log"
	"os"
	"time"
)

func createWriteData(data []byte, reader kyber.Point, serverKey kyber.Point) (*WriteData, error) {
	//func createWriteTransaction(data []byte, reader kyber.Point, serverKey kyber.Point) (*calypso.SimpleWrite, error) {
	var symKey [16]byte
	suite := edwards25519.NewBlakeSHA256Ed25519()
	random.Bytes(symKey[:], random.New())
	encData, err := util.SymEncrypt(data, symKey[:])
	if err != nil {
		return nil, err
	}
	//k, c, _ := util.ElGamalEncrypt(serverKey, symKey[:])
	k, c, _ := util.ElGamalEncrypt(suite, serverKey, symKey[:])
	if err != nil {
		return nil, err
	}
	digest := sha256.Sum256(encData)
	wd := &WriteData{
		Data:     encData,
		DataHash: digest[:],
		K:        k,
		C:        c,
		Reader:   reader,
	}
	return wd, nil
}

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

func runSimpleCalypso(r *onet.Roster, serverKey kyber.Point) error {
	//storageNodeAddr := r.List[0]
	byzd, err := SetupByzcoin(r)
	if err != nil {
		return err
	}
	fmt.Println("Setup byzcoin done")
	writer, reader, wDarc, err := setupDarcs()
	if err != nil {
		return err
	}
	fmt.Println("Setup darcs done")
	_, err = byzd.SpawnDarc(*wDarc, 0)
	if err != nil {
		return err
	}
	fmt.Println("Spawn darcs done")
	data := []byte("On Wisconsin!")
	wd, err := createWriteData(data, reader.Ed25519.Point, serverKey)
	if err != nil {
		return err
	}
	err = StoreEncryptedData(r, wd)
	//err = StoreEncryptedData(storageNodeAddr, wd)
	//storedKey, err := StoreEncryptedData(r, wd.Data, wd.DataHash)
	if err != nil {
		return err
	}
	writeTxn, err := byzd.AddWriteTransaction(wd, writer, *wDarc, 5)
	if err != nil {
		return err
	}
	fmt.Println("Write transaction InstanceID:", writeTxn.InstanceID)
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
	fmt.Println("Read transaction InstanceID:", readTxn.InstanceID)
	rProof, err := byzd.WaitProof(readTxn.InstanceID, time.Second, nil)
	if err != nil {
		return err
	}
	if !rProof.InclusionProof.Match() {
		return errors.New("Read inclusion proof does not match")
	}

	suite := edwards25519.NewBlakeSHA256Ed25519()
	dr, err := byzd.DecryptRequest(r, suite, wrProof, rProof, wd.StoredKey, reader.Ed25519.Secret)
	//dr, err := byzd.DecryptRequest(r, wrProof, rProof, wd.StoredKey, reader.Ed25519.Secret)
	//dr, err := byzd.DecryptRequest(storageNodeAddr, wrProof, rProof, wd.StoredKey, reader.Ed25519.Secret)
	if err != nil {
		return err
	}
	recvData, err := util.RecoverData(dr.Data, suite, reader.Ed25519.Secret, dr.K, dr.C)
	//recvData, err := util.RecoverData(dr.Data, reader.Ed25519.Secret, dr.K, dr.C)
	if err != nil {
		return err
	}
	fmt.Println("Recovered data is:", string(recvData[:]))
	return nil
}

func setup() (*onet.Roster, kyber.Point, error) {
	pkPtr := flag.String("p", "", "pk.txt file")
	dbgPtr := flag.Int("d", 0, "debug level")
	filePtr := flag.String("r", "", "roster.toml file")
	flag.Parse()
	log.SetDebugVisible(*dbgPtr)

	roster, err := util.ReadRoster(*filePtr)
	if err != nil {
		return nil, nil, err
	}
	suite := edwards25519.NewBlakeSHA256Ed25519()
	//serverKey, err := util.GetServerKey(pkPtr)
	serverKey, err := util.GetServerKey(pkPtr, suite)
	if err != nil {
		return nil, nil, err
	}
	return roster, serverKey, nil
}

func main() {
	roster, serverKey, err := setup()
	fmt.Println("SETUP SUCCESS")
	if err != nil {
		log.Errorf("Setup failed: %v", err)
		os.Exit(1)
	}
	err = runSimpleCalypso(roster, serverKey)
	if err != nil {
		log.Errorf("Run SimpleCalypso failed: %v", err)
		os.Exit(1)
	}
}
