package semicentralized

import (
	"github.com/dedis/cothority/byzcoin"
	"github.com/dedis/cothority/calypso"
	"github.com/dedis/cothority/darc"
	"github.com/dedis/cothority/darc/expression"
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

func SetupDarcs() (darc.Signer, darc.Signer, *darc.Darc, error) {
	var writer darc.Signer
	var reader darc.Signer
	writer = darc.NewSignerEd25519(nil, nil)
	reader = darc.NewSignerEd25519(nil, nil)
	writeDarc := darc.NewDarc(darc.InitRules([]darc.Identity{writer.Identity()}, []darc.Identity{writer.Identity()}), []byte("Writer"))
	err := writeDarc.Rules.AddRule(darc.Action("spawn:"+calypso.ContractSemiWriteID), expression.InitOrExpr(writer.Identity().String()))
	if err != nil {
		return writer, reader, nil, err
	}
	err = writeDarc.Rules.AddRule(darc.Action("spawn:"+calypso.ContractReadID), expression.InitOrExpr(reader.Identity().String()))
	if err != nil {
		return writer, reader, nil, err
	}
	return writer, reader, writeDarc, nil
}
