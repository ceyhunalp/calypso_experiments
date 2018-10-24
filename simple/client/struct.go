package main

import (
	"github.com/dedis/cothority/byzcoin"
	"github.com/dedis/cothority/darc"
	//"github.com/dedis/kyber"
	"github.com/dedis/onet"
)

type ByzcoinData struct {
	Signer darc.Signer
	Roster *onet.Roster
	Cl     *byzcoin.Client
	GMsg   *byzcoin.CreateGenesisBlock
	GDarc  *darc.Darc
	Csr    *byzcoin.CreateGenesisBlockResponse
}
