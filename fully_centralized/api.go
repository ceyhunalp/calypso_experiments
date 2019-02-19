package centralized

import (
	"encoding/hex"
	fc "github.com/ceyhunalp/calypso_experiments/fully_centralized/service"
	"github.com/ceyhunalp/calypso_experiments/util"
	"github.com/dedis/cothority"
	"github.com/dedis/kyber"
	"github.com/dedis/kyber/sign/schnorr"
	"github.com/dedis/onet"
)

func CreateWriteTxn(roster *onet.Roster, wd *util.WriteData) (*util.WriteData, error) {
	cl := fc.NewClient()
	defer cl.Close()
	wr := fc.WriteRequest{
		EncData:  wd.Data,
		DataHash: wd.DataHash,
		K:        wd.K,
		C:        wd.C,
		Reader:   wd.Reader,
		//EncReader: wd.EncReader,
	}
	reply, err := cl.Write(roster, &wr)
	if err != nil {
		wd.StoredKey = ""
	} else {
		wd.StoredKey = reply.WriteID
	}
	return wd, err
}

func CreateReadTxn(roster *onet.Roster, wID string, sk kyber.Scalar) (kyber.Point, kyber.Point, error) {
	cl := fc.NewClient()
	defer cl.Close()
	widBytes, err := hex.DecodeString(wID)
	if err != nil {
		return nil, nil, err
	}
	sig, err := schnorr.Sign(cothority.Suite, sk, widBytes)
	if err != nil {
		return nil, nil, err
	}
	rr := fc.ReadRequest{
		WriteID: wID,
		Sig:     sig,
	}
	reply, err := cl.Read(roster, &rr)
	if err != nil {
		return nil, nil, err
	}
	return reply.K, reply.C, nil
}
