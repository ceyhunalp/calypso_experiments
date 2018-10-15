package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/ceyhunalp/centralized_calypso/calypso/util"
	"github.com/dedis/kyber/group/edwards25519"
	"github.com/dedis/kyber/util/random"
	"github.com/dedis/onet/log"
)

func main() {
	pkPtr := flag.String("p", "", "pk.txt file")
	dbgPtr := flag.Int("d", 0, "debug level")
	filePtr := flag.String("r", "", "roster.toml file")
	flag.Parse()
	log.SetDebugVisible(*dbgPtr)

	roster, err := util.ReadRoster(*filePtr)
	if err != nil {
		log.Errorf("Couldn't read roster.toml: %v", err)
		os.Exit(1)
	}
	fmt.Println(roster.List[0].Address)

	msg := []byte("On Wisconsin!")
	suite := edwards25519.NewBlakeSHA256Ed25519()
	var symKey [16]byte
	random.Bytes(symKey[:], random.New())
	encData, err := util.SymEncrypt(msg, symKey[:])
	if err != nil {
		log.Errorf("Symmetric encryption failed: %v", err)
		os.Exit(1)
	}

	serverKey, err := util.GetServerKey(pkPtr, suite)
	if err != nil {
		log.Errorf("Could not retrieve server key: %v", err)
		os.Exit(1)
	}
	k, c, _ := util.ElGamalEncrypt(suite, serverKey, symKey[:])
	if err != nil {
		fmt.Println("Erroring out getting server key")
		os.Exit(1)
	}

	// Reader keys
	rSk := suite.Scalar().Pick(suite.RandomStream())
	rPk := suite.Point().Mul(rSk, nil)

	// Create write transaction
	wID, err := CreateWriteTxn(roster, encData, k, c, rPk)
	if err != nil {
		log.Errorf("Write transaction failed: %v", err)
		os.Exit(1)
	}
	fmt.Println("Write transaction success:", wID)

	// Create read transaction
	kRead, cRead, err := CreateReadTxn(roster, suite, wID, rSk)
	if err != nil {
		log.Errorf("Read transaction failed: %v", err)
		os.Exit(1)
	}

	recvData, err := util.RecoverData(encData, suite, rSk, kRead, cRead)
	if err != nil {
		log.Errorf("Cannot recover data: %v", err)
		os.Exit(1)
	}
	fmt.Println(string(recvData[:]))

	// Try to create duplicate write transaction
	_, err = CreateWriteTxn(roster, encData, k, c, rPk)
	if err != nil {
		log.Errorf("Write transaction failed: %v", err)
		//os.Exit(1)
	}

	// Create unauthorized reader
	newSk := suite.Scalar().Pick(suite.RandomStream())
	_ = suite.Point().Mul(newSk, nil)

	// Create read transaction with unauthorized reader
	_, _, err = CreateReadTxn(roster, suite, wID, newSk)
	if err != nil {
		log.Errorf("Read transaction failed: %v", err)
		os.Exit(1)
	}
}
