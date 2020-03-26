package semicentralized

import (
	"encoding/hex"
	sc "github.com/ceyhunalp/calypso_experiments/semi_centralized/service"
	"github.com/ceyhunalp/calypso_experiments/util"
	"go.dedis.ch/cothority"
	"go.dedis.ch/cothority/byzcoin"
	"go.dedis.ch/cothority/calypso"
	"go.dedis.ch/cothority/darc"
	"go.dedis.ch/kyber"
	"go.dedis.ch/kyber/sign/schnorr"
	"go.dedis.ch/onet"
	"go.dedis.ch/onet/log"
	"go.dedis.ch/protobuf"
	"time"
)

type TransactionReply struct {
	*byzcoin.AddTxResponse
	byzcoin.InstanceID
}

func (byzd *ByzcoinData) DecryptRequest(r *onet.Roster, wrProof *byzcoin.Proof, rProof *byzcoin.Proof, key string, sk kyber.Scalar) (*sc.DecryptReply, error) {
	cl := sc.NewClient()
	defer cl.Close()
	keyBytes, err := hex.DecodeString(key)
	if err != nil {
		log.Errorf("DecryptRequest error: %v", err)
		return nil, err
	}
	sig, err := schnorr.Sign(cothority.Suite, sk, keyBytes)
	if err != nil {
		log.Errorf("DecryptRequest error: %v", err)
		return nil, err
	}
	dr := &sc.DecryptRequest{
		Write: wrProof,
		Read:  rProof,
		SCID:  byzd.Cl.ID,
		Key:   key,
		Sig:   sig,
	}
	return cl.Decrypt(r, dr)
}

func (byzd *ByzcoinData) GetProof(id byzcoin.InstanceID) (*byzcoin.GetProofResponse, error) {
	return byzd.Cl.GetProof(id.Slice())
}

func (byzd *ByzcoinData) AddReadTransaction(proof *byzcoin.Proof, signer darc.Signer, darc darc.Darc, wait int) (*TransactionReply, error) {
	read := &calypso.Read{
		Write: byzcoin.NewInstanceID(proof.InclusionProof.Key),
		Xc:    signer.Ed25519.Point,
	}
	readBuf, err := protobuf.Encode(read)
	if err != nil {
		log.Errorf("AddReadTransaction error: %v", err)
		return nil, err
	}
	ctx := byzcoin.ClientTransaction{
		Instructions: byzcoin.Instructions{{
			InstanceID: byzcoin.NewInstanceID(proof.InclusionProof.Key),
			Nonce:      byzcoin.Nonce{},
			Index:      0,
			Length:     1,
			Spawn: &byzcoin.Spawn{
				ContractID: calypso.ContractReadID,
				Args:       byzcoin.Arguments{{Name: "read", Value: readBuf}},
			},
		}},
	}
	err = ctx.Instructions[0].SignBy(darc.GetID(), signer)
	if err != nil {
		log.Errorf("AddReadTransaction error: %v", err)
		return nil, err
	}
	reply := &TransactionReply{}
	reply.InstanceID = ctx.Instructions[0].DeriveID("")
	if wait == 0 {
		reply.AddTxResponse, err = byzd.Cl.AddTransaction(ctx)
	} else {
		reply.AddTxResponse, err = byzd.Cl.AddTransactionAndWait(ctx, wait)
	}
	if err != nil {
		log.Errorf("AddReadTransaction error: %v", err)
		return nil, err
	}
	return reply, nil
}

func (byzd *ByzcoinData) AddWriteTransaction(wd *util.WriteData, signer darc.Signer, darc darc.Darc, wait int) (*TransactionReply, error) {
	sWrite := &calypso.SemiWrite{
		DataHash:  wd.DataHash,
		K:         wd.K,
		C:         wd.C,
		Reader:    wd.Reader,
		EncReader: wd.EncReader,
	}
	writeBuf, err := protobuf.Encode(sWrite)
	if err != nil {
		log.Errorf("AddWriteTransaction error: %v", err)
		return nil, err
	}
	ctx := byzcoin.ClientTransaction{
		Instructions: byzcoin.Instructions{{
			InstanceID: byzcoin.NewInstanceID(darc.GetBaseID()),
			Nonce:      byzcoin.Nonce{},
			Index:      0,
			Length:     1,
			Spawn: &byzcoin.Spawn{
				ContractID: calypso.ContractSemiWriteID,
				Args: byzcoin.Arguments{{
					Name: "write", Value: writeBuf}},
			},
		}},
	}
	//Sign the transaction
	err = ctx.Instructions[0].SignBy(darc.GetID(), signer)
	if err != nil {
		log.Errorf("AddWriteTransaction error: %v", err)
		return nil, err
	}
	reply := &TransactionReply{}
	reply.InstanceID = ctx.Instructions[0].DeriveID("")
	//Delegate the work to the byzcoin client
	//reply.AddTxResponse, err = byzd.Cl.AddTransaction(ctx)
	if wait == 0 {
		reply.AddTxResponse, err = byzd.Cl.AddTransaction(ctx)
	} else {
		reply.AddTxResponse, err = byzd.Cl.AddTransactionAndWait(ctx, wait)
	}
	if err != nil {
		log.Errorf("AddWriteTransaction error: %v", err)
		return nil, err
	}
	return reply, err
}

func (byzd *ByzcoinData) SpawnDarc(spawnDarc darc.Darc, wait int) (*byzcoin.AddTxResponse, error) {
	darcBuf, err := spawnDarc.ToProto()
	if err != nil {
		log.Errorf("SpawnDarc error: %v", err)
		return nil, err
	}
	ctx := byzcoin.ClientTransaction{
		Instructions: []byzcoin.Instruction{{
			InstanceID: byzcoin.NewInstanceID(byzd.GDarc.GetBaseID()),
			Nonce:      byzcoin.GenNonce(),
			Index:      0,
			Length:     1,
			Spawn: &byzcoin.Spawn{
				ContractID: byzcoin.ContractDarcID,
				Args: []byzcoin.Argument{{
					Name:  "darc",
					Value: darcBuf,
				}},
			},
		}},
	}
	err = ctx.Instructions[0].SignBy(byzd.GDarc.GetBaseID(), byzd.Signer)
	if err != nil {
		log.Errorf("SpawnDarc error: %v", err)
		return nil, err
	}
	return byzd.Cl.AddTransactionAndWait(ctx, wait)
}

func StoreEncryptedData(r *onet.Roster, wd *util.WriteData) error {
	cl := sc.NewClient()
	defer cl.Close()
	sr := sc.StoreRequest{
		Data:     wd.Data,
		DataHash: wd.DataHash,
	}
	reply, err := cl.StoreData(r, &sr)
	if err != nil {
		log.Errorf("StoreEncryptedData error: %v", err)
		return err
	}
	wd.StoredKey = reply.StoredKey
	return nil
}

func SetupByzcoin(r *onet.Roster, blockInterval int) (*ByzcoinData, error) {
	var err error
	byzd := &ByzcoinData{}
	byzd.Signer = darc.NewSignerEd25519(nil, nil)
	byzd.GMsg, err = byzcoin.DefaultGenesisMsg(byzcoin.CurrentVersion, r, []string{"spawn:" + byzcoin.ContractDarcID, "spawn:" + calypso.ContractSemiWriteID, "spawn:" + calypso.ContractReadID}, byzd.Signer.Identity())
	if err != nil {
		log.Errorf("SetupByzcoin error: %v", err)
		return nil, err
	}
	//byzd.GMsg.BlockInterval = 10 * time.Second
	byzd.GMsg.BlockInterval = time.Duration(blockInterval) * time.Second
	log.Info("Block interval is:", byzd.GMsg.BlockInterval)
	byzd.GDarc = &byzd.GMsg.GenesisDarc
	byzd.Cl, _, err = byzcoin.NewLedger(byzd.GMsg, false)
	if err != nil {
		log.Errorf("SetupByzcoin error: %v", err)
		return nil, err
	}
	return byzd, nil
}
