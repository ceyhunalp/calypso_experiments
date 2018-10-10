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
	//keyStr, err := encoding.PointToStringHex(suite, serverKey[0])
	//fmt.Println("What I converted:", keyStr)
	// Reader keys
	rSk := suite.Scalar().Pick(suite.RandomStream())
	rPk := suite.Point().Mul(rSk, nil)

	createWriteTxn(roster, encData, k, c, rPk)

	//cl := calypso.NewClient()
	//writeTxnData := util.WriteTxnData{
	//EncData: encData,
	//K:       k,
	//C:       c,
	//Reader:  rPk,
	//}
	//cl.WriteTxn(roster, &writeTxnData)

}
