package main

import (
	calypso "github.com/ceyhunalp/centralized_calypso/calypso/service"
	"github.com/dedis/kyber"
	"github.com/dedis/onet"
)

func createWriteTxn(roster *onet.Roster, data []byte, k kyber.Point, c kyber.Point, reader kyber.Point) error {
	cl := calypso.NewClient()
	defer cl.Close()
	wr := calypso.WriteRequest{
		EncData: data,
		K:       k,
		C:       c,
		Reader:  reader,
	}
	cl.WriteTxn(roster, &wr)
	return nil
}
