package main

import (
	"encoding/hex"
	simple "github.com/ceyhunalp/centralized_calypso/simple/service"
	//"github.com/dedis/cothority"
	"github.com/dedis/cothority/byzcoin"
	"github.com/dedis/cothority/calypso"
	"github.com/dedis/cothority/darc"
	"github.com/dedis/kyber"
	"github.com/dedis/kyber/sign/schnorr"
	"github.com/dedis/onet"
	//"github.com/dedis/onet/network"
	"github.com/dedis/protobuf"
	"time"
)

type TransactionReply struct {
	*byzcoin.AddTxResponse
	byzcoin.InstanceID
}

//func (byzd *ByzcoinData) DecryptRequest(r *onet.Roster, wrProof *byzcoin.Proof, rProof *byzcoin.Proof, key string, sk kyber.Scalar) (*simple.DecryptReply, error) {
//func (byzd *ByzcoinData) DecryptRequest(dest *network.ServerIdentity, wrProof *byzcoin.Proof, rProof *byzcoin.Proof, key string, sk kyber.Scalar) (*simple.DecryptReply, error) {
func (byzd *ByzcoinData) DecryptRequest(r *onet.Roster, suite schnorr.Suite, wrProof *byzcoin.Proof, rProof *byzcoin.Proof, key string, sk kyber.Scalar) (*simple.DecryptReply, error) {
	cl := simple.NewClient()
	defer cl.Close()
	keyBytes, err := hex.DecodeString(key)
	if err != nil {
		return nil, err
	}
	//sig, err := schnorr.Sign(cothority.Suite, sk, keyBytes)
	sig, err := schnorr.Sign(suite, sk, keyBytes)
	if err != nil {
		return nil, err
	}
	dr := &simple.DecryptRequest{
		Write: wrProof,
		Read:  rProof,
		SCID:  byzd.Cl.ID,
		Key:   key,
		Sig:   sig,
	}
	//return cl.Decrypt(dest, dr)
	return cl.Decrypt(r, dr)
}

func (byzd *ByzcoinData) AddReadTransaction(proof *byzcoin.Proof, signer darc.Signer, darc darc.Darc, wait int) (*TransactionReply, error) {
	read := &calypso.Read{
		Write: byzcoin.NewInstanceID(proof.InclusionProof.Key),
		Xc:    signer.Ed25519.Point,
	}
	readBuf, err := protobuf.Encode(read)
	if err != nil {
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
		return nil, err
	}
	reply := &TransactionReply{}
	reply.InstanceID = ctx.Instructions[0].DeriveID("")
	reply.AddTxResponse, err = byzd.Cl.AddTransactionAndWait(ctx, wait)
	if err != nil {
		return nil, err
	}
	return reply, nil
}

func (byzd *ByzcoinData) WaitProof(id byzcoin.InstanceID, interval time.Duration, value []byte) (*byzcoin.Proof, error) {
	return byzd.Cl.WaitProof(id, interval, value)
}

func (byzd *ByzcoinData) AddWriteTransaction(wd *WriteData, signer darc.Signer, darc darc.Darc, wait int) (*TransactionReply, error) {
	//func (byzd *ByzcoinData) AddWriteTransaction(write *calypso.SimpleWrite, signer darc.Signer, darc darc.Darc, wait int) (*TransactionReply, error) {
	sWrite := &calypso.SimpleWrite{
		DataHash: wd.DataHash,
		K:        wd.K,
		C:        wd.C,
		Reader:   wd.Reader,
	}
	writeBuf, err := protobuf.Encode(sWrite)
	//writeBuf, err := protobuf.Encode(write)
	if err != nil {
		return nil, err
	}
	ctx := byzcoin.ClientTransaction{
		Instructions: byzcoin.Instructions{{
			InstanceID: byzcoin.NewInstanceID(darc.GetBaseID()),
			Nonce:      byzcoin.Nonce{},
			Index:      0,
			Length:     1,
			Spawn: &byzcoin.Spawn{
				ContractID: calypso.ContractSimpleWriteID,
				Args: byzcoin.Arguments{{
					Name: "write", Value: writeBuf}},
			},
		}},
	}
	//Sign the transaction
	err = ctx.Instructions[0].SignBy(darc.GetID(), signer)
	if err != nil {
		return nil, err
	}
	reply := &TransactionReply{}
	reply.InstanceID = ctx.Instructions[0].DeriveID("")
	//Delegate the work to the byzcoin client
	reply.AddTxResponse, err = byzd.Cl.AddTransactionAndWait(ctx, wait)
	if err != nil {
		return nil, err
	}
	return reply, err
}

func (byzd *ByzcoinData) SpawnDarc(spawnDarc darc.Darc, wait int) (*byzcoin.AddTxResponse, error) {
	//func (byzd *ByzcoinData) SpawnDarc(signer darc.Signer, controlDarc darc.Darc, spawnDarc darc.Darc, wait int) (*byzcoin.AddTxResponse, error) {
	darcBuf, err := spawnDarc.ToProto()
	if err != nil {
		return nil, err
	}
	ctx := byzcoin.ClientTransaction{
		Instructions: []byzcoin.Instruction{{
			InstanceID: byzcoin.NewInstanceID(byzd.GDarc.GetBaseID()),
			//InstanceID: byzcoin.NewInstanceID(controlDarc.GetBaseID()),
			Nonce:  byzcoin.GenNonce(),
			Index:  0,
			Length: 1,
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
	//err = ctx.Instructions[0].SignBy(controlDarc.GetBaseID(), signer)
	if err != nil {
		return nil, err
	}
	return byzd.Cl.AddTransactionAndWait(ctx, wait)
}

func StoreEncryptedData(r *onet.Roster, wd *WriteData) error {
	//func StoreEncryptedData(dest *network.ServerIdentity, wd *WriteData) error {
	//func StoreEncryptedData(r *onet.Roster, data []byte, digest []byte) (string, error) {
	cl := simple.NewClient()
	defer cl.Close()
	//sr := simple.StoreRequest{
	//Data:   data,
	//Digest: digest,
	//}
	sr := simple.StoreRequest{
		Data:     wd.Data,
		DataHash: wd.DataHash,
	}
	//reply, err := cl.StoreData(dest, &sr)
	reply, err := cl.StoreData(r, &sr)
	if err != nil {
		return err
	}
	wd.StoredKey = reply.StoredKey
	return nil
}

func SetupByzcoin(r *onet.Roster) (*ByzcoinData, error) {
	var err error
	byzd := &ByzcoinData{}
	byzd.Signer = darc.NewSignerEd25519(nil, nil)
	byzd.GMsg, err = byzcoin.DefaultGenesisMsg(byzcoin.CurrentVersion, r, []string{"spawn:" + byzcoin.ContractDarcID, "spawn:" + calypso.ContractSimpleWriteID, "spawn:" + calypso.ContractReadID}, byzd.Signer.Identity())
	if err != nil {
		return nil, err
	}
	byzd.GMsg.BlockInterval = 100 * time.Millisecond
	byzd.GDarc = &byzd.GMsg.GenesisDarc
	byzd.Cl, _, err = byzcoin.NewLedger(byzd.GMsg, false)
	if err != nil {
		return nil, err
	}
	return byzd, nil
}
