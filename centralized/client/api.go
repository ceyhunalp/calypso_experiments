package main

import (
	"crypto/sha256"
	"encoding/hex"
	centralized "github.com/ceyhunalp/centralized_calypso/centralized/service"
	"github.com/dedis/kyber"
	"github.com/dedis/kyber/sign/schnorr"
	"github.com/dedis/onet"
)

//func CreateWriteTxn(dest *network.ServerIdentity, data []byte, k kyber.Point, c kyber.Point, reader kyber.Point) (string, error) {
func CreateWriteTxn(roster *onet.Roster, data []byte, k kyber.Point, c kyber.Point, reader kyber.Point) (string, error) {
	cl := centralized.NewClient()
	defer cl.Close()
	dataHash := sha256.Sum256(data)

	wr := centralized.WriteRequest{
		EncData:  data,
		DataHash: dataHash[:],
		K:        k,
		C:        c,
		Reader:   reader,
	}
	//reply, err := cl.Write(dest, &wr)
	reply, err := cl.Write(roster, &wr)
	if err != nil {
		return "", err
	}
	return reply.WriteID, err
}

//func CreateReadTxn(dest *network.ServerIdentity, wID string, sk kyber.Scalar) (kyber.Point, kyber.Point, error) {
//func CreateReadTxn(roster *onet.Roster, wID string, sk kyber.Scalar) (kyber.Point, kyber.Point, error) {
func CreateReadTxn(roster *onet.Roster, suite schnorr.Suite, wID string, sk kyber.Scalar) (kyber.Point, kyber.Point, error) {
	cl := centralized.NewClient()
	defer cl.Close()
	widBytes, err := hex.DecodeString(wID)
	if err != nil {
		return nil, nil, err
	}
	//sig, err := schnorr.Sign(cothority.Suite, sk, widBytes)
	sig, err := schnorr.Sign(suite, sk, widBytes)
	if err != nil {
		return nil, nil, err
	}
	rr := centralized.ReadRequest{
		WriteID: wID,
		Sig:     sig,
	}
	//reply, err := cl.Read(dest, &rr)
	reply, err := cl.Read(roster, &rr)
	if err != nil {
		return nil, nil, err
	}
	return reply.K, reply.C, nil
}
