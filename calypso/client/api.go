package main

import (
	"crypto/sha256"
	//"encoding/hex"
	calypso "github.com/ceyhunalp/centralized_calypso/calypso/service"
	"github.com/dedis/kyber"
	//"github.com/dedis/kyber/sign/schnorr"
	"github.com/dedis/onet"
)

func createWriteTxn(roster *onet.Roster, data []byte, k kyber.Point, c kyber.Point, reader kyber.Point) (string, error) {
	cl := calypso.NewClient()
	defer cl.Close()
	digest := sha256.Sum256(data)

	wr := calypso.WriteRequest{
		EncData:  data,
		DataHash: digest[:],
		K:        k,
		C:        c,
		Reader:   reader,
	}
	reply, err := cl.Write(roster, &wr)
	if err != nil {
		return "", err
	}
	return reply.WriteID, err
}

//func createReadTxn(roster *onet.Roster, suite schnorr.Suite, wID string, sk kyber.Scalar) error {
//cl := calypso.NewClient()
//defer cl.Close()
//widBytes, err := hex.DecodeString(wID)
//if err != nil {
//return err
//}
//sig, err := schnorr.Sign(suite, sk, widBytes)
//if err != nil {
//return err
//}
//rr := calypso.ReadRequest{
//WriteID: wID,
//Sig:     sig,
//}
//_, err = cl.Read(roster, &rr)
//return nil
//}
